package main

import (
	"fmt"
	"log"
	"time"
)

type FeedDate struct {
	FeedEndDate   string
	FeedStartDate string
}

// checkFeedEndDate は feed_end_date 以前になった場合に処理を実行する関数です
func checkFeedEndDate(dbFile string, processFunc func(string)) {

	db, err := setupDb(dbFile)
	if err != nil {
		log.Fatalf("データベース接続に失敗: %v", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT feed_end_date, feed_start_date FROM feed_info")
	if err != nil {
		log.Printf("クエリの実行に失敗しました: %v", err)
		return
	}
	defer rows.Close()

	now := time.Now()

	for rows.Next() {
		var feedInfo FeedDate
		if err := rows.Scan(&feedInfo.FeedEndDate, &feedInfo.FeedStartDate); err != nil {
			log.Printf("行のスキャンに失敗しました: %v", err)
			continue
		}

		endDate, err := time.Parse("20060102", feedInfo.FeedEndDate) // feed_end_date の形式に合わせてください
		if err != nil {
			log.Printf("feed_end_date のパースに失敗しました: %v", err)
			continue
		}

		// 現在の日付が feed_end_date 以降かどうかを比較（時刻部分は無視）
		if now.Year() > endDate.Year() ||
			(now.Year() == endDate.Year() && now.Month() > endDate.Month()) ||
			(now.Year() == endDate.Year() && now.Month() == endDate.Month() && now.Day() >= endDate.Day()) {
			rows.Close()
			db.Close()
			fmt.Println("現在の日付が feed_end_date 以降です。処理を実行します。")
			processFunc(dbFile)
		}
	}

	if err := rows.Err(); err != nil {
		log.Printf("行の処理中にエラーが発生しました: %v", err)
	}
}
