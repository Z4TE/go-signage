package main

import (
	"fmt"
	"net/http"
)

func main() {

	var port string = "8888"
	path := "api_key.conf"

	key, readErr := readConfig(path)
	if readErr != nil {
		fmt.Printf("API key is null or invalid: %v\n", readErr)
		// key = "" // デフォルトのAPIキー、 api_key.confの内容が不正だった場合に使用
	}

	var apiURL string = "https://www.ptd-hs.jp/GetVehiclePosition?uid=" + key + "&agency_id=0704&output=json"

	// Bootstrap読み込み
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// トップページ
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		renderTemplate(w, "index", nil)
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
		renderTemplate(w, "status", nil)
	})

	fetchStatus(apiURL)

	fmt.Printf("Listening on localhost:%s...\n", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		// エラーが発生した場合は内容を表示して終了
		fmt.Printf("Failed to launch www server: %v\n", err)
	}
}
