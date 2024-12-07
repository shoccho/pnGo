package main

import (
	"bytes" //TODO: own zlib maybe?
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"pnGo/compression"
	"pnGo/utils"
)

func isPNG(data []byte) bool {
	pngHeader := []uint8{137, 80, 78, 71, 13, 10, 26, 10}
	n := len(pngHeader)
	if len(data) < n {
		return false
	}
	return bytes.Equal(pngHeader, data[0:n])
}

type ColorType int

const (
	y ColorType = iota
	rgb
	pallete
	ya
	rgba
)

type Chunk struct {
	lenght   uint32
	typ      []uint8
	data     []uint8
	crc      uint32
	critical bool
}

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

type PngDecoder struct {
	data     []uint8
	idx      uint
	finished bool
}

func newPngDecoder(data []byte) (*PngDecoder, error) {
	if !isPNG(data) {
		return nil, errors.New("not a png")
	}
	return &PngDecoder{
		data: data,
		idx:  8,
	}, nil
}

func parseIHDR(data []byte) (*IHDR, error) {
	var ihdr IHDR

	reader := bytes.NewReader(data)
	err := binary.Read(reader, binary.BigEndian, &ihdr)
	if err != nil {
		fmt.Println("Error reading bytes into struct:", err)
		return nil, nil
	}
	return &ihdr, nil
}

func (p *PngDecoder) nextChunk() (*Chunk, error) {
	if p.finished {
		return nil, nil
	}
	length, err := p.tryAdvance(4)
	if err != nil {
		return nil, err
	}
	chunk_type, err := p.tryAdvance(4)
	if err != nil {
		return nil, err
	}
	chunk_data, err := p.tryAdvance(uint(utils.BytesToLenght(length)))
	if err != nil {
		return nil, err
	}
	crc, err := p.tryAdvance(4)
	if err != nil {
		return nil, err
	}
	if "IEND" == string(chunk_type) {
		p.finished = true
	}
	return &Chunk{
		lenght:   utils.BytesToLenght(length),
		typ:      chunk_type,
		data:     chunk_data,
		crc:      utils.BytesToLenght(crc),
		critical: chunk_type[0] >= 'A' && chunk_type[0] <= 'Z',
	}, nil
}

func (p *PngDecoder) tryAdvance(length uint) ([]uint8, error) {
	if p.idx+length > uint(len(p.data)) {
		return nil, errors.New("eof")
	}

	p.idx += length
	return p.data[p.idx-length : p.idx], nil
}

