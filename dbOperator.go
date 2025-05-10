package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
)

type Route struct {
	RouteID         string
	AgencyID        string
	RouteShortName  string
	RouteLongName   string
	RouteDesc       string
	RouteType       string
	RouteURL        string
	RouteColor      string
	RouteTextColor  string
	JpParentRouteID string
}

type Stop struct {
	StopID             string
	StopCode           string
	StopName           string
	StopDesc           string
	StopLat            float64
	StopLon            float64
	ZoneID             string
	StopURL            string
	LocationType       int
	ParentStation      string
	StopTimezone       string
	WheelchairBoarding string
}

type StopTime struct {
	TripID            string
	ArrivalTime       string
	DepartureTime     string
	StopID            string
	StopSequence      int
	StopHeadsign      string
	PickupType        string
	DropOffType       string
	ShapeDistTraveled string
	Timepoint         string
}

// SQLite3データベースに接続し、必要であればディレクトリとテーブルを作成する関数
func setupDb(dbFile string) (*sql.DB, error) {
	// データベースディレクトリが存在しない場合は作成
	err := os.MkdirAll("./databases", os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("データベースディレクトリの作成に失敗: %w", err)
	}

	dbPath := fmt.Sprintf("./databases/%s", dbFile)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("データベース接続に失敗: %w", err)
	}
	return db, nil
}

// vehicle_posテーブルを作成
func createVehiclePosTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS vehicle_pos (
			id TEXT PRIMARY KEY,
			route_id TEXT,
			start_date TEXT,
			start_time TEXT,
			trip_id TEXT,
			current_stop_sequence INTEGER,
			latitude REAL,
			longitude REAL,
			speed REAL,
			stop_id TEXT,
			timestamp TEXT,
			label TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("vehicle_posテーブル作成に失敗: %w", err)
	}
	fmt.Println("vehicle_posテーブルを作成または確認しました。")
	return nil
}

// routesテーブルを作成
func createRoutesTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS routes (
			route_id TEXT,
			agency_id TEXT,
			route_short_name TEXT,
			route_long_name TEXT,
			route_desc TEXT,
			route_type TEXT,
			route_url TEXT,
			route_color TEXT,
			route_text_color TEXT,
			jp_parent_route_id TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("routesテーブル作成に失敗: %w", err)
	}
	fmt.Println("routesテーブルを作成または確認しました。")
	return nil
}

// stopsテーブルを作成
func createStopsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS stops (
			stop_id TEXT,
			stop_code TEXT,
			stop_name TEXT,
			stop_desc TEXT,
			stop_lat REAL,
			stop_lon REAL,
			zone_id TEXT,
			stop_url TEXT,
			location_type INTEGER,
			parent_station TEXT,
			stop_timezone TEXT,
			wheelchair_boarding TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("stopsテーブル作成に失敗: %w", err)
	}
	fmt.Println("stopsテーブルを作成または確認しました。")
	return nil
}

// stop_timesテーブルを作成
func createStopTimesTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS stop_times (
			trip_id TEXT,
			arrival_time TEXT,
			departure_time TEXT,
			stop_id	TEXT,
			stop_sequence INTEGER,
			stop_headsign TEXT,
			pickup_type TEXT,
			drop_off_type TEXT,
			shape_dist_traveled TEXT,
			timepoint TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("stop_timesテーブル作成に失敗: %w", err)
	}
	fmt.Println("stops_timesテーブルを作成または確認しました。")
	return nil
}

// vehiclePos構造体のデータをvehicle_posテーブルに挿入
func insertVehiclePos(tx *sql.Tx, entity *struct {
	ID      string `json:"id"`
	Vehicle *struct {
		CurrentStopSequence int64 `json:"currentStopSequence"`
		Position            struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Speed     float64 `json:"speed"`
		} `json:"position"`
		StopID    string `json:"stopId"`
		Timestamp string `json:"timestamp"`
		Trip      struct {
			RouteID   string `json:"routeId"`
			StartDate string `json:"startDate"`
			StartTime string `json:"startTime"`
			TripID    string `json:"tripId"`
		} `json:"trip"`
		ID    string `json:"id"`
		Label string `json:"label"`
	} `json:"vehicle,omitempty"`
}) error {
	if entity.Vehicle == nil {
		return nil // Vehicle 情報がない場合はスキップ
	}

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO vehicle_pos (
			id, route_id, start_date, start_time, trip_id,
			current_stop_sequence, latitude, longitude, speed,
			stop_id, timestamp, label
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		entity.Vehicle.ID,
		entity.Vehicle.Trip.RouteID,
		entity.Vehicle.Trip.StartDate,
		entity.Vehicle.Trip.StartTime,
		entity.Vehicle.Trip.TripID,
		entity.Vehicle.CurrentStopSequence,
		entity.Vehicle.Position.Latitude,
		entity.Vehicle.Position.Longitude,
		entity.Vehicle.Position.Speed,
		entity.Vehicle.StopID,
		entity.Vehicle.Timestamp,
		entity.Vehicle.Label,
	)
	if err != nil {
		return fmt.Errorf("failed to execute statement for vehicle ID %s: %w", entity.Vehicle.ID, err)
	}
	return nil
}

