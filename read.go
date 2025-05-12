package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	uid      string
	agencyID string
}

func readConfig(path string) *Config {

	var config Config

	f, err := os.Open(path)
	if err != nil {
		fmt.Printf("Failed to open file: %v\n", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "uid:") {
			config.uid = strings.TrimSpace(strings.TrimPrefix(line, "uid:"))
		} else if strings.HasPrefix(line, "agencyID:") {
			config.agencyID = strings.TrimSpace(strings.TrimPrefix(line, "agencyID:"))
		}
	}

	return &config
}
