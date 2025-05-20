package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

type Config struct {
	UID      string `json:"uid,omitempty"`
	AgencyID string `json:"agencyID,omitempty"`
	Target   string `json:"target,omitempty"`
}

func readConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	var config Config
	decoder := json.NewDecoder(f)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return &config, nil
}

func readOrCreateConfig(path string) (*Config, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		// ファイルが存在しない場合は作成する
		emptyConfig := &Config{}
		if err := writeConfig(path, emptyConfig); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return emptyConfig, nil
	} else if err != nil {
		// その他のエラー
		return nil, fmt.Errorf("failed to check if config exists: %w", err)
	}

	// ファイルが存在する場合は読み込む
	return readConfig(path)
}

func writeConfig(path string, config *Config) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	err = encoder.Encode(config)
	if err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	return nil
}

func getExecutableDir() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exePath), nil
}

// 静的GTFSデータのダウンロードに必要な情報を返す
func gtfsDownloadInfo() (string, string, string) {
	config, err := readConfig(configPath)
	if err != nil {
		fmt.Printf("Failed to open config file: %v\n", err)
	}

	var targetURL string = "https://www.ptd-hs.jp/GetData?agency_id=" + config.AgencyID + "&uid=" + config.UID

	exeDir, _ := getExecutableDir()
	zipPath := filepath.Join(exeDir, "static", "gtfs.zip")
	destDir := filepath.Join(exeDir, "static")

	return targetURL, zipPath, destDir
}

func updateStatic(dbFile string) {

	targetURL, zipPath, destDir := gtfsDownloadInfo()

	// 旧バージョンのGTFSファイルを削除
	os.Remove(zipPath)
	os.RemoveAll("./static/gtfs")

	// 既存のDBを消去
	staticDbFile := filepath.Join("./databases", dbFile)
	os.Remove(staticDbFile)

	downloadFile(zipPath, targetURL)

	if err := extract(zipPath, destDir); err != nil {
		fmt.Println(err)
	}

	// 新データでDB初期化
	initStaticDb(dbFile)

}

func downloadHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {

		targetURL, zipPath, destDir := gtfsDownloadInfo()

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
