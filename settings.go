package main

import (
	"encoding/json"
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
	uid := r.FormValue("uidInput") // HTMLのinput要素のname属性と一致させる
	agencyID := r.FormValue("agencyIdInput")

	// ファイルに保存

	file, err := os.OpenFile(configPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600) // 読み書き専用、存在しない場合は作成、存在する場合は内容をtruncate
	if err != nil {
		http.Error(w, "Failed to open file", http.StatusInternalServerError)
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	_, err = file.WriteString("uid: " + uid + "\n" + "agency_id: " + agencyID)
	if err != nil {
		http.Error(w, "Failed to write to file", http.StatusInternalServerError)
		fmt.Println("Error writing to file:", err)
		return
	}

	renderTemplate(w, "save", nil)
}

func submitHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(32 << 20) // 32MBのメモリ制限
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	uid := r.FormValue("uid")
	agencyID := r.FormValue("agency_id")

	formData := Config{
		UID:      uid,
		AgencyID: agencyID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(formData)

	fmt.Printf("受信データ: UID=%s, Agency_ID=%s\n", uid, agencyID)

	writeConfig("settings.json", &formData)
}

func updateHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	checkFeedEndDate(staticDbFile, updateStatic)
}
