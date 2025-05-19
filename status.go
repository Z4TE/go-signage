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
	Delay         string `json:"delay"`
	Destination   string `json:"destination"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 全てのオリジンからの接続を許可 (本番環境では制限すべき)
	},
}

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan []TimeTable)
var mutex sync.Mutex

// getTimetable 定期的に運行情報を取得する
func getTimetable() []TimeTable {

	timeTables := make([]TimeTable, 0)

	// マスタDBに接続
	db, err := setupDb(staticDbFile)
	if err != nil {
		log.Fatalf("データベース接続に失敗: %v", err)
	}
	defer db.Close()

	vehiclePosData := fetchVehiclePosition()
	tripUpdateData := fetchTripUpdate()

	// DBを更新
	// insertVehiclePositionResponse(db, vehiclePosData)
	// insertTripUpdateResponse(db, tripUpdateData)

	// vehiclePosData と tripUpdateData を組み合わせて TimeTable を作成
	count := 0
	for _, vpEntity := range vehiclePosData.Entity {
		if count >= 128 {
			break
		}

		routeID := vpEntity.Vehicle.Trip.RouteID
		tripID := vpEntity.Vehicle.Trip.TripID
		stopID := removeLastDash(vpEntity.Vehicle.StopID)

		// fmt.Printf("%s, %s\n", tripID, stopID)

		// routeID をキーとして tripUpdateData から関連する情報を探す
		for _, tuEntity := range tripUpdateData.Entity {
			if tuEntity.TripUpdate.Trip.RouteID == routeID {
				if len(tuEntity.TripUpdate.StopTimeUpdate) > 0 {
					// 最新の更新情報のみを取得
					stopTimeUpdates := tuEntity.TripUpdate.StopTimeUpdate
					if len(stopTimeUpdates) > 0 {
						latestUpdate := stopTimeUpdates[len(stopTimeUpdates)-1]
						// latestUpdate を使って処理を行う
						delay := ""
						if latestUpdate.Departure.Delay > 60 {
							delay = fmt.Sprintf("%d", latestUpdate.Departure.Delay)
						}

						// 路線名を問い合わせ
						routeNameQuery := "SELECT route_long_name FROM routes r WHERE r.route_id = ?"
						routeName, err := QuerySingleString(db, routeNameQuery, routeID)
						if err != nil {
							log.Fatal(err)
						}

						// 行先を問い合わせ
						destinationQuery := "SELECT stop_headsign FROM stop_times st WHERE st.trip_id = ? and st.stop_id = ?"
						destinationName, err := QuerySingleString(db, destinationQuery, tripID, stopID)
						if err != nil {
							log.Fatal(err)
						}

						timeTables = append(timeTables, TimeTable{
							RouteID:       routeName,
							Delay:         delay,
							DepartureTime: "aaa",
							Destination:   destinationName,
						})

					}
				}
			}
		}
		count++
	}
	return timeTables
}

// デバッグ用
// fmt.Printf("test\n")

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Close()

	mutex.Lock()
	clients[ws] = true
	mutex.Unlock()

	fmt.Println("Client connected")

	for {
		var msg map[string]interface{}
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, ws)
			break
		}
		fmt.Printf("Received message: %v\n", msg)
		// クライアントからのメッセージを処理する場合はここに記述
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

		// デバッグ用 取得した運行情報をコンソールに表示
		// fmt.Printf("%s", jsonBytes)

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

	timeTable := getTimetable()

	renderTemplate(w, "time-table", timeTable)
}
