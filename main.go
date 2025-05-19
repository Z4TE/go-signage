package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

// 設定ファイル
const configPath string = "settings.json"

// 作成するDBのリスト
const staticDbFile string = "static.sql"
const dynamicDbFile string = "dynamic.sql"

func main() {
	config, configErr := readOrCreateConfig(configPath)
	if configErr != nil {
		fmt.Printf("Failed to open config file: %v\n", configErr)
	}

	var port string = "8888"
	var errorMessage, errorTitle string

	// DBが存在しなければ初期化
	staticDbFileName := filepath.Join("./databases", staticDbFile)
	dynamicDbFileName := filepath.Join("./databases", dynamicDbFile)

	if _, err := os.Stat(staticDbFileName); os.IsNotExist(err) {
		initStaticDb(staticDbFile)
	} else if err == nil {
		fmt.Println(staticDbFileName, "は存在します。")
	} else {
		fmt.Println("ファイル", staticDbFileName, "の状態を確認中にエラーが発生しました:", err)
	}

	if _, err := os.Stat(dynamicDbFileName); os.IsNotExist(err) {
		initDynamicDb(dynamicDbFile)
	} else if err == nil {
		fmt.Println(dynamicDbFileName, "は存在します。")
	} else {
		fmt.Println("ファイル", dynamicDbFileName, "の状態を確認中にエラーが発生しました:", err)
	}

	// feedの期限を確認
	checkFeedEndDate(staticDbFile, updateStatic)

	if config.UID == "" {
		errorMessage = "API key is empty."
		errorTitle = "API key error"
	}

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
	http.HandleFunc("/save_settings", submitHandler)

	// DB更新用
	http.HandleFunc("/update", updateHandler)

	// ヘルプページ
	http.HandleFunc("/help", func(w http.ResponseWriter, r *http.Request) {
		renderTemplate(w, "help", nil)
	})

	// ダウンローダ
	http.HandleFunc("/dl", downloadHandler)

	// websocket
	http.HandleFunc("/ws", handleConnections)

	go broadcastTimetable()
	go handleBroadcasts()

	// 時刻表
	http.HandleFunc("/time-table", timetableHandler)

	fmt.Printf("Listening on localhost:%s...\n", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		// エラーが発生した場合は内容を表示して終了
		fmt.Printf("Failed to launch www server: %v\n", err)
	}
}
