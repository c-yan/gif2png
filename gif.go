package main

import (
	"errors"
	"fmt"
	"io"
)

type header struct {
	Signature string
	Version   string
}

func (v *header) UnmarshalBinary(data []byte) error {
	const size = 6
	if len(data) < size {
		return fmt.Errorf("Len is not enough. required: %d, actual: %d", size, len(data))
	}
	v.Signature = string(data[:3])
	v.Version = string(data[3:6])
	return nil
}

func readHeadser(r io.Reader) (*header, error) {
	var (
		h   header
		buf [6]byte
	)

	n, err := r.Read(buf[:6])
	if err != nil {
		return nil, err
	}
	if n != 6 {
		return nil, errors.New("Unexpeced EoF")
	}
	h.UnmarshalBinary(buf[:])
	if h.Signature != "GIF" {
		return nil, fmt.Errorf("Unknown signature: %s", h.Signature)
	}

	knownVersions := make(map[string]struct{})
	knownVersions["87a"] = struct{}{}
	knownVersions["89a"] = struct{}{}

	if _, known := knownVersions[h.Version]; !known {
		return nil, fmt.Errorf("Unknown version: %s", h.Version)
	}

	return &h, nil
}

// ReadGif reads the image data from reader as GIF format.
func ReadGif(r io.Reader) (*ImageData, error) {
	var (
		err  error
		data ImageData
		h    *header
	)

	h, err = readHeadser(r)
	if err != nil {
		return nil, err
	}
	if h.Version != "87a" {
		return nil, errors.New("Not implemented")
	}

	data.width = 320
	data.height = 320
	data.palette = make([]Rgb, 256)
	data.palette[0].b = 255
	data.data = make([]byte, data.width*data.height)
	return &data, nil
}
