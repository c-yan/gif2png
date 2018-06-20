package main

import (
	"fmt"
)

// UnmarshalBinary converts palette entry to byte slice.
func (v Rgb) UnmarshalBinary(data []byte) error {
	if cap(data) < 3 {
		return fmt.Errorf("Capacity is not enough. required: %d, actual: %d", 3, cap(data))
	}
	data[0] = v.r
	data[1] = v.g
	data[2] = v.b
	return nil
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
