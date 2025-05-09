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
		} `json:"vehicle,omitempty"` // omitempty を追加して、vehicle がない場合もエラーにならないように
	} `json:"entity"`
}

// 時刻表に掲載する情報の構造体
type TimeTable struct {
	Entity []struct {
		Vehicle struct {
			RouteID             string `json:"routeId"`
			CurrentStopSequence string `json:"currentStopSequence"`
			NextStopID          string `json:"nextStopId"`
			NextStopTime        string `json:"nextStopTime"`
		}
	}
}

func fetchStatus(apiURL string) *ResponseData {

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

	return &data
}
