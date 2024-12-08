package main

import (
	"fmt"
	"os"
	"pnGo/pngDecoder"
	"pnGo/utils"
)

func main() {

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <file-path>")
		return
	}
	inputFilePath := os.Args[1]

	file, err := os.Open(inputFilePath)
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

	pd, err := pngDecoder.NewDecoder(data)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}
	pngData, err := pd.Decode()
	if err != nil {
		panic(err)
	}
	outputFile, err := utils.CreatePPM("output.ppm", int(pngData.Width), int(pngData.Height))
	if err != nil {
		panic(err)
	}
	defer outputFile.Close()

	for _, scanline := range pngData.Data {
		outputFile.Write(scanline)
	}
}
