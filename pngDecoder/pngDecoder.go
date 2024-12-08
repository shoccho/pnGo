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

type PngData struct {
	Height uint32
	Width  uint32
	Data   [][]byte
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

func NewDecoder(data []byte) (*PngDecoder, error) {
	if !isPNG(data) {
		return nil, errors.New("not a png")
	}
	return &PngDecoder{
		data: data,
		idx:  8,
	}, nil
}

type FilterMethod byte

const (
	NONE FilterMethod = iota
	LEFT
	UP
	AVG
	PAETH
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

func (p *PngDecoder) nextChunk() (*Chunk, error) {
	if p.finished {
		return nil, nil
	}
	length, err := p.tryAdvance(4)
	if err != nil {
		return nil, err
	}
	chunkType, err := p.tryAdvance(4)
	if err != nil {
		return nil, err
	}
	chunkData, err := p.tryAdvance(uint(utils.BytesToLenght(length)))
	if err != nil {
		return nil, err
	}
	crc, err := p.tryAdvance(4)
	if err != nil {
		return nil, err
	}
	if string(chunkType) == "IEND" {
		p.finished = true
	}
	return &Chunk{
		lenght:   utils.BytesToLenght(length),
		typ:      chunkType,
		data:     chunkData,
		crc:      utils.BytesToLenght(crc),
		critical: chunkType[0] >= 'A' && chunkType[0] <= 'Z',
	}, nil
}

func (p *PngDecoder) tryAdvance(length uint) ([]uint8, error) {
	if p.idx+length > uint(len(p.data)) {
		return nil, errors.New("eof")
	}

	p.idx += length
	return p.data[p.idx-length : p.idx], nil
}

func (pd *PngDecoder) Decode() (*PngData, error) {
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
				if ihdr.colorType() != rgba {
					return nil, fmt.Errorf("unsupported color type")
				}
			} else if string(chunk.typ) == "IDAT" {
				compressedData = append(compressedData, chunk.data...)
			} else if string(chunk.typ) == "IEND" {
				break
			}
		}
		chunk, err = pd.nextChunk()
	}
	if ihdr.CompressionMethod != 0 {
		return nil, fmt.Errorf("unsupported compression method")
	}
	decompressed, err := compression.InflateData(compressedData)

	if err != nil {
		fmt.Println("error :", err.Error())
	}
	if ihdr.InterlaceMethod != 0 {
		return nil, fmt.Errorf("unsupported Interlacing mode")
	}
	bytesPerPixel := 4 * ihdr.BitDepth / 8

	scanlineSize := ihdr.Width*uint32(bytesPerPixel) + 1

	scanlines := make([][]byte, ihdr.Height)

	for i := 0; i < int(ihdr.Height); i++ {
		filterMethod := decompressed[i*int(scanlineSize)]
		start := i * int(scanlineSize)
		end := start + int(scanlineSize) - 1
		scanline := decompressed[start:end]
		if filterMethod == byte(NONE) {
			dp, err := utils.BytesToUint32Slice(scanline[1:])
			if err != nil {
				fmt.Print("error in bytes to slice", err.Error())
			}
			processedLine := make([]byte, len(dp)*3)
			processedIndex := 0
			for _, px := range dp {
				processedLine[processedIndex] = byte(px)
				processedIndex++
				processedLine[processedIndex] = byte(px >> 8)
				processedIndex++
				processedLine[processedIndex] = byte(px >> 16)
				processedIndex++
			}
			scanlines[i] = processedLine
		} else if filterMethod == byte(LEFT) {
			processedLine := make([]byte, len(scanline)/4*3)
			processedIndex := 0
			var rawIndex uint = 1
			for rawIndex < uint(len(scanline)) {
				for b := 0; b < 3; b++ {
					colorIndex := rawIndex + uint(b)
					var prev byte = 0
					if colorIndex >= uint(bytesPerPixel) {
						prev = scanline[colorIndex-uint(bytesPerPixel)]
					}
					curr := scanline[colorIndex]
					scanline[colorIndex] = byte(byte(prev) + curr)
					processedLine[processedIndex] = scanline[colorIndex]
					processedIndex++
				}
				rawIndex += uint(bytesPerPixel)
			}
			scanlines[i] = processedLine
		} else if filterMethod == byte(UP) {
			processedLine := make([]byte, len(scanline)/4*3)
			processedIndex := 0
			var rawIndex int = 1

			for rawIndex < int(len(scanline)) {
				for b := 0; b < 3; b++ {
					colorIndex := rawIndex + b
					var prev byte = 0
					if i > 0 && processedIndex < len(scanlines[i-1]) {

						prev = scanlines[i-1][processedIndex]
					}
					curr := scanline[colorIndex]
					scanline[colorIndex] = byte(byte(prev) + curr)
					processedLine[processedIndex] = scanline[colorIndex]
					processedIndex++
				}
				rawIndex += int(bytesPerPixel)
			}
			scanlines[i] = processedLine
		} else if filterMethod == byte(AVG) {
			processedLine := make([]byte, len(scanline)/4*3)
			processedIndex := 0
			var rawIndex uint = 1

			for rawIndex < uint(len(scanline)) {
				for b := 0; b < 3; b++ {
					colorIndex := rawIndex + uint(b)
					var left, above byte
					if colorIndex >= uint(bytesPerPixel) {
						left = scanline[colorIndex-uint(bytesPerPixel)]
					}
					if i > 0 {
						above = scanlines[i-1][processedIndex]
					}
					curr := scanline[colorIndex]
					avg := int(math.Floor((float64(left) + float64(above)) / 2.0))
					scanline[colorIndex] = byte((int(curr) + avg))
					processedLine[processedIndex] = scanline[colorIndex]
					processedIndex++
				}
				rawIndex += uint(bytesPerPixel)
			}
			scanlines[i] = processedLine

		} else if filterMethod == byte(PAETH) {
			processedLine := make([]byte, len(scanline)/4*3)
			processedIndex := 0
			var rawIndex uint = 1
			for rawIndex < uint(len(scanline)) {
				for b := 0; b < 3; b++ {
					colorIndex := rawIndex + uint(b)
					var left, above, upperLeft byte

					if colorIndex >= uint(bytesPerPixel) {
						left = scanline[colorIndex-uint(bytesPerPixel)]
					}

					if i > 0 {
						above = scanlines[i-1][processedIndex]
					}
					if i > 0 && processedIndex >= 3 {
						upperLeft = scanlines[i-1][processedIndex-3]
					}

					curr := scanline[colorIndex]
					paeth := paethPredictor(int(left), int(above), int(upperLeft))

					scanline[colorIndex] = byte((int(curr) + paeth))
					processedLine[processedIndex] = scanline[colorIndex]
					processedIndex++
				}
				rawIndex += uint(bytesPerPixel)
			}
			scanlines[i] = processedLine
		} else {
			fmt.Println("Unsupported filter method", filterMethod)
		}

	}

	return &PngData{ihdr.Height, ihdr.Width, scanlines}, nil
}
