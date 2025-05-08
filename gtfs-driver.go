package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type ResponseData struct {
	Entity []struct {
		ID      string `json:"id"`
		Vehicle struct {
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
		} `json:"vehicle,omitempty"` // omitempty を追加して、vehicle がない場合もエラーにならないように
	} `json:"entity"`
}

func fetchStatus(apiURL string) {

	resp, err := http.Get(apiURL)
	if err != nil {
		fmt.Println("HTTP request error:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Failed to read response body:", err)
		os.Exit(1)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Printf("HTTP status error: %d, Response: %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}

	var data ResponseData
	err = json.Unmarshal(body, &data)
	if err != nil {
		fmt.Println("Failed to decode JSON format:", err)
		fmt.Println("Response:", string(body))
		os.Exit(1)
	}

	fmt.Println("Fetched JSON data:")
	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Println("JSON formatting error:", err)
		return
	}
	fmt.Println(string(prettyJSON))

	fmt.Println("\n車両情報:")
	for _, entity := range data.Entity {
		if entity.Vehicle.ID != "" { // Vehicle が存在する場合のみ処理
			fmt.Printf("  車両ID: %s, ラベル: %s\n", entity.Vehicle.ID, entity.Vehicle.Label)
			fmt.Printf("  現在停留所シーケンス: %d\n", entity.Vehicle.CurrentStopSequence)
			fmt.Printf("  位置情報 (緯度: %f, 経度: %f, 速度: %f)\n", entity.Vehicle.Position.Latitude, entity.Vehicle.Position.Longitude, entity.Vehicle.Position.Speed)
			fmt.Printf("  停留所ID: %s, タイムスタンプ: %s\n", entity.Vehicle.StopID, entity.Vehicle.Timestamp)
			fmt.Printf("  トリップ情報 (ルートID: %s, 開始日: %s, 開始時間: %s, トリップID: %s)\n",
				entity.Vehicle.Trip.RouteID, entity.Vehicle.Trip.StartDate, entity.Vehicle.Trip.StartTime, entity.Vehicle.Trip.TripID)
		}
	}
}
