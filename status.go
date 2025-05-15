package main

import (
	"encoding/json"
	"fmt"
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

// getTimetable 定期的に運行情報を取得する
func getTimetable() []TimeTable {

	dbFile := "dynamic.sql"

	db, err := setupDb(dbFile)
	if err != nil {
		log.Fatalf("データベース接続に失敗: %v", err)
	}
	defer db.Close()

	vehiclePosData := fetchVehiclePosition()
	tripUpdateData := fetchTripUpdate()

	insertVehiclePositionResponse(db, vehiclePosData)
	insertTripUpdateResponse(db, tripUpdateData)

	for _, vp := range vehiclePosData.Entity {
		fmt.Printf("%s\n", vp.ID)
	}

	for _, tu := range tripUpdateData.Entity {
		fmt.Printf("%s\n", tu.TripUpdate.Trip.RouteID)
	}

	return nil

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
		{Remark: "鮫洲で通過待ち", RouteID: "普通", DepartureTime: "17:19", Destination: "浦賀"},
		{Remark: "", RouteID: "急行", DepartureTime: "17:22", Destination: "羽田空港"},
		{Remark: "遅延: 約3分", RouteID: "快特", DepartureTime: "17:27", Destination: "三崎口"},
	}
	renderTemplate(w, "time-table", timetableData)
}
