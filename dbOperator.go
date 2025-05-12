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

// 動的情報用のテーブルたちを作成
func createTablesOnDynamicDb(db *sql.DB) error {
	// response_header テーブル作成
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS response_header (
			timestamp TEXT NOT NULL,
			gtfs_realtime_version TEXT,
			incrementality TEXT,
			response_type TEXT NOT NULL CHECK (response_type IN ('trip_update', 'vehicle_position')),
			PRIMARY KEY (timestamp, response_type)
		)
	`)
	if err != nil {
		return fmt.Errorf("response_headerテーブル作成に失敗: %w", err)
	}
	fmt.Println("response_headerテーブルを作成または確認しました。")

	// entity テーブル作成
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS entity (
			id TEXT PRIMARY KEY NOT NULL,
			response_timestamp TEXT NOT NULL,
			response_type TEXT NOT NULL,
			FOREIGN KEY (response_timestamp, response_type) REFERENCES response_header(timestamp, response_type)
		)
	`)
	if err != nil {
		return fmt.Errorf("entityテーブル作成に失敗: %w", err)
	}
	fmt.Println("entityテーブルを作成または確認しました。")

	// trip テーブル作成
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS trip (
			trip_id TEXT PRIMARY KEY NOT NULL,
			route_id TEXT NOT NULL,
			start_date TEXT NOT NULL,
			start_time TEXT NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("tripテーブル作成に失敗: %w", err)
	}
	fmt.Println("tripテーブルを作成または確認しました。")

	// vehicle テーブル作成
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS vehicle (
			id TEXT PRIMARY KEY NOT NULL,
			label TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("vehicleテーブル作成に失敗: %w", err)
	}
	fmt.Println("vehicleテーブルを作成または確認しました。")

	// trip_update テーブル作成
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS trip_update (
			entity_id TEXT NOT NULL,
			trip_id TEXT NOT NULL,
			vehicle_id TEXT,
			FOREIGN KEY (entity_id) REFERENCES entity(id),
			FOREIGN KEY (trip_id) REFERENCES trip(trip_id),
			FOREIGN KEY (vehicle_id) REFERENCES vehicle(id),
			PRIMARY KEY (entity_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("trip_updateテーブル作成に失敗: %w", err)
	}
	fmt.Println("trip_updateテーブルを作成または確認しました。")

	// stop_time_update テーブル作成
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS stop_time_update (
			trip_update_entity_id TEXT NOT NULL,
			stop_sequence INTEGER NOT NULL,
			arrival_delay INTEGER,
			arrival_uncertainty INTEGER,
			departure_delay INTEGER,
			departure_uncertainty INTEGER,
			stop_id TEXT NOT NULL,
			PRIMARY KEY (trip_update_entity_id, stop_sequence),
			FOREIGN KEY (trip_update_entity_id) REFERENCES trip_update(entity_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("stop_time_updateテーブル作成に失敗: %w", err)
	}
	fmt.Println("stop_time_updateテーブルを作成または確認しました。")

	// vehicle_position テーブル作成
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS vehicle_position (
			entity_id TEXT NOT NULL,
			current_stop_sequence INTEGER,
			latitude REAL,
			longitude REAL,
			speed REAL,
			stop_id TEXT,
			position_timestamp TEXT,
			trip_id TEXT,
			vehicle_id TEXT,
			FOREIGN KEY (entity_id) REFERENCES entity(id),
			FOREIGN KEY (trip_id) REFERENCES trip(trip_id),
			FOREIGN KEY (vehicle_id) REFERENCES vehicle(id),
			PRIMARY KEY (entity_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("vehicle_positionテーブル作成に失敗: %w", err)
	}
	fmt.Println("vehicle_positionテーブルを作成または確認しました。")

	return nil
}

// TripUpdateResponseのデータをテーブルに挿入
func insertTripUpdateResponse(db *sql.DB, response *TripUpdateResponse) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("トランザクション開始に失敗: %w", err)
	}
	defer tx.Rollback() // エラーが発生した場合にロールバック

	// response_header への挿入
	_, err = tx.Exec(`
		INSERT INTO response_header (timestamp, gtfs_realtime_version, incrementality, response_type)
		VALUES (?, ?, ?, ?)
	`, response.Header.Timestamp, response.Header.GtfsRealtimeVersion, response.Header.Incrementality, "trip_update")
	if err != nil {
		return fmt.Errorf("response_header への挿入に失敗: %w", err)
	}

	for _, entity := range response.Entity {
		// entity への挿入
		_, err = tx.Exec(`
			INSERT INTO entity (id, response_timestamp, response_type)
			VALUES (?, ?, ?)
		`, entity.ID, response.Header.Timestamp, "trip_update")
		if err != nil {
			return fmt.Errorf("entity への挿入に失敗 (ID: %s): %w", entity.ID, err)
		}

		if entity.TripUpdate != nil {
			// trip への挿入 (存在しない場合のみ)
			_, err = tx.Exec(`
				INSERT OR IGNORE INTO trip (trip_id, route_id, start_date, start_time)
				VALUES (?, ?, ?, ?)
			`, entity.TripUpdate.Trip.TripID, entity.TripUpdate.Trip.RouteID, entity.TripUpdate.Trip.StartDate, entity.TripUpdate.Trip.StartTime)
			if err != nil {
				return fmt.Errorf("trip への挿入に失敗 (TripID: %s): %w", entity.TripUpdate.Trip.TripID, err)
			}

			// vehicle への挿入 (存在しない場合のみ)
			if entity.TripUpdate.Vehicle != nil {
				_, err = tx.Exec(`
					INSERT OR IGNORE INTO vehicle (id, label)
					VALUES (?, ?)
				`, entity.TripUpdate.Vehicle.ID, entity.TripUpdate.Vehicle.Label)
				if err != nil {
					return fmt.Errorf("vehicle への挿入に失敗 (ID: %s): %w", entity.TripUpdate.Vehicle.ID, err)
				}
			}

			// trip_update への挿入
			_, err = tx.Exec(`
				INSERT INTO trip_update (entity_id, trip_id, vehicle_id)
				VALUES (?, ?, ?)
			`, entity.ID, entity.TripUpdate.Trip.TripID, entity.TripUpdate.Vehicle.ID)
			if err != nil {
				return fmt.Errorf("trip_update への挿入に失敗 (EntityID: %s): %w", entity.ID, err)
			}

			// stop_time_update への挿入
			for _, stu := range entity.TripUpdate.StopTimeUpdate {
				_, err = tx.Exec(`
					INSERT INTO stop_time_update (trip_update_entity_id, stop_sequence, arrival_delay, arrival_uncertainty, departure_delay, departure_uncertainty, stop_id)
					VALUES (?, ?, ?, ?, ?, ?, ?)
				`, entity.ID, stu.StopSequence, stu.Arrival.Delay, stu.Arrival.Uncertainty, stu.Departure.Delay, stu.Departure.Uncertainty, stu.StopID)
				if err != nil {
					return fmt.Errorf("stop_time_update への挿入に失敗 (EntityID: %s, StopSequence: %d): %w", entity.ID, stu.StopSequence, err)
				}
			}
		}
	}

	return tx.Commit()
}

