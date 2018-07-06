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
	return ReadGif(in, true)
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
	var src string
	if len(os.Args) > 1 {
		src = os.Args[1]
	} else {
		src = "test.gif"
	}
	data, err := readFile(src)
	if err != nil {
		log.Fatal(err)
	}
	err = writeFile("test.png", data)
	if err != nil {
		log.Fatal(err)
	}
}
