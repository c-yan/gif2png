package main

import (
	"log"
	"os"
)

func readFile(path string) (*ImageData, error) {
	in, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer in.Close()
	return ReadGif(in)
}

func writeFile(path string, data *ImageData) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	return WritePng(out, data)
}

func main() {
	data, err := readFile("test.gif")
	if err != nil {
		log.Fatal(err)
	}
	err = writeFile("test.png", data)
	if err != nil {
		log.Fatal(err)
	}
}
