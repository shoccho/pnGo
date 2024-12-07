package utils

import (
	"encoding/binary"
	"fmt"
	"os"
)

func BytesToLenght(data []byte) uint32 {
	return binary.BigEndian.Uint32(data)
}

func BytesToUint32Slice(b []byte) ([]uint32, error) {
	uint32Slice := make([]uint32, len(b)/4+1)
	for i := 0; i < len(b); i += 4 {
		uint32Slice[i/4] = binary.LittleEndian.Uint32(b[i : i+4])
	}
	return uint32Slice, nil
}

func CreatePPM(name string, width, height int) (*os.File, error) {
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