// Route構造体のデータをroutesテーブルに挿入
func insertRoute(db *sql.DB, route *Route) error {
	_, err := db.Exec(`
		INSERT INTO routes (route_id, agency_id, route_short_name, route_long_name, route_desc, route_type, route_url, route_color, route_text_color, jp_parent_route_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, route.RouteID, route.AgencyID, route.RouteShortName, route.RouteLongName, route.RouteDesc, route.RouteType, route.RouteURL, route.RouteColor, route.RouteTextColor, route.JpParentRouteID)
	if err != nil {
		return fmt.Errorf("routesテーブルへのデータ挿入に失敗: %w", err)
	}
	return nil
}

// Stop構造体のデータをstopsテーブルに挿入
func insertStop(db *sql.DB, stop *Stop) error {
	_, err := db.Exec(`
		INSERT INTO stops (stop_id, stop_code, stop_name, stop_desc, stop_lat, stop_lon, zone_id, stop_url, location_type, parent_station, stop_timezone, wheelchair_boarding)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, stop.StopID, stop.StopCode, stop.StopName, stop.StopDesc, stop.StopLat, stop.StopLon, stop.ZoneID, stop.StopURL, stop.LocationType, stop.ParentStation, stop.StopTimezone, stop.WheelchairBoarding)
	if err != nil {
		return fmt.Errorf("stopsテーブルへのデータ挿入に失敗: %w", err)
	}
	return nil
}

