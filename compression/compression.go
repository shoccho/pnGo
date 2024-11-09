package compression

import (
	"bytes"
	"compress/zlib"
	"io"
)

func InflateData(compressedData []byte) ([]byte, error) {
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