func createPPM(name string, width, height int) (*os.File, error) {
	file, err := os.Create(name)
	if err != nil {
		return nil, err
	}
	_, err = fmt.Fprintf(file, "P6\n%d %d\n255\n", width, height)
	if err != nil {
		file.Close()
		return nil, err
	}

	return file, nil
}
func bytesToUint32Slice(b []byte) ([]uint32, error) {
	uint32Slice := make([]uint32, len(b)/4+1)
	for i := 0; i < len(b); i += 4 {
		uint32Slice[i/4] = binary.LittleEndian.Uint32(b[i : i+4])
	}
	return uint32Slice, nil
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

func main() {
	file, err := os.Open("test.png")
	if err != nil {
		fmt.Printf("Error opening file %s\n", err.Error())
		return
	}
	defer file.Close()
	data := make([]byte, 1*1024*1024)
	_, err = file.Read(data)
	if err != nil {
		fmt.Printf("error Reading data %s", err.Error())
	}

	pd, err := newPngDecoder(data)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}
	var ihdr *IHDR
	compressedData := []uint8{}
	chunk, err := pd.nextChunk()
	for err == nil && chunk != nil {
		if chunk.critical {
			if string(chunk.typ) == "IHDR" {
				ihdr, err = parseIHDR(chunk.data)
				if err != nil {
					fmt.Println(err.Error())
				}
				fmt.Println(ihdr)
				fmt.Println(ihdr.colorType())
				//support more color types
				if ihdr.colorType() != rgba {
					fmt.Printf("Unsupported color type")
					return
				}
			} else if string(chunk.typ) == "IDAT" {
				compressedData = append(compressedData, chunk.data...)
			} else if string(chunk.typ) == "IEND" {
				break
			}
		}
		chunk, err = pd.nextChunk()
	}
	if ihdr.Compression_method != 0 {
		fmt.Println("Unsupported compression")
		return
	}
	decompressed, err := compression.InflateData(compressedData)

	if err != nil {
		fmt.Println("error :", err.Error())
	}
	if ihdr.Interlace_method != 0 {
		fmt.Println("Unsupported Interlacing mode")
		return
	}
	bytes_per_pixel := 4 * ihdr.Bit_depth / 8

	scanline_size := ihdr.Width*uint32(bytes_per_pixel) + 1
	fmt.Println(len(decompressed), scanline_size, ihdr.Height)

	outputFile, err := createPPM("output.ppm", int(ihdr.Width), int(ihdr.Height))
	if err != nil {
		panic(err)
	}
	defer outputFile.Close()
	scanlines := make([][]byte, ihdr.Height)

	for i := 0; i < int(ihdr.Height); i++ {
		filter_method := decompressed[i*int(scanline_size)]
		start := i * int(scanline_size)
		end := start + int(scanline_size) - 1
		scanline := decompressed[start:end]
		if filter_method == 0 {
			//none
			dp, err := bytesToUint32Slice(scanline[1:])
			if err != nil {
				fmt.Print("error in bytes to slice", err.Error())
			}
			scl := make([]byte, len(dp)*3)

			for idp, px := range dp {
				ri := idp * 3
				scl[ri] = byte(px)
				scl[ri+1] = byte(px >> 8)
				scl[ri+2] = byte(px >> 16)

				outputFile.Write([]byte{byte(px)})
				outputFile.Write([]byte{byte(px >> 8)})
				outputFile.Write([]byte{byte(px >> 16)})
			}
			scanlines[i] = scl
		} else if filter_method == 1 {
			//left
			scl := make([]byte, len(scanline)*3)
			ti := 0
			var sc_idx uint = 1
			for sc_idx < uint(len(scanline)) {
				for b := 0; b < 3; b++ {
					c_idx := sc_idx + uint(b)
					var prev byte = 0
					if c_idx >= uint(bytes_per_pixel) {
						prev = scanline[c_idx-uint(bytes_per_pixel)]
					}
					curr := scanline[c_idx]
					scanline[c_idx] = byte(byte(prev) + curr)
					scl[ti] = scanline[c_idx]
					ti++
					outputFile.Write([]byte{scanline[c_idx]})
				}
				sc_idx += uint(bytes_per_pixel)
			}
			scanlines[i] = scl
		} else if filter_method == 2 {
			//up
			scl := make([]byte, len(scanline)*3)
			ti := 0
			var sc_idx int = 1

			for sc_idx < int(len(scanline)) {
				for b := 0; b < 3; b++ {
					c_idx := sc_idx + b
					var prev byte = 0
					if i > 0 && ti < len(scanlines[i-1]) {

						prev = scanlines[i-1][ti]
					}
					curr := scanline[c_idx]
					scanline[c_idx] = byte(byte(prev) + curr)
					scl[ti] = scanline[c_idx]
					ti++
					outputFile.Write([]byte{scanline[c_idx]})
				}
				sc_idx += int(bytes_per_pixel)

			}
			scanlines[i] = scl
		} else if filter_method == 3 {
			//avg
			scl := make([]byte, len(scanline)*3)
			ti := 0
			var sc_idx uint = 1

			for sc_idx < uint(len(scanline)) {
				for b := 0; b < 3; b++ {
					c_idx := sc_idx + uint(b)
					var left, above byte

					if c_idx >= uint(bytes_per_pixel) {
						left = scanline[c_idx-uint(bytes_per_pixel)]
					}

					if i > 0 {
						above = scanlines[i-1][ti]
					}

					curr := scanline[c_idx]

					avg := int(math.Floor((float64(left) + float64(above)) / 2.0))
					scanline[c_idx] = byte((int(curr) + avg))
					scl[ti] = scanline[c_idx]

					ti++
					outputFile.Write([]byte{scanline[c_idx]})
				}

				sc_idx += uint(bytes_per_pixel)
			}
			scanlines[i] = scl

		} else if filter_method == 4 {
			//paeth
			scl := make([]byte, len(scanline)*3)
			ti := 0
			var sc_idx uint = 1

			for sc_idx < uint(len(scanline)) {
				for b := 0; b < 3; b++ {
					c_idx := sc_idx + uint(b)
					var left, above, upperLeft byte

					if c_idx >= uint(bytes_per_pixel) {
						left = scanline[c_idx-uint(bytes_per_pixel)]
					}

					if i > 0 {
						above = scanlines[i-1][ti]
					}
					if i > 0 && ti >= 3 {
						upperLeft = scanlines[i-1][ti-3]
					}

					curr := scanline[c_idx]
					paeth := paethPredictor(int(left), int(above), int(upperLeft))

					scanline[c_idx] = byte((int(curr) + paeth))
					scl[ti] = scanline[c_idx]

					ti++
					outputFile.Write([]byte{scanline[c_idx]})
				}

				sc_idx += uint(bytes_per_pixel)
			}
			scanlines[i] = scl
		} else {
			fmt.Println("Unsupported filter method", filter_method)
		}
	}
	// for _, scline := range scanlines {
	// outputFile.Write(scline)
	// }
}