// StopTime構造体のデータをstop_timesテーブルに挿入
func insertStopTime(db *sql.DB, stopTime *StopTime) error {
	_, err := db.Exec(`
		INSERT INTO stop_times (trip_id,arrival_time,departure_time,stop_id,stop_sequence,stop_headsign,pickup_type,drop_off_type,shape_dist_traveled,timepoint)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, stopTime.TripID, stopTime.ArrivalTime, stopTime.DepartureTime, stopTime.StopID, stopTime.StopSequence, stopTime.StopHeadsign, stopTime.PickupType, stopTime.DropOffType, stopTime.ShapeDistTraveled, stopTime.Timepoint)
	if err != nil {
		return fmt.Errorf("stop_timesテーブルへのデータ挿入に失敗: %w", err)
	}
	return nil
}

// routes.txtを処理しdbを操作
func processRoutesFile(db *sql.DB, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("routesファイルオープンに失敗: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ','

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("routesレコードの読み込みに失敗: %w", err)
		}

		route := &Route{
			RouteID:         record[0],
			AgencyID:        record[1],
			RouteShortName:  record[2],
			RouteLongName:   record[3],
			RouteDesc:       record[4],
			RouteType:       record[5],
			RouteURL:        record[6],
			RouteColor:      record[7],
			RouteTextColor:  record[8],
			JpParentRouteID: record[9],
		}

		err = insertRoute(db, route)
		if err != nil {
			log.Printf("routesデータの挿入エラー: %v", err)
		}
	}
	return nil
}

// stops.txtを処理しdbを操作
func processStopsFile(db *sql.DB, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("stopsファイルオープンに失敗: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ','

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("stopsレコードの読み込みに失敗: %w", err)
		}

		var stopLat, stopLon float64
		var locationType int
		stopLat, err = strconv.ParseFloat(record[4], 64)
		if err != nil {
			log.Printf("緯度 '%s' の変換エラー: %v", record[4], err)
			continue // エラーが発生したらこのレコードをスキップ
		}
		stopLon, err = strconv.ParseFloat(record[5], 64)
		if err != nil {
			log.Printf("経度 '%s' の変換エラー: %v", record[5], err)
			continue // エラーが発生したらこのレコードをスキップ
		}
		locationType, err = strconv.Atoi(record[8])
		if err != nil {
			log.Printf("location_type '%s' の変換エラー: %v", record[8], err)
			continue // エラーが発生したらこのレコードをスキップ
		}

		stop := &Stop{
			StopID:             record[0],
			StopCode:           record[1],
			StopName:           record[2],
			StopDesc:           record[3],
			StopLat:            stopLat,
			StopLon:            stopLon,
			ZoneID:             record[6],
			StopURL:            record[7],
			LocationType:       locationType,
			ParentStation:      record[9],
			StopTimezone:       record[10],
			WheelchairBoarding: record[11],
		}

		err = insertStop(db, stop)
		if err != nil {
			log.Printf("stopsデータの挿入エラー: %v", err)
		}
	}
	return nil
}

// stop_times.txtを処理しdbを操作
func processStopTimesFile(db *sql.DB, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("stop_timesファイルオープンに失敗: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ','

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("stop_timesレコードの読み込みに失敗: %w", err)
		}

		stopTime := &StopTime{
			TripID:            record[0],
			ArrivalTime:       record[1],
			DepartureTime:     record[2],
			StopID:            record[3],
			StopSequence:      atoi(record[4]),
			StopHeadsign:      record[5],
			PickupType:        record[6],
			DropOffType:       record[7],
			ShapeDistTraveled: record[8],
			Timepoint:         record[9],
		}

		err = insertStopTime(db, stopTime)
		if err != nil {
			log.Printf("stop_timesデータの挿入エラー: %v", err)
		}
	}
	return nil
}

// routesテーブルを検索し、結果をRouteのスライスで返す
func searchRoutes(db *sql.DB, whereClause string, args ...interface{}) ([]Route, error) {
	query := fmt.Sprintf("SELECT route_id, agency_id, route_short_name, route_long_name, route_desc, route_type, route_url, route_color, route_text_color, jp_parent_route_id FROM routes WHERE %s", whereClause)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("routesテーブルの検索に失敗: %w", err)
	}
	defer rows.Close()

	var routes []Route
	for rows.Next() {
		var r Route
		if err := rows.Scan(&r.RouteID, &r.AgencyID, &r.RouteShortName, &r.RouteLongName, &r.RouteDesc, &r.RouteType, &r.RouteURL, &r.RouteColor, &r.RouteTextColor, &r.JpParentRouteID); err != nil {
			return nil, fmt.Errorf("routesテーブルの行のスキャンに失敗: %w", err)
		}
		routes = append(routes, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("routesテーブルの検索結果の処理中にエラーが発生: %w", err)
	}

	return routes, nil
}

// stopsテーブルを検索し、結果をStopのスライスで返す
func searchStops(db *sql.DB, whereClause string, args ...interface{}) ([]Stop, error) {
	query := fmt.Sprintf("SELECT stop_id, stop_code, stop_name, stop_desc, stop_lat, stop_lon, zone_id, stop_url, location_type, parent_station, stop_timezone, wheelchair_boarding FROM stops WHERE %s", whereClause)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("stopsテーブルの検索に失敗: %w", err)
	}
	defer rows.Close()

	var stops []Stop
	for rows.Next() {
		var s Stop
		if err := rows.Scan(&s.StopID, &s.StopCode, &s.StopName, &s.StopDesc, &s.StopLat, &s.StopLon, &s.ZoneID, &s.StopURL, &s.LocationType, &s.ParentStation, &s.StopTimezone, &s.WheelchairBoarding); err != nil {
			return nil, fmt.Errorf("stopsテーブルの行のスキャンに失敗: %w", err)
		}
		stops = append(stops, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("stopsテーブルの検索結果の処理中にエラーが発生: %w", err)
	}

	return stops, nil
}

// stopsテーブルを検索し、結果をStopのスライスで返す
func searchStopTimes(db *sql.DB, whereClause string, args ...interface{}) ([]StopTime, error) {
	query := fmt.Sprintf("SELECT trip_id, arrival_time, departure_time, stop_id, stop_sequence, stop_headsign, pickup_type, drop_off_type, shape_dist_traveled, timepoint FROM stop_times WHERE %s", whereClause)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("stop_timesテーブルの検索に失敗: %w", err)
	}
	defer rows.Close()

	var stopTimes []StopTime
	for rows.Next() {
		var st StopTime
		if err := rows.Scan(&st.TripID, &st.ArrivalTime, &st.DepartureTime, &st.StopID, &st.StopSequence, &st.StopHeadsign, &st.PickupType, &st.DropOffType, &st.ShapeDistTraveled, &st.Timepoint); err != nil {
			return nil, fmt.Errorf("stop_timesテーブルの行のスキャンに失敗: %w", err)
		}
		stopTimes = append(stopTimes, st)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("stop_timesテーブルの検索結果の処理中にエラーが発生: %w", err)
	}

	return stopTimes, nil
}

// 文字列をintに安全に変換するヘルパー関数
func atoi(s string) int {
	i := 0
	_, err := fmt.Sscan(s, &i)
	if err != nil {
		log.Printf("文字列 '%s' を int に変換できませんでした: %v", s, err)
		return 0
	}
	return i
}

func searchDb(dbFile string) {
	// DB検索
	db, err := setupDb(dbFile)
	if err != nil {
		log.Fatalf("データベース接続に失敗: %v", err)
	}
	defer db.Close()

	// routesテーブルの検索例
	fmt.Println("\nroutesテーブルの検索例:")
	routes, err := searchRoutes(db, "route_type = ?", 3)
	if err != nil {
		log.Fatalf("routesテーブルの検索エラー: %v", err)
	}
	for _, route := range routes {
		fmt.Printf("Route ID: %s, Long Name: %s\n", route.RouteID, route.RouteLongName)
	}

	// stopsテーブルの検索例
	fmt.Println("\nstop_timesテーブルの検索例:")
	stops, err := searchStops(db, "stop_name LIKE ?", "%福島%")
	if err != nil {
		log.Fatalf("stopsテーブルの検索エラー: %v", err)
	}
	for _, stop := range stops {
		fmt.Printf("Stop ID: %s, Name: %s, Latitude: %f, Longitude: %f\n", stop.StopID, stop.StopName, stop.StopLat, stop.StopLon)
	}

	// stop_timesテーブルの検索例
	fmt.Println("\nstop_timesテーブルの検索例:")
	stopTimes, err := searchStopTimes(db, "stop_sequence = ?", "1")
	if err != nil {
		log.Fatalf("stop_timesテーブルの検索エラー: %v", err)
	}
	for _, stopTime := range stopTimes {
		fmt.Printf("Stop Sequence: %d, Stop ID: %s, Deaprture Time: %s\n", stopTime.StopSequence, stopTime.StopID, stopTime.DepartureTime)
	}
}

func initStaticDb(dbFile string) {

	// 既存のdbを削除
	os.Remove(dbFile)

	// routes.txt の処理
	routesFile := "./static/gtfs/routes.txt"
	db, err := setupDb(dbFile)
	if err != nil {
		log.Fatalf("データベース接続に失敗: %v", err)
	}
	defer db.Close()

	err = createRoutesTable(db)
	if err != nil {
		log.Fatalf("routesテーブルの作成に失敗: %v", err)
	}

	err = processRoutesFile(db, routesFile)
	if err != nil {
		log.Fatalf("routesファイルの処理に失敗: %v", err)
	}
	fmt.Println("routesデータの登録が完了しました。")

	// stops.txt の処理
	stopsFile := "./static/gtfs/stops.txt"
	err = createStopsTable(db)
	if err != nil {
		log.Fatalf("stopsテーブルの作成に失敗: %v", err)
	}

	err = processStopsFile(db, stopsFile)
	if err != nil {
		log.Fatalf("stopsファイルの処理に失敗: %v", err)
	}
	fmt.Println("stopsデータの登録が完了しました。")

	// stop_times.txt の処理
	stopTimesFile := "./static/gtfs/stop_times.txt"
	err = createStopTimesTable(db)
	if err != nil {
		log.Fatalf("stop_timesテーブルの作成に失敗: %v", err)
	}

	err = processStopTimesFile(db, stopTimesFile)
	if err != nil {
		log.Fatalf("stop_timesファイルの処理に失敗: %v", err)
	}
	fmt.Println("stop_timesデータの登録が完了しました。")
}

func testDynamicDb(dbFile string, data ResponseData) {

	db, err := setupDb(dbFile)
	if err != nil {
		log.Fatalf("データベース接続に失敗: %v", err)
	}
	defer db.Close()

	err = createVehiclePosTable(db)
	if err != nil {
		log.Fatalf("routesテーブルの作成に失敗: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		fmt.Println("Failed to begin transaction:", err)
		return
	}
	defer tx.Rollback() // エラーが発生した場合にロールバック

	// 取得した各 Entity を挿入
	for _, entity := range data.Entity {
		err := insertVehiclePos(tx, &entity) // Entity へのポインタを渡す
		if err != nil {
			fmt.Println("Error inserting/updating vehicle:", err)
			return // エラーが発生したら処理を中断 (必要に応じて継続することも可能)
		}
	}

	// トランザクションのコミット
	err = tx.Commit()
	if err != nil {
		fmt.Println("Failed to commit transaction:", err)
		return
	}
}
