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

// stopsテーブルを作成する関数
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

// stop_timesテーブルを作成する関数
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
