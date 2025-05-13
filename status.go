package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type TimeTable struct {
	Remark        string `json:"remark"`
	RouteID       string `json:"route_id"`
	DepartureTime string `json:"departure_time"`
	Destination   string `json:"destination"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 全てのオリジンからの接続を許可 (本番環境では制限すべき)
	},
}

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan []TimeTable)
var mutex sync.Mutex

// getTimetable 定期的に運行情報を取得する関数 (実際の実装は外部API呼び出しやDBアクセスなど)
func getTimetable() []TimeTable {
	// ここは実際には外部APIやDBからデータを取得する処理が入ります
	// サンプルのデータとして固定値を返します
	return []TimeTable{
		{Remark: "まもなく", RouteID: "1", DepartureTime: "17:05", Destination: "福島"},
		{Remark: "", RouteID: "2", DepartureTime: "17:10", Destination: "郡山"},
		{Remark: "", RouteID: "3", DepartureTime: "17:15", Destination: "いわき"},
	}
}

func broadcastTimetable() {
	ticker := time.NewTicker(1 * time.Minute) // 1分間隔で運行情報を送信
	defer ticker.Stop()

	for range ticker.C {
		timetable := getTimetable()
		broadcast <- timetable
	}
}

func handleBroadcasts() {
	for timetable := range broadcast {
		jsonBytes, err := json.Marshal(timetable)
		if err != nil {
			log.Println("Error marshalling JSON:", err)
			continue
		}

		mutex.Lock()
		for client := range clients {
			err := client.WriteMessage(websocket.TextMessage, jsonBytes)
			if err != nil {
				log.Printf("error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
		mutex.Unlock()
	}
}

func timetableHandler(w http.ResponseWriter, r *http.Request) {
	timetableData := []TimeTable{
		{Remark: "まもなく", RouteID: "1", DepartureTime: "17:05", Destination: "福島"},
		{Remark: "", RouteID: "2", DepartureTime: "17:10", Destination: "郡山"},
		{Remark: "", RouteID: "3", DepartureTime: "17:15", Destination: "いわき"},
	}
	renderTemplate(w, "time-table", timetableData)
}
