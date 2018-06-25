package main

import (
	"compress/lzw"
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

type imageDescriptor struct {
	ImageSeparator        byte
	ImageLeftPosition     uint16
	ImageTopPosition      uint16
	ImageWidth            uint16
	ImageHeight           uint16
	LocalColorTableFlag   bool
	InterlaceFlag         bool
	SortFlag              bool
	SizeOfLocalColorTable uint
	LocalColorTable       []byte
}

type blockReader struct {
	buf     [255]byte
	bufLen  int
	bufNext int
	r       io.Reader
}

func newBlockReader(r io.Reader) *blockReader {
	return &blockReader{
		r:       r,
		bufLen:  0,
		bufNext: 0,
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func readByte(r io.Reader) (byte, error) {
	var buf [1]byte
	n, err := r.Read(buf[:])
	if n == 0 {
		return 0, io.ErrUnexpectedEOF
	}
	return buf[0], err
}

func (v *blockReader) readNextBlock() error {
	blockSize, err := readByte(v.r)
	if err != nil {
		return err
	}
	if blockSize == 0 {
		return io.EOF
	}
	if _, err = io.ReadFull(v.r, v.buf[:blockSize]); err != nil {
		return err
	}
	v.bufLen = int(blockSize)
	v.bufNext = 0
	return nil
}

func (v *blockReader) Read(p []byte) (n int, err error) {
	if v.bufNext >= v.bufLen {
		err = v.readNextBlock()
		if err == io.EOF {
			return 0, io.ErrUnexpectedEOF
		}
		if err != nil {
			return 0, err
		}
	}
	n = min(len(p), v.bufLen-v.bufNext)
	for i := 0; i < n; i++ {
		p[i] = v.buf[i+v.bufNext]
	}
	v.bufNext += n
	return
}

const (
	headerSize                  = 6
	logicalScreenDescriptorSize = 7
	imageDescriptorSize         = 10
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

func (v *imageDescriptor) UnmarshalBinary(data []byte) error {
	if len(data) < imageDescriptorSize {
		return fmt.Errorf("Len is not enough. required: %d, actual: %d", imageDescriptorSize, len(data))
	}
	v.ImageSeparator = data[0]
	v.ImageLeftPosition = binary.LittleEndian.Uint16(data[1:])
	v.ImageTopPosition = binary.LittleEndian.Uint16(data[3:])
	v.ImageWidth = binary.LittleEndian.Uint16(data[5:])
	v.ImageHeight = binary.LittleEndian.Uint16(data[7:])
	v.LocalColorTableFlag = ((data[9] & 0x80) >> 7) == 1
	v.InterlaceFlag = ((data[9] & 0x40) >> 6) == 1
	v.SortFlag = ((data[9] & 0x20) >> 5) == 1
	v.SizeOfLocalColorTable = uint(math.Pow(2, float64(data[9]&0x7+1)))
	return nil
}

func readHeadser(r io.Reader) (*header, error) {
	var (
		h   header
		buf [headerSize]byte
	)

	_, err := io.ReadFull(r, buf[:])
	if err != nil {
		return nil, err
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
	)

	_, err := io.ReadFull(r, buf[:])
	if err != nil {
		return nil, err
	}
	l.UnmarshalBinary(buf[:])

	if l.GlobalColorTableFlag {
		l.GlobalColorTable = make([]byte, l.SizeOfGlobalColorTable*3)
		_, err = io.ReadFull(r, l.GlobalColorTable)
		if err != nil {
			return nil, err
		}
	}

	return &l, nil
}

func readImageDescriptor(r io.Reader) (*imageDescriptor, error) {
	var (
		i   imageDescriptor
		buf [imageDescriptorSize]byte
	)

	_, err := io.ReadFull(r, buf[:])
	if err != nil {
		return nil, err
	}
	i.UnmarshalBinary(buf[:])

	if i.LocalColorTableFlag {
		i.LocalColorTable = make([]byte, i.SizeOfLocalColorTable*3)
		_, err = io.ReadFull(r, i.LocalColorTable)
		if err != nil {
			return nil, err
		}
	}

	return &i, nil
}

func readTableBasedImageData(r io.Reader, width int, height int) (*ImageData, error) {
	var (
		data     ImageData
		err      error
		litWidth byte
	)
	data.width = width
	data.height = height
	data.data = make([]byte, width*height)
	litWidth, err = readByte(r)
	if err != nil {
		return nil, err
	}
	lr := lzw.NewReader(newBlockReader(r), lzw.LSB, int(litWidth))
	defer lr.Close()
	_, err = io.ReadFull(lr, data.data)
	if err != nil {
		return nil, err
	}
	_, err = readByte(r)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// ReadGif reads the image data from reader as GIF format.
func ReadGif(r io.Reader) (*ImageData, error) {
	var (
		err               error
		data              *ImageData
		h                 *header
		l                 *logicalScreenDescriptor
		i                 *imageDescriptor
		errNotImplemented = errors.New("Not implemented")
	)

	h, err = readHeadser(r)
	if err != nil {
		return nil, err
	}
	if h.Version != "87a" {
		return nil, errNotImplemented
	}
	l, err = readLogicalScreenDescriptor(r)
	if err != nil {
		return nil, err
	}
	if l.ColorResolution != 8 {
		return nil, errNotImplemented
	}
	i, err = readImageDescriptor(r)
	if err != nil {
		return nil, err
	}

	data, err = readTableBasedImageData(r, int(i.ImageWidth), int(i.ImageHeight))
	if err != nil {
		return nil, err
	}

	if l.GlobalColorTableFlag {
		data.palette = make([]Rgb, l.SizeOfGlobalColorTable)
		data.palette.UnmarshalBinary(l.GlobalColorTable)
	}
	if i.LocalColorTableFlag {
		data.palette = make([]Rgb, i.SizeOfLocalColorTable)
		data.palette.UnmarshalBinary(i.LocalColorTable)
	}

	return data, nil
}
