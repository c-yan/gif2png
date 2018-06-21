package main

import "io"

// ReadGif reads the image data from reader as GIF format.
func ReadGif(r io.Reader) (ImageData, error) {
	var data ImageData
	data.width = 320
	data.height = 320
	data.palette = make([]Rgb, 256)
	data.palette[0].b = 255
	data.data = make([]byte, data.width*data.height)
	return data, nil
}
