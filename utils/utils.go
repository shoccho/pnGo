package utils

import "encoding/binary"

func BytesToLenght(data []byte) uint32 {
	return binary.BigEndian.Uint32(data)
}
