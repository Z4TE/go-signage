package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

const configPath string = "settings.conf"

func main() {
	config := readConfig(configPath)

	var port string = "8888"
	var errorMessage, errorTitle string
	var apiURL string = "https://www.ptd-hs.jp/GetVehiclePosition?uid=" + config.uid + "&agency_id=" + config.agency_id + "&output=json"

	// 作成するDBのリスト
	dbFiles := []string{"dynamic.sql", "static.sql"}

	// DBが存在しなければ初期化
	for _, dbFile := range dbFiles {
		filename := filepath.Join("./databases", dbFile)

		if _, err := os.Stat(filename); os.IsNotExist(err) {
			initStaticDb(dbFile)
		} else if err == nil {
			fmt.Println(filename, "は存在します。")
		} else {
			fmt.Println("ファイル", filename, "の状態を確認中にエラーが発生しました:", err)
		}
	}

	if config.uid == "" {
		errorMessage = "API key is empty."
		errorTitle = "API key error"
	}

	// gtfs.goから車両情報を取得
	data := fetchStatus(apiURL)

	//testDynamicDb("dynamic.sql", *data)

	// Bootstrap読み込み
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

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
		renderTemplate(w, "status", data)
	})

	// ヘルプページ
	http.HandleFunc("/help", func(w http.ResponseWriter, r *http.Request) {
		renderTemplate(w, "help", nil)
	})

	// ダウンローダ
	http.HandleFunc("/dl", downloadHandler)

	// 時刻表(リアルタイム)
	http.HandleFunc("/time-table", handleConnections)

	fmt.Printf("Listening on localhost:%s...\n", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		// エラーが発生した場合は内容を表示して終了
		fmt.Printf("Failed to launch www server: %v\n", err)
	}
}
