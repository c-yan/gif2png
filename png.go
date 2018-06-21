package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"io"
	"log"
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

func writePngSignature(w io.Writer) {
	w.Write([]byte{137, 80, 78, 71, 13, 10, 26, 10})
}

func writeChunk(w io.Writer, chunkType string, data []byte) {
	ctb := []byte(chunkType)
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(len(data)))
	w.Write(b)
	w.Write(ctb)
	w.Write(data)
	binary.BigEndian.PutUint32(b, crc32.Update(crc32.ChecksumIEEE(ctb), crc32.IEEETable, data))
	w.Write(b)
}

func writeIHDR(w io.Writer, data *ImageData) {
	b, _ := imageHeader{
		Width:             uint32(data.width),
		Height:            uint32(data.height),
		BitDepth:          8,
		ColorType:         paletteUsed | trueColorUsed,
		CompressionMethod: deflateCompression,
		FilterMethod:      noneFilter,
		InterlaceMethod:   noInterlace,
	}.MarshalBinary()
	writeChunk(w, "IHDR", b)
}

func writePLTE(w io.Writer, data *ImageData) {
	b, _ := data.palette.MarshalBinary()
	writeChunk(w, "PLTE", b)
}

func serialize(data *ImageData) []byte {
	b := make([]byte, 0, (data.width+1)*data.height)
	for i := 0; i < data.height; i++ {
		b = append(b, 0)
		b = append(b, data.data[data.width*i:data.width*(i+1)]...)
	}
	return b
}

func writeIDAT(w io.Writer, data *ImageData) {
	buf := &bytes.Buffer{}
	zw, err := zlib.NewWriterLevel(buf, zlib.BestCompression)
	if err != nil {
		log.Fatal(err)
	}
	defer zw.Close()
	zw.Write(serialize(data))
	zw.Flush()
	writeChunk(w, "IDAT", buf.Bytes())
}

// WritePng writes the image data to writer in PNG format.
func WritePng(w io.Writer, data *ImageData) error {
	writePngSignature(w)
	writeIHDR(w, data)
	writePLTE(w, data)
	writeIDAT(w, data)
	writeChunk(w, "IEND", nil)
	return nil
}
