package pngDecoder

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"pnGo/compression"
	"pnGo/utils"
)

type Chunk struct {
	lenght   uint32
	typ      []uint8
	data     []uint8
	crc      uint32
	critical bool
}

type PngDecoder struct {
	data     []uint8
	idx      uint
	finished bool
}

func NewDecoder(data []byte) (*PngDecoder, error) {
	if !isPNG(data) {
		return nil, errors.New("not a png")
	}
	return &PngDecoder{
		data: data,
		idx:  8,
	}, nil
}
func ParseIHDR(data []byte) (*IHDR, error) {
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
	if string(chunk_type) == "IEND" {
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

func (pd *PngDecoder) Decode() {
	var ihdr *IHDR
	compressedData := []uint8{}
	chunk, err := pd.nextChunk()
	for err == nil && chunk != nil {
		if chunk.critical {
			if string(chunk.typ) == "IHDR" {
				ihdr, err = ParseIHDR(chunk.data)
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

	outputFile, err := utils.CreatePPM("output.ppm", int(ihdr.Width), int(ihdr.Height))
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
			dp, err := utils.BytesToUint32Slice(scanline[1:])
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
}
