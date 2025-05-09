package main

import (
	"fmt"
	"net/http"
	"time"
)

const configPath string = "settings.conf"
const refreshInterval = 2 * time.Minute

var latestStatus *TimeTable

func main() {

	var port string = "8888"
	var errorMessage, errorTitle string

	config := readConfig(configPath)

	if config.uid == "" {
		errorMessage = "API key is empty."
		errorTitle = "API key error"
	}

	var apiURL string = "https://www.ptd-hs.jp/GetVehiclePosition?uid=" + config.uid + "&agency_id=" + config.agency_id + "&output=json"

	// Bootstrap読み込み
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// 定期的に車両情報を取得して更新する
	go func() {
		for range time.Tick(refreshInterval) {
			data := fetchStatus(apiURL)
			latestStatus = populateTimeTable(data)
			fmt.Println("Time table has been updated:", time.Now().Format(time.RFC3339))
		}
	}()

	// gtfs.goから車両情報を取得
	data := fetchStatus(apiURL)
	latestStatus := populateTimeTable(data)

	// トップページ
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("settings_saved") == "true" {
			errorMessage = "Please restart the server to apply the new settings."
			errorTitle = "Restart required"
		}

		data := struct {
			ErrorMessage, ErrorTitle string
		}{
			ErrorMessage: errorMessage,
			ErrorTitle:   errorTitle,
		}

		renderTemplate(w, "index", data)
	})

	// テストページ
	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		renderTemplate(w, "test", nil)
	})

	// 設定ページ
	http.HandleFunc("/settings", func(w http.ResponseWriter, r *http.Request) {
		renderTemplate(w, "settings", nil)
	})

	// 設定保存用
	http.HandleFunc("/save_settings", saveSettings)

	// 時刻表ページ
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		renderTemplate(w, "status", latestStatus)
	})

	// ヘルプページ
	http.HandleFunc("/help", func(w http.ResponseWriter, r *http.Request) {
		renderTemplate(w, "help", nil)
	})

	// ダウンローダ
	http.HandleFunc("/dl", downloadHandler)

	fmt.Printf("Listening on localhost:%s...\n", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		// エラーが発生した場合は内容を表示して終了
		fmt.Printf("Failed to launch www server: %v\n", err)
	}
}
