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
	Width             uint32
	Height            uint32
	BitDepth          byte
	ColorType         byte
	CompressionMethod byte
	FilterMethod      byte
	InterlaceMethod   byte
}

func (ihdr *IHDR) colorType() ColorType {
	switch ihdr.ColorType {
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
