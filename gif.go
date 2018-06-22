package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

type header struct {
	Signature string
	Version   string
}

type logicalScreenDescriptor struct {
	LogicalScreenWidth     uint16
	LogicalScreenHeight    uint16
	GlobalColorTableFlag   bool
	ColorResolution        byte
	SortFlag               bool
	SizeOfGlobalColorTable uint
	BackgroundColorIndex   byte
	PixelAspectRatio       byte
	GlobalColorTable       []byte
}

const (
	headerSize                  = 6
	logicalScreenDescriptorSize = 7
)

func (v *header) UnmarshalBinary(data []byte) error {
	if len(data) < headerSize {
		return fmt.Errorf("Len is not enough. required: %d, actual: %d", headerSize, len(data))
	}
	v.Signature = string(data[:3])
	v.Version = string(data[3:6])
	return nil
}

func (v *logicalScreenDescriptor) UnmarshalBinary(data []byte) error {
	if len(data) < logicalScreenDescriptorSize {
		return fmt.Errorf("Len is not enough. required: %d, actual: %d", logicalScreenDescriptorSize, len(data))
	}
	v.LogicalScreenWidth = binary.LittleEndian.Uint16(data[:])
	v.LogicalScreenHeight = binary.LittleEndian.Uint16(data[2:])
	v.GlobalColorTableFlag = ((data[4] & 0x80) >> 7) == 1
	v.ColorResolution = ((data[4] & 0x70) >> 4) + 1
	v.SortFlag = ((data[4] & 0x8) >> 3) == 1
	v.SizeOfGlobalColorTable = uint(math.Pow(2, float64(data[4]&0x7+1)))
	v.BackgroundColorIndex = data[5]
	v.PixelAspectRatio = data[6]
	return nil
}

func readHeadser(r io.Reader) (*header, error) {
	var (
		h   header
		buf [headerSize]byte
	)

	n, err := r.Read(buf[:])
	if err != nil {
		return nil, err
	}
	if n != headerSize {
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

func readLogicalScreenDescriptor(r io.Reader) (*logicalScreenDescriptor, error) {
	var (
		l   logicalScreenDescriptor
		buf [logicalScreenDescriptorSize]byte
		n   int
		err error
	)

	n, err = r.Read(buf[:])
	if err != nil {
		return nil, err
	}
	if n != logicalScreenDescriptorSize {
		return nil, errors.New("Unexpeced EoF")
	}
	l.UnmarshalBinary(buf[:])

	if l.GlobalColorTableFlag {
		l.GlobalColorTable = make([]byte, l.SizeOfGlobalColorTable*3)
		n, err = r.Read(l.GlobalColorTable)
		if err != nil {
			return nil, err
		}
		if n != int(l.SizeOfGlobalColorTable*3) {
			return nil, errors.New("Unexpeced EoF")
		}
	}

	return &l, nil
}

// ReadGif reads the image data from reader as GIF format.
func ReadGif(r io.Reader) (*ImageData, error) {
	var (
		err  error
		data ImageData
		h    *header
		l    *logicalScreenDescriptor
	)

	h, err = readHeadser(r)
	if err != nil {
		return nil, err
	}
	if h.Version != "87a" {
		return nil, errors.New("Not implemented")
	}
	l, err = readLogicalScreenDescriptor(r)
	if err != nil {
		return nil, err
	}

	data.width = int(l.LogicalScreenWidth)
	data.height = int(l.LogicalScreenHeight)
	if l.GlobalColorTableFlag {
		data.palette = make([]Rgb, l.SizeOfGlobalColorTable)
		data.palette.UnmarshalBinary(l.GlobalColorTable)
	}
	data.data = make([]byte, data.width*data.height)
	return &data, nil
}
