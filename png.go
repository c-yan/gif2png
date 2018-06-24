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
		Width:             uint32(data.width),
		Height:            uint32(data.height),
		BitDepth:          8,
		ColorType:         paletteUsed | trueColorUsed,
		CompressionMethod: deflateCompression,
		FilterMethod:      noneFilter,
		InterlaceMethod:   noInterlace,
	}.MarshalBinary()
	return writeChunk(w, "IHDR", b)
}

func writePLTE(w io.Writer, data *ImageData) error {
	b, _ := data.palette.MarshalBinary()
	return writeChunk(w, "PLTE", b)
}

func serialize(data *ImageData) []byte {
	b := make([]byte, 0, (data.width+1)*data.height)
	for i := 0; i < data.height; i++ {
		b = append(b, 0)
		b = append(b, data.data[data.width*i:data.width*(i+1)]...)
	}
	return b
}

func writeIDAT(w io.Writer, data *ImageData) error {
	buf := &bytes.Buffer{}
	zw, err := zlib.NewWriterLevel(buf, zlib.BestCompression)
	if err != nil {
		return err
	}
	defer zw.Close()
	if _, err := zw.Write(serialize(data)); err != nil {
		return err
	}
	if err := zw.Flush(); err != nil {
		return err
	}
	if err := writeChunk(w, "IDAT", buf.Bytes()); err != nil {
		return err
	}
	return nil
}

func writeIEND(w io.Writer) error {
	return writeChunk(w, "IEND", nil)
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
	if err := writeIDAT(w, data); err != nil {
		return err
	}
	if err := writeIEND(w); err != nil {
		return err
	}
	return nil
}
