package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
)

func BytesToLenght(data []byte) uint32 {
	return binary.BigEndian.Uint32(data)
}
func isPNG(data []byte) bool {
	pngHeader := []uint8{137, 80, 78, 71, 13, 10, 26, 10}
	n := len(pngHeader)
	if len(data) < n {
		return false
	}
	return bytes.Equal(pngHeader, data[0:n])
}

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

	// Read the bytes into the struct
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
	chunk_data, err := p.tryAdvance(uint(BytesToLenght(length)))
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
		lenght:   BytesToLenght(length),
		typ:      chunk_type,
		data:     chunk_data,
		crc:      BytesToLenght(crc),
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

	fmt.Println("isPNG?: ", isPNG(data))
	pd, err := newPngDecoder(data)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}
	chunk, err := pd.nextChunk()
	for err == nil && chunk != nil {
		if chunk.critical {
			fmt.Printf("Processing chunk: %s\n", string(chunk.typ))
			if string(chunk.typ) == "IHDR" {
				ihdr, err := parseIHDR(chunk.data)
				if err != nil {
					fmt.Println(err.Error())
				}
				fmt.Println(ihdr)
			}
		}
		chunk, err = pd.nextChunk()
	}
	if err != nil {
		fmt.Println("error :", err.Error())
	}

}
