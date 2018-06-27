package main

import (
	"fmt"
)

// MarshalBinary converts palette entries to byte slice.
func (v Palette) MarshalBinary() (data []byte, err error) {
	data = make([]byte, 3*len(v))
	for i := 0; i < len(v); i++ {
		data[i*3] = v[i].r
		data[i*3+1] = v[i].g
		data[i*3+2] = v[i].b
	}
	return data, nil
}

// UnmarshalBinary converts byte slice to palette entries.
func (v Palette) UnmarshalBinary(data []byte) error {
	if len(v)*3 != len(data) {
		return fmt.Errorf("Len is not valid. required: %d, actual: %d", len(v)*3, len(data))
	}
	for i := 0; i < len(v); i++ {
		v[i].r = data[i*3]
		v[i].g = data[i*3+1]
		v[i].b = data[i*3+2]
	}
	return nil
}

// Rgb holds pixel data.
type Rgb struct {
	r byte
	g byte
	b byte
}

// Palette holds palette data.
type Palette []Rgb

// ImageFrame holds picture data.
type ImageFrame struct {
	palette Palette
	data    []byte
}

// ImageData holds picture frames.
type ImageData struct {
	width   int
	height  int
	palette Palette
	frames  []ImageFrame
}
