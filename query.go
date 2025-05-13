package main

import (
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func getStopNameByStopSeq(dynamicDbFile string, staticDbFile string, stopSeq int) {

	// DBを開く
	dynamicDb, err := setupDb(dynamicDbFile)
	if err != nil {
		log.Fatalf("データベース接続に失敗: %v", err)
	}
	defer dynamicDb.Close()

	staticDb, err := setupDb(staticDbFile)
	if err != nil {
		log.Fatalf("データベース接続に失敗: %v", err)
	}
	defer staticDb.Close()

	// static.sqlをdynamic.sqlにアタッチ
	_, err = dynamicDb.Exec("ATTACH DATABASE './databases/" + staticDbFile + "' AS static")
	if err != nil {
		log.Fatal(err)
	}

	// アタッチされたテーブルに対してクエリを実行
	rows, err := dynamicDb.Query(`
		SELECT 
			vp.id, s.stop_name
		FROM 
			vehicle_pos vp
		JOIN 
			static.stop_times st ON vp.trip_id = st.trip_id AND vp.current_stop_sequence = st.stop_sequence
		JOIN 
			static.stops s ON st.stop_id = s.stop_id
		WHERE 
			vp.current_stop_sequence = ?;
	`, stopSeq)
	if err != nil {
		log.Fatal(err)
	}

	for rows.Next() {
		var id string
		var stopName string
		if err := rows.Scan(&id, &stopName); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("ID: %s, Stop Name: %s\n", id, stopName)
	}

	defer rows.Close()
}
