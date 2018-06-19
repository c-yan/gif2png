package main

// ToByteSlice converts palette entry to byte slice.
func (v Rgb) ToByteSlice(p []byte) {
	p[0] = v.r
	p[1] = v.g
	p[2] = v.b
}

// Rgb holds pixel data.
type Rgb struct {
	r byte
	g byte
	b byte
}

// ImageData holds picture data.
type ImageData struct {
	width   int
	height  int
	palette []Rgb
	data    []byte
}
