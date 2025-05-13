package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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

func getExecutableDir() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exePath), nil
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {

	config := readConfig(configPath)
	var targetURL string = "https://www.ptd-hs.jp/GetData?agency_id=" + config.agencyID + "&uid=" + config.uid

	exeDir, _ := getExecutableDir()
	zipPath := filepath.Join(exeDir, "static", "gtfs.zip")
	destDir := filepath.Join(exeDir, "static")

	if r.Method == "POST" {

		renderTemplate(w, "save", nil)
		downloadFile(zipPath, targetURL)

		if err := extract(zipPath, destDir); err != nil {
			fmt.Println(err)
		}

	} else {
		fmt.Fprintf(w, "Download Failed.")
	}
}

func downloadFile(filepath string, url string) error {

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func extract(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	ext := filepath.Ext(src)
	rep := regexp.MustCompile(ext + "$")
	dir := filepath.Base(rep.ReplaceAllString(src, ""))

	destDir := filepath.Join(dest, dir)
	// ファイル名のディレクトリを作成する
	if err := os.MkdirAll(destDir, 0600); err != nil {
		return err
	}

	for _, f := range r.File {
		if f.Mode().IsDir() {
			continue
		}
		if err := saveExtractedFile(destDir, *f); err != nil {
			return err
		}
	}
	return nil
}

func saveExtractedFile(destDir string, f zip.File) error {
	// 展開先のパス
	destPath := filepath.Join(destDir, f.Name)

	// 子・孫ディレクトリがあれば作成
	if err := os.MkdirAll(filepath.Dir(destPath), f.Mode()); err != nil {
		return err
	}

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, rc); err != nil {
		return err
	}

	return nil
}
