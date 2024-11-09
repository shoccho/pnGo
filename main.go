package main

import (
	"bytes"
	"compress/zlib" //TODO: own zlib maybe?
	"encoding/binary"
	"errors"
	"fmt"
	"io"
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

func inflateData(compressedData []byte) ([]byte, error) {
	reader := bytes.NewReader(compressedData)

	zlibReader, err := zlib.NewReader(reader)
	if err != nil {
		return nil, err
	}
	defer zlibReader.Close()
	var decompressedData bytes.Buffer

	_, err = io.Copy(&decompressedData, zlibReader)
	if err != nil {
		return nil, err
	}
	return decompressedData.Bytes(), nil
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
	if len(b)%4 != 0 {
		return nil, fmt.Errorf("byte slice length must be a multiple of 4")
	}
	uint32Slice := make([]uint32, len(b)/4)
	for i := 0; i < len(b); i += 4 {
		uint32Slice[i/4] = binary.LittleEndian.Uint32(b[i : i+4])
	}
	return uint32Slice, nil
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
			// fmt.Printf("Processing chunk: %s\n", string(chunk.typ))
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
	decompressed, err := inflateData(compressedData)

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
	defer outputFile.Close()

	for i := 0; i < int(ihdr.Height); i++ {
		filter_method := decompressed[i*int(scanline_size)]
		if filter_method != 0 {
			fmt.Println("Unsupported filter method", filter_method)

		}
		start := i*int(scanline_size) + 1
		end := start + int(scanline_size) - 1
		dp, err := bytesToUint32Slice(decompressed[start:end])
		if err != nil {
			fmt.Print("error in bytes to slice", err.Error())
		}
		for _, px := range dp {
			outputFile.Write([]byte{byte(px)})
			outputFile.Write([]byte{byte(px >> 8)})
			outputFile.Write([]byte{byte(px >> 16)})
		}
	}
}
