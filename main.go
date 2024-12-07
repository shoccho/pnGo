package main

import (
	//TODO: own zlib maybe?

	"fmt"
	"os"
	"pnGo/pngDecoder"
)

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

	pd, err := pngDecoder.NewDecoder(data)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}
	pd.Decode()
	// for _, scline := range scanlines {
	// outputFile.Write(scline)
	// }
}
