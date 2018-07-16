package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"io"
)

const (
	paletteUsed   = 1
	trueColorUsed = 2
	alphaUsed     = 4
)

const (
	deflateCompression = iota
)

const (
	noneFilter = iota
	subFilter
	upFilter
	averageFilter
	paethFilter
)

const (
	noInterlace = iota
	adam7Interlace
)

type imageHeader struct {
	Width             uint32
	Height            uint32
	BitDepth          byte
	ColorType         byte
	CompressionMethod byte
	FilterMethod      byte
	InterlaceMethod   byte
}

type frameControl struct {
	SequenceNumber uint32
	Width          uint32
	Height         uint32
	XOffset        uint32
	YOffset        uint32
	DelayNum       uint16
	DelayDen       uint16
	DisposeOp      byte
	BlendOp        byte
}

func (v imageHeader) MarshalBinary() (data []byte, err error) {
	data = make([]byte, 13)
	binary.BigEndian.PutUint32(data[0:4], v.Width)
	binary.BigEndian.PutUint32(data[4:8], v.Height)
	data[8] = v.BitDepth
	data[9] = v.ColorType
	data[10] = v.CompressionMethod
	data[11] = v.FilterMethod
	data[12] = v.InterlaceMethod
	return data, nil
}

func (v frameControl) MarshalBinary() (data []byte, err error) {
	data = make([]byte, 26)

	binary.BigEndian.PutUint32(data[:4], v.SequenceNumber)
	binary.BigEndian.PutUint32(data[4:8], v.Width)
	binary.BigEndian.PutUint32(data[8:12], v.Height)
	binary.BigEndian.PutUint32(data[12:16], v.XOffset)
	binary.BigEndian.PutUint32(data[16:20], v.YOffset)
	binary.BigEndian.PutUint16(data[20:22], v.DelayNum)
	binary.BigEndian.PutUint16(data[22:24], v.DelayDen)
	data[24] = v.DisposeOp
	data[25] = v.BlendOp
	return data, nil
}

func writePngSignature(w io.Writer) error {
	_, err := w.Write([]byte{137, 80, 78, 71, 13, 10, 26, 10})
	return err
}

func writeChunk(w io.Writer, chunkType string, data []byte) error {
	ctb := []byte(chunkType)
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(len(data)))
	if _, err := w.Write(b); err != nil {
		return err
	}
	if _, err := w.Write(ctb); err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	binary.BigEndian.PutUint32(b, crc32.Update(crc32.ChecksumIEEE(ctb), crc32.IEEETable, data))
	if _, err := w.Write(b); err != nil {
		return err
	}
	return nil
}

func writeIHDR(w io.Writer, data *ImageData) error {
	b, _ := imageHeader{
		Width:             uint32(data.frames[0].width),
		Height:            uint32(data.frames[0].height),
		BitDepth:          8,
		ColorType:         paletteUsed | trueColorUsed,
		CompressionMethod: deflateCompression,
		FilterMethod:      noneFilter,
		InterlaceMethod:   noInterlace,
	}.MarshalBinary()
	return writeChunk(w, "IHDR", b)
}

func writePLTE(w io.Writer, data *ImageData) error {
	var b []byte
	b, _ = data.palette.MarshalBinary()
	return writeChunk(w, "PLTE", b)
}

func writeTRNS(w io.Writer, entries int, transparencyIndex int) error {
	var b [256]byte
	for i := range b {
		b[i] = 255
	}
	b[transparencyIndex] = 0
	return writeChunk(w, "tRNS", b[:entries])
}

func serialize(frame *ImageFrame) []byte {
	b := make([]byte, 0, (frame.width+1)*frame.height)
	for i := 0; i < frame.height; i++ {
		b = append(b, 0)
		b = append(b, frame.data[frame.width*i:frame.width*(i+1)]...)
	}
	return b
}

func writeACTL(w io.Writer, data *ImageData) error {
	var buf [8]byte

	binary.BigEndian.PutUint32(buf[:4], uint32(len(data.frames)))
	binary.BigEndian.PutUint32(buf[4:], 0)
	if err := writeChunk(w, "acTL", buf[:]); err != nil {
		return err
	}
	return nil
}

func writeFCTL(w io.Writer, frame *ImageFrame, seq int) error {
	var f frameControl

	f.SequenceNumber = uint32(seq)
	f.Width = uint32(frame.width)
	f.Height = uint32(frame.height)
	f.XOffset = uint32(frame.xOffset)
	f.YOffset = uint32(frame.yOffset)
	f.DelayNum = uint16(frame.delay)
	f.DelayDen = 100
	f.DisposeOp = 0
	if frame.transparencyIndex == -1 {
		f.BlendOp = 0
	} else {
		f.BlendOp = 1
	}

	b, _ := f.MarshalBinary()
	if err := writeChunk(w, "fcTL", b); err != nil {
		return err
	}
	return nil
}

func writeData(w io.Writer, data []byte) error {
	zw, err := zlib.NewWriterLevel(w, zlib.BestCompression)
	if err != nil {
		return err
	}
	defer zw.Close()
	_, err = zw.Write(data)
	if err != nil {
		return err
	}
	err = zw.Flush()
	if err != nil {
		return err
	}
	return nil
}

func writeIDAT(w io.Writer, data *ImageData) error {
	buf := &bytes.Buffer{}
	err := writeData(buf, serialize(&data.frames[0]))
	if err != nil {
		return err
	}
	err = writeChunk(w, "IDAT", buf.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func writeFDAT(w io.Writer, frame *ImageFrame, seq int) error {
	var b [4]byte
	buf := &bytes.Buffer{}
	binary.BigEndian.PutUint32(b[:], uint32(seq))
	_, err := buf.Write(b[:])
	if err != nil {
		return err
	}
	err = writeData(buf, serialize(frame))
	if err != nil {
		return err
	}
	err = writeChunk(w, "fdAT", buf.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func writeIEND(w io.Writer) error {
	return writeChunk(w, "IEND", nil)
}

func writeAnimationPngData(w io.Writer, data *ImageData) error {
	if err := writeACTL(w, data); err != nil {
		return err
	}
	seq := 0
	if err := writeFCTL(w, &data.frames[0], seq); err != nil {
		return err
	}
	seq++
	if err := writeIDAT(w, data); err != nil {
		return err
	}
	for _, f := range data.frames[1:] {
		if err := writeFCTL(w, &f, seq); err != nil {
			return err
		}
		seq++
		if err := writeFDAT(w, &f, seq); err != nil {
			return err
		}
		seq++
	}
	if err := writeIEND(w); err != nil {
		return err
	}
	return nil
}

func writeNormalPngData(w io.Writer, data *ImageData) error {
	if err := writeIDAT(w, data); err != nil {
		return err
	}
	if err := writeIEND(w); err != nil {
		return err
	}
	return nil
}

// WritePng writes the image data to writer in PNG format.
func WritePng(w io.Writer, data *ImageData) error {
	if err := writePngSignature(w); err != nil {
		return err
	}
	if err := writeIHDR(w, data); err != nil {
		return err
	}
	if err := writePLTE(w, data); err != nil {
		return err
	}
	if data.transparencyIndex != -1 {
		if err := writeTRNS(w, len(data.palette), data.transparencyIndex); err != nil {
			return err
		}
	}
	if len(data.frames) > 1 {
		return writeAnimationPngData(w, data)
	}
	return writeNormalPngData(w, data)
}
