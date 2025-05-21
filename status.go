package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
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
	staticDb, err := setupDb(staticDbFile)
	if err != nil {
		log.Fatalf("データベース接続に失敗: %v", err)
	}
	defer staticDb.Close()

	// トランザクションDBに接続
	dynamicDb, err := setupDb(dynamicDbFile)
	if err != nil {
		log.Fatalf("データベース接続に失敗: %v", err)
	}
	defer dynamicDb.Close()

	// マスタDBにトランザクションDBを接続
	staticDb.Exec("ATTACH DATABASE './databases/dynamic.sql' AS d;")

	vehiclePosData := fetchVehiclePosition()
	tripUpdateData := fetchTripUpdate()

	ExecuteNonQuery(dynamicDb, "DELETE stop_time_update")

	// DBを更新
	insertTripUpdateResponse(staticDb, tripUpdateData)

	// vehiclePosData と tripUpdateData を組み合わせて TimeTable を作成
	count := 0
	for _, vpEntity := range vehiclePosData.Entity {
		if count >= 128 {
			break
		}

		// routeID := vpEntity.Vehicle.Trip.RouteID
		tripID := vpEntity.Vehicle.Trip.TripID
		// stopID := removeLastSymbol("-", vpEntity.Vehicle.StopID)
		// stopSeq := vpEntity.Vehicle.CurrentStopSequence

		// 停留所名から表示する項目を検索
		stopName := "福島駅東口" // あとで変数化する
		// ばかでかいクエリ trip_update_entity_idの最新版を抽出しようとしたらこうなった
		// 有識者による修正求む
		staticDataQuery := `
			WITH ParsedTripUpdate AS (
				SELECT
					st.departure_time,
					st.stop_headsign,
					r.route_long_name,
					stu.departure_delay,
					st.stop_sequence,
					stu.trip_update_entity_id,
					CAST(
						SUBSTR(
							SUBSTR(
								stu.trip_update_entity_id,
								INSTR(stu.trip_update_entity_id, '-') + 1
							),
							INSTR(
								SUBSTR(
									stu.trip_update_entity_id,
									INSTR(stu.trip_update_entity_id, '-') + 1
								),
								'-'
							) + 1,
							INSTR(
								SUBSTR(
									SUBSTR(
										stu.trip_update_entity_id,
										INSTR(stu.trip_update_entity_id, '-') + 1
									),
									INSTR(
										SUBSTR(
											stu.trip_update_entity_id,
											INSTR(stu.trip_update_entity_id, '-') + 1
										),
										'-'
									) + 1
								),
								'-'
							) - 1
						) AS INTEGER
					) AS version_number
				FROM
					stop_times AS st
				JOIN
					stops AS s
					ON st.stop_id = s.stop_id
				JOIN
					trips AS t
					ON st.trip_id = t.trip_id
				JOIN
					routes AS r
					ON t.route_id = r.route_id
				JOIN
					d.stop_time_update AS stu
					ON stu.stop_id = s.stop_id
				JOIN
					d.vehicle_position AS vp
					ON vp.current_stop_sequence = st.stop_sequence
				WHERE
					s.stop_name = ? AND st.trip_id = ?
			)
			SELECT DISTINCT
				departure_time,
				stop_headsign,
				route_long_name,
				departure_delay,
				stop_sequence,
				trip_update_entity_id
			FROM
				ParsedTripUpdate
			WHERE
				version_number = (SELECT MAX(version_number) FROM ParsedTripUpdate)
			LIMIT 128;
		`
		staticData, err := QueryRows(staticDb, staticDataQuery, stopName, tripID)
		if err != nil {
			fmt.Printf("%v\n", err)
		}

		currentTime := time.Now()

		for _, i := range staticData {
			departureTimeStr := i["departure_time"].(string)

			// departure timeの文字列をtime.Timeオブジェクトにパース
			parts := strings.Split(departureTimeStr, ":")
			if len(parts) != 3 {
				fmt.Printf("Invalid departure_time format: %s. Skipping.\n", departureTimeStr)
				continue
			}

			hour, _ := strconv.Atoi(parts[0])
			minute, _ := strconv.Atoi(parts[1])
			second, _ := strconv.Atoi(parts[2])

			// 現在時刻のtime.Timeオブジェクトを作成
			departureTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), hour, minute, second, 0, currentTime.Location())

			// departure_timeが現在以降の場合のみappend
			if departureTime.Before(currentTime) {
				continue
			}

			// 遅延が60秒未満の場合は表示しない
			delay := ""
			if i["departure_delay"].(int64) > 60 {
				delay = fmt.Sprintf("遅れ 約%d分", i["departure_delay"].(int64)/60)
			}

			timeTables = append(timeTables, TimeTable{
				RouteID:       i["route_long_name"].(string),
				Delay:         delay,
				DepartureTime: removeLastSymbol(":", i["departure_time"].(string)),
				Destination:   i["stop_headsign"].(string),
			})
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
	ticker := time.NewTicker(30 * time.Second) // 1分間隔で運行情報を送信
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

	// timeTable := getTimetable()

	renderTemplate(w, "time-table", nil)
}
