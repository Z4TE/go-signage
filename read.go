package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	uid       string
	agency_id string
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
		} else if strings.HasPrefix(line, "agency_id:") {
			config.agency_id = strings.TrimSpace(strings.TrimPrefix(line, "agency_id:"))
		}
	}

	return &config
}
