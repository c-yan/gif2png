package main

// MarshalBinary converts palette entry to byte slice.
func (v Palette) MarshalBinary() (data []byte, err error) {
	data = make([]byte, 3*len(v))
	for i := 0; i < len(v); i++ {
		data[i*3] = v[i].r
		data[i*3+1] = v[i].g
		data[i*3+2] = v[i].b
	}
	return data, nil
}

// Rgb holds pixel data.
type Rgb struct {
	r byte
	g byte
	b byte
}

// Palette holds palette data.
type Palette []Rgb

// ImageData holds picture data.
type ImageData struct {
	width   int
	height  int
	palette Palette
	data    []byte
}
