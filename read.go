package main

import (
	"fmt"
	"io"
	"os"
)

func readConfig(path string) (string, error) {

	f, err := os.Open(path)
	if err != nil {
		fmt.Printf("Failed to file: %v\n", err)
	}
	defer f.Close()

	contentBytes, err := io.ReadAll(f)
	if err != nil {
		fmt.Printf("Failed to read file: %v\n", err)
	}
	contentString := string(contentBytes)
	return contentString, nil
}
