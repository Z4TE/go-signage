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

type CalendarDate struct {
	ServiceID     string
	Date          string
	ExceptionType string
}

type Calendar struct {
	ServiceID string
	Monday    int
	Tuesday   int
	Wednesday int
	Thursday  int
	Friday    int
	Saturday  int
	Sunday    int
	StartDate string
	EndDate   string
}

type FareAttribute struct {
	FareID           string
	Price            int
	CurrencyType     string
	PaymentMethod    int
	Transfers        int
	AgencyID         string
	TransferDuration string
}

type FareRule struct {
	FareID        string
	RouteID       string
	OriginID      string
	DestinationID string
	ContainsID    string
}

type FeedInfo struct {
	FeedPublisherName string
	FeedPublisherURL  string
	FeedLang          string
	FeedStartDate     string
	FeedEndDate       string
	FeedVersion       string
}

type OfficeJP struct {
	OfficeID    int
	OfficeName  string
	OfficeURL   string
	OfficePhone string
}

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
	JPParentRouteID string
}

type Shape struct {
	ShapeID           string
	ShapePtLat        float64
	ShapePtLon        float64
	ShapePtSequence   int
	ShapeDistTraveled string
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

type Translation struct {
	TableName   string
	FieldName   string
	Language    string
	Translation string
	RecordID    string
	RecordSubID string
	FieldValue  string
}

type Trip struct {
	RouteID              string
	ServiceID            string
	TripID               string
	TripHeadsign         string
	TripShortName        string
	DirectionID          string
	BlockID              string
	ShapeID              string
	WheelchairAccessible int
	BikesAllowed         int
	JPTripDesc           string
	JPTripDescSymbol     string
	JPOfficeID           int
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
func createDynamicTables(db *sql.DB) error {
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

// GTFSの静的情報に関連するテーブルをまとめて作成
func createStaticTables(db *sql.DB) error {
	// calendar_datesテーブルを作成
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS calendar_dates (
			service_id TEXT,
			date TEXT,
			exception_type TEXT,
			PRIMARY KEY (service_id, date)
		)
	`)
	if err != nil {
		return fmt.Errorf("calendar_datesテーブル作成に失敗: %w", err)
	}
	fmt.Println("calendar_datesテーブルを作成または確認しました。")

	// calendarテーブルを作成
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS calendar (
			service_id TEXT PRIMARY KEY,
			monday INTEGER,
			tuesday INTEGER,
			wednesday INTEGER,
			thursday INTEGER,
			friday INTEGER,
			saturday INTEGER,
			sunday INTEGER,
			start_date TEXT,
			end_date TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("calendarテーブル作成に失敗: %w", err)
	}
	fmt.Println("calendarテーブルを作成または確認しました。")

	// fare_attributesテーブルを作成
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS fare_attributes (
			fare_id TEXT PRIMARY KEY,
			price REAL,
			currency_type TEXT,
			payment_method INTEGER,
			transfers INTEGER,
			agency_id TEXT,
			transfer_duration INTEGER
		)
	`)
	if err != nil {
		return fmt.Errorf("fare_attributesテーブル作成に失敗: %w", err)
	}
	fmt.Println("fare_attributesテーブルを作成または確認しました。")

	// fare_rulesテーブルを作成
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS fare_rules (
			fare_id TEXT,
			route_id TEXT,
			origin_id TEXT,
			destination_id TEXT,
			contains_id TEXT,
			FOREIGN KEY (fare_id) REFERENCES fare_attributes(fare_id),
			FOREIGN KEY (route_id) REFERENCES routes(route_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("fare_rulesテーブル作成に失敗: %w", err)
	}
	fmt.Println("fare_rulesテーブルを作成または確認しました。")

	// feed_infoテーブルを作成
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS feed_info (
			feed_publisher_name TEXT,
			feed_publisher_url TEXT,
			feed_lang TEXT,
			feed_start_date TEXT,
			feed_end_date TEXT,
			feed_version TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("feed_infoテーブル作成に失敗: %w", err)
	}
	fmt.Println("feed_infoテーブルを作成または確認しました。")

	// office_jpテーブルを作成
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS office_jp (
			office_id INTEGER PRIMARY KEY,
			office_name TEXT,
			office_url TEXT,
			office_phone TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("office_jpテーブル作成に失敗: %w", err)
	}
	fmt.Println("office_jpテーブルを作成または確認しました。")

	// routesテーブルを作成
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS routes (
			route_id TEXT PRIMARY KEY,
			agency_id TEXT,
			route_short_name TEXT,
			route_long_name TEXT,
			route_desc TEXT,
			route_type INTEGER,
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

	// shapesテーブルを作成
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS shapes (
			shape_id TEXT,
			shape_pt_lat REAL,
			shape_pt_lon REAL,
			shape_pt_sequence INTEGER,
			shape_dist_traveled REAL,
			PRIMARY KEY (shape_id, shape_pt_sequence)
		)
	`)
	if err != nil {
		return fmt.Errorf("shapesテーブル作成に失敗: %w", err)
	}
	fmt.Println("shapesテーブルを作成または確認しました。")

	// stopsテーブルを作成
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS stops (
			stop_id TEXT PRIMARY KEY,
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
			wheelchair_boarding INTEGER
		)
	`)
	if err != nil {
		return fmt.Errorf("stopsテーブル作成に失敗: %w", err)
	}
	fmt.Println("stopsテーブルを作成または確認しました。")

	// stop_timesテーブルを作成
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS stop_times (
			trip_id TEXT,
			arrival_time TEXT,
			departure_time TEXT,
			stop_id TEXT,
			stop_sequence INTEGER,
			stop_headsign TEXT,
			pickup_type INTEGER,
			drop_off_type INTEGER,
			shape_dist_traveled REAL,
			timepoint INTEGER,
			FOREIGN KEY (trip_id) REFERENCES trips(trip_id),
			FOREIGN KEY (stop_id) REFERENCES stops(stop_id),
			PRIMARY KEY (trip_id, stop_sequence)
		)
	`)
	if err != nil {
		return fmt.Errorf("stop_timesテーブル作成に失敗: %w", err)
	}
	fmt.Println("stops_timesテーブルを作成または確認しました。")

	// translationsテーブルを作成
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS translations (
		table_name TEXT,
		field_name TEXT,
		language TEXT,
		translation TEXT,
		record_id TEXT,
		record_sub_id TEXT,
		field_value TEXT,
		PRIMARY KEY (table_name, field_name, language, record_id, record_sub_id, field_value)
	)
	`)
	if err != nil {
		return fmt.Errorf("translationsテーブル作成に失敗: %w", err)
	}
	fmt.Println("translationsテーブルを作成または確認しました。")

	// tripsテーブルを作成
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS trips (
			route_id TEXT,
			service_id TEXT,
			trip_id TEXT PRIMARY KEY,
			trip_headsign TEXT,
			trip_short_name TEXT,
			direction_id INTEGER,
			block_id TEXT,
			shape_id TEXT,
			wheelchair_accessible INTEGER,
			bikes_allowed INTEGER,
			jp_trip_desc TEXT,
			jp_trip_desc_symbol TEXT,
			jp_office_id INTEGER,
			FOREIGN KEY (route_id) REFERENCES routes(route_id),
			FOREIGN KEY (service_id) REFERENCES calendar(service_id),
			FOREIGN KEY (shape_id) REFERENCES shapes(shape_id),
			FOREIGN KEY (jp_office_id) REFERENCES office_jp(office_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("tripsテーブル作成に失敗: %w", err)
	}
	fmt.Println("tripsテーブルを作成または確認しました。")

	return nil
}

// calendar_datesテーブルにデータを挿入
func insertCalendarDate(db *sql.DB, cd *CalendarDate) error {
	_, err := db.Exec(`
		INSERT INTO calendar_dates (service_id, date, exception_type)
		VALUES (?, ?, ?)
	`, cd.ServiceID, cd.Date, cd.ExceptionType)
	if err != nil {
		return fmt.Errorf("calendar_datesテーブルへのデータ挿入に失敗: %w", err)
	}
	return nil
}

// calendarテーブルにデータを挿入
func insertCalendar(db *sql.DB, c *Calendar) error {
	_, err := db.Exec(`
		INSERT INTO calendar (service_id, monday, tuesday, wednesday, thursday, friday, saturday, sunday, start_date, end_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, c.ServiceID, c.Monday, c.Tuesday, c.Wednesday, c.Thursday, c.Friday, c.Saturday, c.Sunday, c.StartDate, c.EndDate)
	if err != nil {
		return fmt.Errorf("calendarテーブルへのデータ挿入に失敗: %w", err)
	}
	return nil
}

// fare_attributesテーブルにデータを挿入
func insertFareAttribute(db *sql.DB, fa *FareAttribute) error {
	_, err := db.Exec(`
		INSERT INTO fare_attributes (fare_id, price, currency_type, payment_method, transfers, agency_id, transfer_duration)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, fa.FareID, fa.Price, fa.CurrencyType, fa.PaymentMethod, fa.Transfers, fa.AgencyID, fa.TransferDuration)
	if err != nil {
		return fmt.Errorf("fare_attributesテーブルへのデータ挿入に失敗: %w", err)
	}
	return nil
}

// fare_rulesテーブルにデータを挿入
func insertFareRule(db *sql.DB, fr *FareRule) error {
	_, err := db.Exec(`
		INSERT INTO fare_rules (fare_id, route_id, origin_id, destination_id, contains_id)
		VALUES (?, ?, ?, ?, ?)
	`, fr.FareID, fr.RouteID, fr.OriginID, fr.DestinationID, fr.ContainsID)
	if err != nil {
		return fmt.Errorf("fare_rulesテーブルへのデータ挿入に失敗: %w", err)
	}
	return nil
}

// feed_infoテーブルにデータを挿入
func insertFeedInfo(db *sql.DB, fi *FeedInfo) error {
	_, err := db.Exec(`
		INSERT INTO feed_info (feed_publisher_name, feed_publisher_url, feed_lang, feed_start_date, feed_end_date, feed_version)
		VALUES (?, ?, ?, ?, ?, ?)
	`, fi.FeedPublisherName, fi.FeedPublisherURL, fi.FeedLang, fi.FeedStartDate, fi.FeedEndDate, fi.FeedVersion)
	if err != nil {
		return fmt.Errorf("feed_infoテーブルへのデータ挿入に失敗: %w", err)
	}
	return nil
}

// office_jpテーブルにデータを挿入
func insertOfficeJP(db *sql.DB, oj *OfficeJP) error {
	_, err := db.Exec(`
		INSERT INTO office_jp (office_id, office_name, office_url, office_phone)
		VALUES (?, ?, ?, ?)
	`, oj.OfficeID, oj.OfficeName, oj.OfficeURL, oj.OfficePhone)
	if err != nil {
		return fmt.Errorf("office_jpテーブルへのデータ挿入に失敗: %w", err)
	}
	return nil
}

// routesテーブルにデータを挿入
func insertRoute(db *sql.DB, r *Route) error {
	_, err := db.Exec(`
		INSERT INTO routes (route_id, agency_id, route_short_name, route_long_name, route_desc, route_type, route_url, route_color, route_text_color, jp_parent_route_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, r.RouteID, r.AgencyID, r.RouteShortName, r.RouteLongName, r.RouteDesc, r.RouteType, r.RouteURL, r.RouteColor, r.RouteTextColor, r.JPParentRouteID)
	if err != nil {
		return fmt.Errorf("routesテーブルへのデータ挿入に失敗: %w", err)
	}
	return nil
}

// shapesテーブルにデータを挿入
func insertShape(db *sql.DB, s *Shape) error {
	_, err := db.Exec(`
		INSERT INTO shapes (shape_id, shape_pt_lat, shape_pt_lon, shape_pt_sequence, shape_dist_traveled)
		VALUES (?, ?, ?, ?, ?)
	`, s.ShapeID, s.ShapePtLat, s.ShapePtLon, s.ShapePtSequence, s.ShapeDistTraveled)
	if err != nil {
		return fmt.Errorf("shapesテーブルへのデータ挿入に失敗: %w", err)
	}
	return nil
}

// stopsテーブルにデータを挿入
func insertStop(db *sql.DB, st *Stop) error {
	_, err := db.Exec(`
		INSERT INTO stops (stop_id, stop_code, stop_name, stop_desc, stop_lat, stop_lon, zone_id, stop_url, location_type, parent_station, stop_timezone, wheelchair_boarding)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, st.StopID, st.StopCode, st.StopName, st.StopDesc, st.StopLat, st.StopLon, st.ZoneID, st.StopURL, st.LocationType, st.ParentStation, st.StopTimezone, st.WheelchairBoarding)
	if err != nil {
		return fmt.Errorf("stopsテーブルへのデータ挿入に失敗: %w", err)
	}
	return nil
}

// stop_timesテーブルにデータを挿入
func insertStopTime(db *sql.DB, st *StopTime) error {
	_, err := db.Exec(`
		INSERT INTO stop_times (trip_id, arrival_time, departure_time, stop_id, stop_sequence, stop_headsign, pickup_type, drop_off_type, shape_dist_traveled, timepoint)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, st.TripID, st.ArrivalTime, st.DepartureTime, st.StopID, st.StopSequence, st.StopHeadsign, st.PickupType, st.DropOffType, st.ShapeDistTraveled, st.Timepoint)
	if err != nil {
		return fmt.Errorf("stop_timesテーブルへのデータ挿入に失敗: %w", err)
	}
	return nil
}

// translationsテーブルにデータを挿入
func insertTranslation(db *sql.DB, t *Translation) error {
	_, err := db.Exec(`
		INSERT INTO translations (table_name, field_name, language, translation, record_id, record_sub_id, field_value)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, t.TableName, t.FieldName, t.Language, t.Translation, t.RecordID, t.RecordSubID, t.FieldValue)
	if err != nil {
		return fmt.Errorf("translationsテーブルへのデータ挿入に失敗: %w", err)
	}
	return nil
}

// tripsテーブルにデータを挿入
func insertTrip(db *sql.DB, tr *Trip) error {
	_, err := db.Exec(`
		INSERT INTO trips (route_id, service_id, trip_id, trip_headsign, trip_short_name, direction_id, block_id, shape_id, wheelchair_accessible, bikes_allowed, jp_trip_desc, jp_trip_desc_symbol, jp_office_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, tr.RouteID, tr.ServiceID, tr.TripID, tr.TripHeadsign, tr.TripShortName, tr.DirectionID, tr.BlockID, tr.ShapeID, tr.WheelchairAccessible, tr.BikesAllowed, tr.JPTripDesc, tr.JPTripDescSymbol, tr.JPOfficeID)
	if err != nil {
		return fmt.Errorf("tripsテーブルへのデータ挿入に失敗: %w", err)
	}
	return nil
}

// calendar_dates.txtを処理しdbを操作
func processCalendarDatesFile(db *sql.DB, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("calendar_datesファイルオープンに失敗: %w", err)
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
			return fmt.Errorf("calendar_datesレコードの読み込みに失敗: %w", err)
		}

		calendarDate := &CalendarDate{
			ServiceID:     record[0],
			Date:          record[1],
			ExceptionType: record[2],
		}

		err = insertCalendarDate(db, calendarDate)
		if err != nil {
			log.Printf("stop_timesデータの挿入エラー: %v", err)
		}
	}
	return nil
}

// calendar.txtを処理しdbを操作
func processCalendarFile(db *sql.DB, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("calendarファイルオープンに失敗: %w", err)
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
			return fmt.Errorf("calendarレコードの読み込みに失敗: %w", err)
		}

		calendar := &Calendar{
			ServiceID: record[0],
			Monday:    atoi(record[1]),
			Tuesday:   atoi(record[2]),
			Wednesday: atoi(record[3]),
			Thursday:  atoi(record[4]),
			Friday:    atoi(record[5]),
			Saturday:  atoi(record[6]),
			Sunday:    atoi(record[7]),
			StartDate: record[8],
			EndDate:   record[9],
		}

		err = insertCalendar(db, calendar)
		if err != nil {
			log.Printf("calendarデータの挿入エラー: %v", err)
		}
	}
	return nil
}

// fare_attributes.txtを処理しdbを操作
func processFareAttributesFile(db *sql.DB, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("fare_attributesファイルオープンに失敗: %w", err)
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
			return fmt.Errorf("fare_attributesレコードの読み込みに失敗: %w", err)
		}

		fareAttribute := &FareAttribute{
			FareID:           record[0],
			Price:            atoi(record[1]),
			CurrencyType:     record[2],
			PaymentMethod:    atoi(record[3]),
			Transfers:        atoi(record[4]),
			AgencyID:         record[5],
			TransferDuration: record[6],
		}

		err = insertFareAttribute(db, fareAttribute)
		if err != nil {
			log.Printf("fare_attributesデータの挿入エラー: %v", err)
		}
	}
	return nil
}

// fare_rules.txtを処理しdbを操作
func processFareRulesFile(db *sql.DB, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("fare_rulesファイルオープンに失敗: %w", err)
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
			return fmt.Errorf("fare_rulesレコードの読み込みに失敗: %w", err)
		}

		fareRule := &FareRule{
			FareID:        record[0],
			RouteID:       record[1],
			OriginID:      record[2],
			DestinationID: record[3],
			ContainsID:    record[4],
		}

		err = insertFareRule(db, fareRule)
		if err != nil {
			log.Printf("fare_attributesデータの挿入エラー: %v", err)
		}
	}
	return nil
}

// feed_info.txtを処理しdbを操作
func processFeedInfoFile(db *sql.DB, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("feed_infoファイルオープンに失敗: %w", err)
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
			return fmt.Errorf("feed_infoレコードの読み込みに失敗: %w", err)
		}

		feedInfo := &FeedInfo{
			FeedPublisherName: record[0],
			FeedPublisherURL:  record[1],
			FeedLang:          record[2],
			FeedStartDate:     record[3],
			FeedEndDate:       record[4],
			FeedVersion:       record[5],
		}

		err = insertFeedInfo(db, feedInfo)
		if err != nil {
			log.Printf("feed_infoデータの挿入エラー: %v", err)
		}
	}
	return nil
}

// office_jp.txtを処理しdbを操作
func processOfficeJPFile(db *sql.DB, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("office_jpファイルオープンに失敗: %w", err)
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
			return fmt.Errorf("office_jpレコードの読み込みに失敗: %w", err)
		}

		officeJP := &OfficeJP{
			OfficeID:    atoi(record[0]),
			OfficeName:  record[1],
			OfficeURL:   record[2],
			OfficePhone: record[3],
		}

		err = insertOfficeJP(db, officeJP)
		if err != nil {
			log.Printf("office_jpデータの挿入エラー: %v", err)
		}
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
			JPParentRouteID: record[9],
		}

		err = insertRoute(db, route)
		if err != nil {
			log.Printf("routesデータの挿入エラー: %v", err)
		}
	}
	return nil
}

// shapes.txtを処理しdbを操作
func processShapesFile(db *sql.DB, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("shapesファイルオープンに失敗: %w", err)
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
			return fmt.Errorf("shapesレコードの読み込みに失敗: %w", err)
		}

		var shapePtLat, shapePtLon float64
		var shapePtSequence int
		shapePtLat, err = strconv.ParseFloat(record[1], 64)
		if err != nil {
			log.Printf("緯度 '%s' の変換エラー: %v", record[1], err)
			continue // エラーが発生したらこのレコードをスキップ
		}
		shapePtLon, err = strconv.ParseFloat(record[2], 64)
		if err != nil {
			log.Printf("経度 '%s' の変換エラー: %v", record[2], err)
			continue // エラーが発生したらこのレコードをスキップ
		}
		shapePtSequence, err = strconv.Atoi(record[3])
		if err != nil {
			log.Printf("shape_pt_sequence '%s' の変換エラー: %v", record[3], err)
			continue // エラーが発生したらこのレコードをスキップ
		}

		shape := &Shape{
			ShapeID:           record[0],
			ShapePtLat:        shapePtLat,
			ShapePtLon:        shapePtLon,
			ShapePtSequence:   shapePtSequence,
			ShapeDistTraveled: record[4],
		}

		err = insertShape(db, shape)
		if err != nil {
			log.Printf("shapesデータの挿入エラー: %v", err)
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

// translations.txtを処理しdbを操作
func processTranslationsFile(db *sql.DB, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("translationsファイルオープンに失敗: %w", err)
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
			return fmt.Errorf("translationsレコードの読み込みに失敗: %w", err)
		}

		translation := &Translation{
			TableName:   record[0],
			FieldName:   record[1],
			Language:    record[2],
			Translation: record[3],
			RecordID:    record[4],
			RecordSubID: record[5],
			FieldValue:  record[6],
		}

		err = insertTranslation(db, translation)
		if err != nil {
			log.Printf("translationsデータの挿入エラー: %v", err)
		}
	}
	return nil
}

// trips.txtを処理しdbを操作
func processTripsFile(db *sql.DB, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("tripsファイルオープンに失敗: %w", err)
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
			return fmt.Errorf("tripsレコードの読み込みに失敗: %w", err)
		}

		trip := &Trip{
			RouteID:              record[0],
			ServiceID:            record[1],
			TripID:               record[2],
			TripHeadsign:         record[3],
			TripShortName:        record[4],
			DirectionID:          record[5],
			BlockID:              record[6],
			ShapeID:              record[7],
			WheelchairAccessible: atoi(record[8]),
			BikesAllowed:         atoi(record[9]),
			JPTripDesc:           record[10],
			JPTripDescSymbol:     record[11],
			JPOfficeID:           atoi(record[12]),
		}

		err = insertTrip(db, trip)
		if err != nil {
			log.Printf("tripsデータの挿入エラー: %v", err)
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

	db, err := setupDb(dbFile)
	if err != nil {
		log.Fatalf("データベース接続に失敗: %v", err)
	}
	defer db.Close()

	err = createStaticTables(db)
	if err != nil {
		log.Fatalf("routesテーブルの作成に失敗: %v", err)
	}

	// calendar_dates.txt の処理
	calendarDatesFile := "./static/gtfs/calendar_dates.txt"
	err = processCalendarDatesFile(db, calendarDatesFile)
	if err != nil {
		log.Fatalf("calendar_datesファイルの処理に失敗: %v", err)
	}
	fmt.Println("calendar_datesデータの登録が完了しました。")

	// calendar.txt の処理
	calendarFile := "./static/gtfs/calendar.txt"
	err = processCalendarFile(db, calendarFile)
	if err != nil {
		log.Fatalf("calendarファイルの処理に失敗: %v", err)
	}
	fmt.Println("calendarデータの登録が完了しました。")

	// fare_attributes.txt の処理
	fareAttributesFile := "./static/gtfs/fare_attributes.txt"
	err = processFareAttributesFile(db, fareAttributesFile)
	if err != nil {
		log.Fatalf("fare_attributeファイルの処理に失敗: %v", err)
	}
	fmt.Println("fare_attributeデータの登録が完了しました。")

	// fare_rules.txt の処理
	fareRulesFile := "./static/gtfs/fare_rules.txt"
	err = processFareRulesFile(db, fareRulesFile)
	if err != nil {
		log.Fatalf("fare_rulesファイルの処理に失敗: %v", err)
	}
	fmt.Println("fare_rulesデータの登録が完了しました。")

	// feed_info.txt の処理
	feedInfoFile := "./static/gtfs/feed_info.txt"
	err = processFeedInfoFile(db, feedInfoFile)
	if err != nil {
		log.Fatalf("feed_infoファイルの処理に失敗: %v", err)
	}
	fmt.Println("feed_infoデータの登録が完了しました。")

	// office_jp.txt の処理
	officeJPFile := "./static/gtfs/office_jp.txt"
	err = processOfficeJPFile(db, officeJPFile)
	if err != nil {
		log.Fatalf("office_jpファイルの処理に失敗: %v", err)
	}
	fmt.Println("office_jpデータの登録が完了しました。")

	// routes.txt の処理
	routesFile := "./static/gtfs/routes.txt"
	err = processRoutesFile(db, routesFile)
	if err != nil {
		log.Fatalf("routesファイルの処理に失敗: %v", err)
	}
	fmt.Println("routesデータの登録が完了しました。")

	// shapes.txt の処理
	shapesFile := "./static/gtfs/shapes.txt"
	err = processShapesFile(db, shapesFile)
	if err != nil {
		log.Fatalf("shapesファイルの処理に失敗: %v", err)
	}
	fmt.Println("shapesデータの登録が完了しました。")

	// stops.txt の処理
	stopsFile := "./static/gtfs/stops.txt"
	err = processStopsFile(db, stopsFile)
	if err != nil {
		log.Fatalf("stopsファイルの処理に失敗: %v", err)
	}
	fmt.Println("stopsデータの登録が完了しました。")

	// stop_times.txt の処理
	stopTimesFile := "./static/gtfs/stop_times.txt"
	err = processStopTimesFile(db, stopTimesFile)
	if err != nil {
		log.Fatalf("stop_timesファイルの処理に失敗: %v", err)
	}
	fmt.Println("stop_timesデータの登録が完了しました。")

	// translations.txt の処理
	translationsFile := "./static/gtfs/translations.txt"
	err = processTranslationsFile(db, translationsFile)
	if err != nil {
		log.Fatalf("translationsファイルの処理に失敗: %v", err)
	}
	fmt.Println("translationsデータの登録が完了しました。")

	// trips.txt の処理
	tripsFile := "./static/gtfs/trips.txt"
	err = processTripsFile(db, tripsFile)
	if err != nil {
		log.Fatalf("tripsファイルの処理に失敗: %v", err)
	}
	fmt.Println("tripsデータの登録が完了しました。")
}

func initDynamicDb(dbFile string) {

	db, err := setupDb(dbFile)
	if err != nil {
		log.Fatalf("データベース接続に失敗: %v", err)
	}
	defer db.Close()

	createDynamicTables(db)

	vehiclePosData := fetchVehiclePosition()
	tripUpdateData := fetchTripUpdate()

	insertVehiclePositionResponse(db, vehiclePosData)
	insertTripUpdateResponse(db, tripUpdateData)
}
