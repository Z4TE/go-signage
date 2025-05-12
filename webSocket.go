package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 実際にはOriginヘッダーの検証を行うべきです
	},
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Close()

	for {
		time.Sleep(2 * time.Minute)
		trainInfo, err := getLatestTrainInfoFromDB("dynamic.sql")
		if err != nil {
			log.Println("データベースからの情報取得エラー:", err)
			continue
		}

		err = ws.WriteJSON(trainInfo)
		if err != nil {
			log.Printf("WebSocket送信エラー: %v", err)
			break
		}
	}
}

func getLatestTrainInfoFromDB(dbFile string) (map[string]string, error) {
	db, err := setupDb(dbFile)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := "SELECT departure_time, delay_status, operation_status FROM train_status ORDER BY update_time DESC LIMIT 1"
	row := db.QueryRow(query)

	var departureTime string
	var delayStatus string
	var operationStatus string
	err = row.Scan(&departureTime, &delayStatus, &operationStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("SQLクエリ実行エラー: %w", err)
	}

	trainInfo := map[string]string{
		"出発時刻": departureTime,
		"遅延":   delayStatus,
		"運行状況": operationStatus,
		"取得時刻": time.Now().Format("15:04"),
	}
	return trainInfo, nil
}
