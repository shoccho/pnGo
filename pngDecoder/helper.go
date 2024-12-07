package pngDecoder

import "bytes"

func isPNG(data []byte) bool {
	pngHeader := []uint8{137, 80, 78, 71, 13, 10, 26, 10}
	n := len(pngHeader)
	if len(data) < n {
		return false
	}
	return bytes.Equal(pngHeader, data[0:n])
}

func paethPredictor(a, b, c int) int {
	p := a + b - c
	pa := abs(p - a)
	pb := abs(p - b)
	pc := abs(p - c)

	if pa <= pb && pa <= pc {
		return a
	} else if pb <= pc {
		return b
	}
	return c
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
