package main

import (
	"os"
)

func main() {
	var data ImageData
	data.width = 320
	data.height = 320
	data.palette = make([]Rgb, 256)
	data.palette[0].b = 255
	data.data = make([]byte, data.width*data.height)
	file, err := os.Create("test.png")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	WritePng(file, data)
}
