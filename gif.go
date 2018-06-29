package main

import (
	"compress/lzw"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
)

var errNotImplemented = errors.New("Not implemented")

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

type graphicControlExtension struct {
	DisposalMethod        int
	UserInputFlag         bool
	TransparentColorFlag  bool
	DelayTime             uint16
	TransparentColorIndex byte
}

func (v *header) String() string {
	return fmt.Sprintf("%s%s", v.Signature, v.Version)
}

func (v *logicalScreenDescriptor) String() string {
	return fmt.Sprintf(`
		LogicalScreenWidth: %d
		LogicalScreenHeight: %d
		GlobalColorTableFlag: %v
		ColorResolution: %d
		SortFlag: %v
		SizeOfGlobalColorTable: %d
		BackgroundColorIndex: %d
		PixelAspectRatio: %d`,
		v.LogicalScreenWidth,
		v.LogicalScreenHeight,
		v.GlobalColorTableFlag,
		v.ColorResolution, v.SortFlag,
		v.SizeOfGlobalColorTable,
		v.BackgroundColorIndex,
		v.PixelAspectRatio)
}

func (v *imageDescriptor) String() string {
	return fmt.Sprintf(`
		ImageSeparator: %x
		ImageLeftPosition: %d
		ImageTopPosition: %d
		ImageWidth: %d
		ImageHeight: %d
		LocalColorTableFlag: %v
		InterlaceFlag: %v
		SortFlag: %v
		SizeOfLocalColorTable: %d`,
		v.ImageSeparator,
		v.ImageLeftPosition,
		v.ImageTopPosition,
		v.ImageWidth,
		v.ImageHeight,
		v.LocalColorTableFlag,
		v.InterlaceFlag,
		v.SortFlag,
		v.SizeOfLocalColorTable)
}

func (v *graphicControlExtension) String() string {
	return fmt.Sprintf(`
		DisposalMethod: %d
		UserInputFlag: %v
		TransparentColorFlag: %v
		DelayTime: %d
		TransparentColorIndex: %d`,
		v.DisposalMethod,
		v.UserInputFlag,
		v.TransparentColorFlag,
		v.DelayTime,
		v.TransparentColorIndex)
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

type peekReader struct {
	buf   [2]byte
	index int
	r     io.Reader
}

func newPeekReader(r io.Reader) *peekReader {
	return &peekReader{
		r:     r,
		index: 0,
	}
}

func (v *peekReader) Read(p []byte) (n int, err error) {
	if len(p) < v.index {
		return 0, errNotImplemented
	}
	for i := 0; i < v.index; i++ {
		p[i] = v.buf[i]
	}
	n, err = v.r.Read(p[v.index:])
	n += v.index
	v.index = 0
	return n, err
}

func (v *peekReader) Peek() (byte, error) {
	b, err := readByte(v.r)
	if err != nil {
		return 0, nil
	}
	v.buf[v.index] = b
	v.index++
	return b, nil
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
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	}
	if err != nil {
		return err
	}
	if blockSize == 0 {
		return io.EOF
	}
	_, err = io.ReadFull(v.r, v.buf[:blockSize])
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	}
	if err != nil {
		return err
	}
	v.bufLen = int(blockSize)
	v.bufNext = 0
	return nil
}

