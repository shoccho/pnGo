package pngDecoder

import (
	"math"
	"pnGo/utils"
)

type FilterMethod byte

const (
	NONE FilterMethod = iota
	LEFT
	UP
	AVG
	PAETH
)

func processNoneFilter(scanline []byte) []byte {
	dp, err := utils.BytesToUint32Slice(scanline)
	if err != nil {
		panic(err.Error())
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
	return processedLine
}

func processLeftFilter(scanline []byte, bytesPerPixel byte) []byte {
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
	return processedLine
}

func processUpFilter(previousLine []byte, scanline []byte, bytesPerPixel byte) []byte {
	processedLine := make([]byte, len(scanline)/4*3)
	processedIndex := 0
	var rawIndex int = 1

	for rawIndex < int(len(scanline)) {
		for b := 0; b < 3; b++ {
			colorIndex := rawIndex + b
			var prev byte = 0
			if previousLine != nil && processedIndex < len(previousLine) {
				prev = previousLine[processedIndex]
			}
			curr := scanline[colorIndex]
			scanline[colorIndex] = byte(byte(prev) + curr)
			processedLine[processedIndex] = scanline[colorIndex]
			processedIndex++
		}
		rawIndex += int(bytesPerPixel)
	}
	return processedLine
}

func processAvgFilter(previousLine []byte, scanline []byte, bytesPerPixel byte) []byte {
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
			if previousLine != nil {
				above = previousLine[processedIndex]
			}
			curr := scanline[colorIndex]
			avg := int(math.Floor((float64(left) + float64(above)) / 2.0))
			scanline[colorIndex] = byte((int(curr) + avg))
			processedLine[processedIndex] = scanline[colorIndex]
			processedIndex++
		}
		rawIndex += uint(bytesPerPixel)
	}
	return processedLine
}

func processPaethFilter(previousLine []byte, scanline []byte, bytesPerPixel byte) []byte {
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

			if previousLine != nil {
				above = previousLine[processedIndex]
			}
			if previousLine != nil && processedIndex >= 3 {
				upperLeft = previousLine[processedIndex-3]
			}

			curr := scanline[colorIndex]
			paeth := paethPredictor(int(left), int(above), int(upperLeft))

			scanline[colorIndex] = byte((int(curr) + paeth))
			processedLine[processedIndex] = scanline[colorIndex]
			processedIndex++
		}
		rawIndex += uint(bytesPerPixel)
	}
	return processedLine
}
