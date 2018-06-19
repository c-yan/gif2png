package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"io"
)

func toByteSlice(v uint32) []byte {
	result := make([]byte, 4)
	binary.BigEndian.PutUint32(result, v)
	return result
}

func writePngSignature(w io.Writer) {
	w.Write([]byte{137, 80, 78, 71, 13, 10, 26, 10})
}

func writeChunk(w io.Writer, chunkType string, data []byte) {
	ctb := []byte(chunkType)
	w.Write(toByteSlice(uint32(len(data))))
	w.Write(ctb)
	w.Write(data)
	w.Write(toByteSlice(crc32.Update(crc32.ChecksumIEEE(ctb), crc32.IEEETable, data)))
}

func writeIHDR(w io.Writer, data ImageData) {
	b := make([]byte, 0, 13)
	b = append(b, toByteSlice(uint32(data.width))...)
	b = append(b, toByteSlice(uint32(data.height))...)
	b = append(b, []byte{8, 3, 0, 0, 0}...)
	writeChunk(w, "IHDR", b)
}

func writePLTE(w io.Writer, data ImageData) {
	b := make([]byte, 0, 768)
	for _, e := range data.palette {
		b = append(b, e.ToByteSlice()...)
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

func WritePng(w io.Writer, data ImageData) {
	writePngSignature(w)
	writeIHDR(w, data)
	writePLTE(w, data)
	writeIDAT(w, data)
	writeChunk(w, "IEND", make([]byte, 0))
}