func (v *blockReader) Read(p []byte) (n int, err error) {
	if v.bufNext >= v.bufLen {
		err = v.readNextBlock()
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
	graphicControlExtensionSize = 4
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
	if v.GlobalColorTableFlag {
		v.SizeOfGlobalColorTable = uint(math.Pow(2, float64(data[4]&0x7+1)))
	} else {
		v.SizeOfGlobalColorTable = 0
	}
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
	if v.LocalColorTableFlag {
		v.SizeOfLocalColorTable = uint(math.Pow(2, float64(data[9]&0x7+1)))
	} else {
		v.SizeOfLocalColorTable = 0
	}
	return nil
}

func (v *graphicControlExtension) UnmarshalBinary(data []byte) error {
	if len(data) < graphicControlExtensionSize {
		return fmt.Errorf("Len is not enough. required: %d, actual: %d", graphicControlExtensionSize, len(data))
	}
	v.DisposalMethod = int(data[0]&0x1c) >> 2
	v.UserInputFlag = data[0]&2>>1 == 1
	v.TransparentColorFlag = data[0]&1 == 1
	v.DelayTime = binary.LittleEndian.Uint16(data[1:])
	v.TransparentColorIndex = data[3]
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

func skipBlock(r io.Reader) error {
	var buf [255]byte
	_, err := r.Read(buf[:2])
	if err != nil {
		return err
	}
	br := newBlockReader(r)
	for {
		_, err := br.Read(buf[:])
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
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

func readTableBasedImageData(r io.Reader, width int, height int) (*ImageFrame, error) {
	var (
		err      error
		litWidth byte
		frame    ImageFrame
	)
	frame.data = make([]byte, width*height)
	litWidth, err = readByte(r)
	if err != nil {
		return nil, err
	}
	lr := lzw.NewReader(newBlockReader(r), lzw.LSB, int(litWidth))
	defer lr.Close()
	_, err = io.ReadFull(lr, frame.data)
	if err != nil {
		return nil, err
	}
	_, err = readByte(r)
	if err != nil {
		return nil, err
	}
	return &frame, nil
}

func readGraphicControlExtension(r io.Reader) (*graphicControlExtension, error) {
	var (
		g   graphicControlExtension
		buf [graphicControlExtensionSize]byte
	)
	_, err := io.ReadFull(r, buf[:2])
	if err != nil {
		return nil, err
	}
	_, err = io.ReadFull(newBlockReader(r), buf[:])
	if err != nil {
		return nil, err
	}
	g.UnmarshalBinary(buf[:])
	_, err = readByte(r)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func readTrailer(r io.Reader) error {
	b, err := readByte(r)
	if err != nil {
		return err
	}
	if b != 0x3b {
		return fmt.Errorf("This block is not trailer, code=0x%02x", b)
	}
	return nil
}

// ReadGif reads the image data from reader as GIF format.
func ReadGif(r io.Reader, verbose bool) (*ImageData, error) {
	var data ImageData

	h, err := readHeadser(r)
	if err != nil {
		return nil, err
	}
	if verbose {
		log.Printf("GIF Header: %s\n", h)
	}

	l, err := readLogicalScreenDescriptor(r)
	if err != nil {
		return nil, err
	}
	if l.ColorResolution != 8 {
		return nil, errNotImplemented
	}
	if verbose {
		log.Printf("Logical Screen Descriptor: %s\n", l)
	}
	if l.GlobalColorTableFlag {
		data.palette = make([]Rgb, l.SizeOfGlobalColorTable)
		data.palette.UnmarshalBinary(l.GlobalColorTable)
	}

	pr := newPeekReader(r)
	for {
		b, err := pr.Peek()
		if err != nil {
			return nil, err
		}

		switch b {
		case 0x2C:
			i, err := readImageDescriptor(pr)
			if err != nil {
				return nil, err
			}
			if verbose {
				log.Printf("Image Descriptor: %s\n", i)
			}

			data.width = int(i.ImageWidth)
			data.height = int(i.ImageHeight)

			frame, err := readTableBasedImageData(pr, int(i.ImageWidth), int(i.ImageHeight))
			if err != nil {
				return nil, err
			}

			if i.LocalColorTableFlag {
				frame.palette = make([]Rgb, i.SizeOfLocalColorTable)
				frame.palette.UnmarshalBinary(i.LocalColorTable)
			}

			data.frames = append(data.frames, *frame)
		case 0x21:
			b, err := pr.Peek()
			if err != nil {
				return nil, err
			}

			switch b {
			case 0xF9:
				//Graphic Control Extension
				g, err := readGraphicControlExtension(pr)
				if err != nil {
					return nil, err
				}
				if verbose {
					log.Printf("Graphic Control Extension: %s\n", g)
				}
			case 0xFE:
				//Comment Extension
				if verbose {
					log.Println("Skip Comment Extension")
				}
				err := skipBlock(pr)
				if err != nil {
					return nil, err
				}
			case 0x01:
				//Plain Text Extension
				if verbose {
					log.Println("Skip Plain Text Extension")
				}
				err := skipBlock(pr)
				if err != nil {
					return nil, err
				}
			case 0xFF:
				//Application Extension
				if verbose {
					log.Println("Skip Application Extension")
				}
				err := skipBlock(pr)
				if err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("Unknown code: 0x21%02x", b)
			}
		case 0x3b:
			err := readTrailer(pr)
			if err != nil {
				return nil, err
			}
			return &data, nil
		default:
			return nil, fmt.Errorf("Unknown code: 0x%02x", b)
		}
	}
}
