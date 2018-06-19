package main

func (v Rgb) ToByteSlice() []byte {
	result := make([]byte, 3)
	result[0] = v.r
	result[1] = v.g
	result[2] = v.b
	return result
}

type Rgb struct {
	r byte
	g byte
	b byte
}

type ImageData struct {
	width   int
	height  int
	palette []Rgb
	data    []byte
}
