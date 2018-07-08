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

type applicationExtension struct {
	ApplicationIdentifier         [8]byte
	ApplicationAuthenticationCode [3]byte
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
		ImageLeftPosition: %d
		ImageTopPosition: %d
		ImageWidth: %d
		ImageHeight: %d
		LocalColorTableFlag: %v
		InterlaceFlag: %v
		SortFlag: %v
		SizeOfLocalColorTable: %d`,
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

func (v *applicationExtension) String() string {
	return fmt.Sprintf("%s %s", v.ApplicationIdentifier, v.ApplicationAuthenticationCode)
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
	imageDescriptorSize         = 9
	graphicControlExtensionSize = 4
	applicationExtensionSize    = 11
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
	v.GlobalColorTableFlag = data[4]>>7&1 == 1
	v.ColorResolution = data[4]>>4&7 + 1
	v.SortFlag = data[4]>>3&1 == 1
	if v.GlobalColorTableFlag {
		v.SizeOfGlobalColorTable = uint(math.Pow(2, float64(data[4]&7+1)))
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
	v.ImageLeftPosition = binary.LittleEndian.Uint16(data[0:])
	v.ImageTopPosition = binary.LittleEndian.Uint16(data[2:])
	v.ImageWidth = binary.LittleEndian.Uint16(data[4:])
	v.ImageHeight = binary.LittleEndian.Uint16(data[6:])
	v.LocalColorTableFlag = data[8]>>7&1 == 1
	v.InterlaceFlag = data[8]>>6&1 == 1
	v.SortFlag = data[8]>>5&1 == 1
	if v.LocalColorTableFlag {
		v.SizeOfLocalColorTable = uint(math.Pow(2, float64(data[8]&7+1)))
	} else {
		v.SizeOfLocalColorTable = 0
	}
	return nil
}

func (v *graphicControlExtension) UnmarshalBinary(data []byte) error {
	if len(data) < graphicControlExtensionSize {
		return fmt.Errorf("Len is not enough. required: %d, actual: %d", graphicControlExtensionSize, len(data))
	}
	v.DisposalMethod = int(data[0] >> 2 & 7)
	v.UserInputFlag = data[0]>>1&1 == 1
	v.TransparentColorFlag = data[0]&1 == 1
	v.DelayTime = binary.LittleEndian.Uint16(data[1:])
	v.TransparentColorIndex = data[3]
	return nil
}

func (v *applicationExtension) UnmarshalBinary(data []byte) error {
	if len(data) < applicationExtensionSize {
		return fmt.Errorf("Len is not enough. required: %d, actual: %d", applicationExtensionSize, len(data))
	}
	copy(v.ApplicationIdentifier[:], data[:8])
	copy(v.ApplicationAuthenticationCode[:], data[8:11])
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

func readBlock(r io.Reader, buf []byte) error {
	size, err := readByte(r)
	if err != nil {
		return err
	}
	if int(size) != len(buf) {
		return fmt.Errorf("buf size error: block size is %d, but buf size is %d", size, len(buf))
	}

	_, err = io.ReadFull(r, buf[:])
	if err != nil {
		return err
	}
	return nil
}

func skipBlock(r io.Reader) error {
	var buf [255]byte
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
	var frame ImageFrame

	frame.width = width
	frame.height = height

	frame.data = make([]byte, width*height)
	litWidth, err := readByte(r)
	if err != nil {
		return nil, err
	}
	lr := lzw.NewReader(newBlockReader(r), lzw.LSB, int(litWidth))
	defer lr.Close()
	_, err = io.ReadFull(lr, frame.data)
	if err != nil {
		return nil, err
	}

	err = skipBlock(r)
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
	_, err := io.ReadFull(newBlockReader(r), buf[:])
	if err != nil {
		return nil, err
	}
	g.UnmarshalBinary(buf[:])

	err = skipBlock(r)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func readApplicationExtension(r io.Reader) (*applicationExtension, error) {
	var (
		a   applicationExtension
		buf [applicationExtensionSize]byte
	)
	n, err := readByte(r)
	if err != nil {
		return nil, err
	}
	if n != applicationExtensionSize {
		return nil, fmt.Errorf("Unexpected block size. expected: %d, actual: %d", applicationExtensionSize, n)
	}

	_, err = io.ReadFull(r, buf[:])
	if err != nil {
		return nil, err
	}
	a.UnmarshalBinary(buf[:])

	err = skipBlock(r)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func deinterlace(frame *ImageFrame, width, height int) []byte {
	startingRow := [4]int{0, 4, 2, 1}
	rowSkipSize := [4]int{8, 8, 4, 2}

	d := make([]byte, width*height)
	dy := 0

	for i := 0; i < 4; i++ {
		for sy := startingRow[i]; sy < height; sy += rowSkipSize[i] {
			copy(d[dy*width:(dy+1)*width], frame.data[sy*width:(sy+1)*width])
		}
	}

	return d
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

	data.width = int(l.LogicalScreenWidth)
	data.height = int(l.LogicalScreenHeight)

	if verbose {
		log.Printf("Logical Screen Descriptor: %s\n", l)
	}

	if l.ColorResolution != 8 {
		return nil, errors.New("Not supported: ColorResolution != 8")
	}

	if l.GlobalColorTableFlag {
		data.palette = make([]Rgb, l.SizeOfGlobalColorTable)
		data.palette.UnmarshalBinary(l.GlobalColorTable)
	}

	nextDelay := 0
	for {
		b, err := readByte(r)
		if err != nil {
			return nil, err
		}

		switch b {
		case 0x2C:
			i, err := readImageDescriptor(r)
			if err != nil {
				return nil, err
			}
			if verbose {
				log.Printf("Image Descriptor: %s\n", i)
			}

			frame, err := readTableBasedImageData(r, int(i.ImageWidth), int(i.ImageHeight))
			if err != nil {
				return nil, err
			}
			frame.xOffset = int(i.ImageLeftPosition)
			frame.yOffset = int(i.ImageTopPosition)
			frame.delay = nextDelay

			if i.LocalColorTableFlag {
				frame.palette = make([]Rgb, i.SizeOfLocalColorTable)
				frame.palette.UnmarshalBinary(i.LocalColorTable)
			}

			if i.InterlaceFlag {
				frame.data = deinterlace(frame, int(i.ImageWidth), int(i.ImageHeight))
			}

			data.frames = append(data.frames, *frame)
		case 0x21:
			b, err := readByte(r)
			if err != nil {
				return nil, err
			}

			switch b {
			case 0xF9:
				//Graphic Control Extension
				g, err := readGraphicControlExtension(r)
				if err != nil {
					return nil, err
				}
				if verbose {
					log.Printf("Graphic Control Extension: %s\n", g)
				}
				nextDelay = int(g.DelayTime)
			case 0xFE:
				//Comment Extension
				if verbose {
					log.Println("Skip Comment Extension")
				}
				err := skipBlock(r)
				if err != nil {
					return nil, err
				}
			case 0x01:
				//Plain Text Extension
				if verbose {
					log.Println("Skip Plain Text Extension")
				}
				err := skipBlock(r)
				if err != nil {
					return nil, err
				}
			case 0xFF:
				//Application Extension
				a, err := readApplicationExtension(r)
				if err != nil {
					return nil, err
				}
				if verbose {
					log.Printf("Application Extension: %s\n", a)
				}
			default:
				return nil, fmt.Errorf("Unknown code: 0x21%02x", b)
			}
		case 0x3b:
			return &data, nil
		default:
			return nil, fmt.Errorf("Unknown code: 0x%02x", b)
		}
	}
}
