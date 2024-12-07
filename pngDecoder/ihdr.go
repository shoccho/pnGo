package pngDecoder

type ColorType int

const (
	y ColorType = iota
	rgb
	pallete
	ya
	rgba
)

type IHDR struct {
	Width              uint32
	Height             uint32
	Bit_depth          byte
	Color_type         byte
	Compression_method byte
	Filter_method      byte
	Interlace_method   byte
}

func (ihdr *IHDR) colorType() ColorType {
	switch ihdr.Color_type {
	case 0:
		return y
	case 2:
		return rgb
	case 3:
		return pallete
	case 4:
		return ya
	case 6:
		return rgba
	}
	return -1
}
