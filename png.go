package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"io"
)

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

func writeIHDR(w io.Writer, data ImageData) {
	b := make([]byte, 8, 13)
	binary.BigEndian.PutUint32(b[0:4], uint32(data.width))
	binary.BigEndian.PutUint32(b[4:8], uint32(data.height))
	b = append(b, []byte{8, 3, 0, 0, 0}...)
	writeChunk(w, "IHDR", b)
}

func writePLTE(w io.Writer, data ImageData) {
	t := make([]byte, 3)
	b := make([]byte, 0, 768)
	for _, e := range data.palette {
		e.ToByteSlice(t)
		b = append(b, t...)
	}
	writeChunk(w, "PLTE", b)
}

func serialize(data ImageData) []byte {
	b := make([]byte, 0, (data.width+1)*data.height)
	for i := 0; i < data.height; i++ {
		b = append(b, 0)
		b = append(b, data.data[data.width*i:data.width*(i+1)]...)
	}
	return b
}

func writeIDAT(w io.Writer, data ImageData) {
	buf := &bytes.Buffer{}
	zw, err := zlib.NewWriterLevel(buf, zlib.BestCompression)
	if err != nil {
		panic(err)
	}
	defer zw.Close()
	zw.Write(serialize(data))
	zw.Flush()
	writeChunk(w, "IDAT", buf.Bytes())
}

// WritePng writes the image data to writer in PNG format.
func WritePng(w io.Writer, data ImageData) {
	writePngSignature(w)
	writeIHDR(w, data)
	writePLTE(w, data)
	writeIDAT(w, data)
	writeChunk(w, "IEND", make([]byte, 0))
}
