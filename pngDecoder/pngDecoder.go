package pngDecoder

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
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
	bytesPerPixel := 4 * ihdr.BitDepth / 8 //maybe int?

	scanlineSize := ihdr.Width*uint32(bytesPerPixel) + 1

	scanlines := make([][]byte, ihdr.Height)

	for i := 0; i < int(ihdr.Height); i++ {
		filterMethod := decompressed[i*int(scanlineSize)]
		start := i * int(scanlineSize)
		end := start + int(scanlineSize) - 1
		scanline := decompressed[start:end]
		if filterMethod == byte(NONE) {
			scanlines[i] = processNoneFilter(scanline[1:])
		} else if filterMethod == byte(LEFT) {
			scanlines[i] = processLeftFilter(scanline, bytesPerPixel)
		} else if filterMethod == byte(UP) {
			if i == 0 {
				scanlines[i] = processUpFilter(nil, scanline, bytesPerPixel)
			} else {
				scanlines[i] = processUpFilter(scanlines[i-1], scanline, bytesPerPixel)
			}
		} else if filterMethod == byte(AVG) {
			if i == 0 {
				scanlines[i] = processAvgFilter(nil, scanline, bytesPerPixel)
			} else {
				scanlines[i] = processAvgFilter(scanlines[i-1], scanline, bytesPerPixel)
			}
		} else if filterMethod == byte(PAETH) {
			if i == 0 {
				scanlines[i] = processPaethFilter(nil, scanline, bytesPerPixel)
			} else {
				scanlines[i] = processPaethFilter(scanlines[i-1], scanline, bytesPerPixel)
			}
		} else {
			fmt.Println("Unsupported filter method", filterMethod)
		}
	}

	return &PngData{ihdr.Height, ihdr.Width, scanlines}, nil
}
