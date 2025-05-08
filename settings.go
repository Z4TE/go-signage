package main

import (
	"fmt"
	"net/http"
	"os"
)

func saveSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// フォームからAPIキーを取得
	apiKey := r.FormValue("keyInput") // HTMLのinput要素のname属性と一致させる

	// ファイルに保存
	filePath := "api_key.conf"
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600) // 読み書き専用、存在しない場合は作成、存在する場合は内容をtruncate
	if err != nil {
		http.Error(w, "Failed to open file", http.StatusInternalServerError)
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	_, err = file.WriteString(apiKey)
	if err != nil {
		http.Error(w, "Failed to write to file", http.StatusInternalServerError)
		fmt.Println("Error writing to file:", err)
		return
	}

	renderTemplate(w, "save", nil)
}