// VehiclePositionResponseのデータをテーブルに挿入
func insertVehiclePositionResponse(db *sql.DB, response *VehiclePositionResponse) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("トランザクション開始に失敗: %w", err)
	}
	defer tx.Rollback() // エラーが発生した場合にロールバック

	if len(response.Entity) > 0 && response.Entity[0].Vehicle != nil {
		// response_header への挿入
		_, err = tx.Exec(`
			INSERT INTO response_header (timestamp, response_type)
			VALUES (?, ?)
		`, response.Entity[0].Vehicle.Timestamp, "vehicle_position") // Vehicle の Timestamp を使用
		if err != nil {
			return fmt.Errorf("response_header への挿入に失敗: %w", err)
		}
	} else {
		fmt.Println("警告: response.Entity が空または最初の要素に Vehicle がありません。response_header への挿入をスキップします。")
		// ここで return nil とするか、エラーを返すかはアプリケーションの要件によります。
		// エラーを返す場合は、以下のようにします。
		// return fmt.Errorf("response.Entity が空または最初の要素に Vehicle がありません")
	}

	for _, entity := range response.Entity {
		if entity.Vehicle != nil {
			// entity への挿入
			_, err = tx.Exec(`
				INSERT INTO entity (id, response_timestamp, response_type)
				VALUES (?, ?, ?)
			`, entity.ID, entity.Vehicle.Timestamp, "vehicle_position")
			if err != nil {
				return fmt.Errorf("entity への挿入に失敗 (ID: %s): %w", entity.ID, err)
			}

			// trip への挿入 (存在しない場合のみ)
			_, err = tx.Exec(`
				INSERT OR IGNORE INTO trip (trip_id, route_id, start_date, start_time)
				VALUES (?, ?, ?, ?)
			`, entity.Vehicle.Trip.TripID, entity.Vehicle.Trip.RouteID, entity.Vehicle.Trip.StartDate, entity.Vehicle.Trip.StartTime)
			if err != nil {
				return fmt.Errorf("trip への挿入に失敗 (TripID: %s): %w", entity.Vehicle.Trip.TripID, err)
			}

			// vehicle への挿入 (存在しない場合のみ)
			_, err = tx.Exec(`
				INSERT OR IGNORE INTO vehicle (id, label)
				VALUES (?, ?)
			`, entity.Vehicle.ID, entity.Vehicle.Label)
			if err != nil {
				return fmt.Errorf("vehicle への挿入に失敗 (ID: %s): %w", entity.Vehicle.ID, err)
			}

			// vehicle_position への挿入
			_, err = tx.Exec(`
				INSERT INTO vehicle_position (entity_id, current_stop_sequence, latitude, longitude, speed, stop_id, position_timestamp, trip_id, vehicle_id)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, entity.ID, entity.Vehicle.CurrentStopSequence, entity.Vehicle.Position.Latitude, entity.Vehicle.Position.Longitude, entity.Vehicle.Position.Speed, entity.Vehicle.StopID, entity.Vehicle.Timestamp, entity.Vehicle.Trip.TripID, entity.Vehicle.ID)
			if err != nil {
				return fmt.Errorf("vehicle_position への挿入に失敗 (EntityID: %s): %w", entity.ID, err)
			}
		}
	}

	return tx.Commit()
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

func initDynamicDb(dbFile string) {

	db, err := setupDb(dbFile)
	if err != nil {
		log.Fatalf("データベース接続に失敗: %v", err)
	}
	defer db.Close()

	createTablesOnDynamicDb(db)

	vehiclePosData := fetchVehiclePosition()
	tripUpdateData := fetchTripUpdate()

	insertVehiclePositionResponse(db, vehiclePosData)
	insertTripUpdateResponse(db, tripUpdateData)
}
