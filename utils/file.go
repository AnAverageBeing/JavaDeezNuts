package utils

import (
	"io"
	"os"
)

func GetFileContent(filePath string) (string, error) {
	// Open the file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}

	// Create a buffer to store the file content
	var contentBuffer []byte

	// Read the file content into the buffer
	buffer := make([]byte, 1024) // Read 1024 bytes at a time
	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return "", err
		}
		if n == 0 {
			break
		}
		contentBuffer = append(contentBuffer, buffer[:n]...)
	}

	file.Close()

	// Convert the buffer to a string and return
	content := string(contentBuffer)
	return content, nil
}
