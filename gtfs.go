package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// 車両の位置情報を格納する構造体
type VehiclePositionResponse struct {
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

// 更新情報を格納する構造体
type TripUpdateResponse struct {
	Entity []struct {
		ID         string `json:"id"`
		TripUpdate *struct {
			StopTimeUpdate []struct {
				Arrival struct {
					Delay       int `json:"delay"`
					Uncertainty int `json:"uncertainty"`
				} `json:"arrival"`
				Departure struct {
					Delay       int `json:"delay"`
					Uncertainty int `json:"uncertainty"`
				} `json:"departure"`
				StopID       string `json:"stopId"`
				StopSequence int    `json:"stopSequence"`
			} `json:"stopTimeUpdate"`
			Trip struct {
				RouteID   string `json:"routeId"`
				StartDate string `json:"startDate"`
				StartTime string `json:"startTime"`
				TripID    string `json:"tripId"`
			} `json:"trip"`
			Vehicle *struct {
				ID    string `json:"id"`
				Label string `json:"label"`
			} `json:"vehicle"`
		} `json:"tripUpdate"`
	} `json:"entity"`
	Header struct {
		GtfsRealtimeVersion string `json:"gtfsRealtimeVersion"`
		Incrementality      string `json:"incrementality"`
		Timestamp           string `json:"timestamp"`
	} `json:"header"`
}

func fetchVehiclePosition() *VehiclePositionResponse {

	config := readConfig(configPath)
	var apiURL string = "https://www.ptd-hs.jp/GetVehiclePosition?agency_id=" + config.agencyID + "&uid=" + config.uid + "&output=json"

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

	var data VehiclePositionResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		fmt.Println("Failed to decode JSON format:", err)
		//fmt.Println("Response:", string(body))
		os.Exit(1)
	}

	return &data
}

func fetchTripUpdate() *TripUpdateResponse {

	config := readConfig(configPath)
	var apiURL string = "https://www.ptd-hs.jp/GetTripUpdate?agency_id=" + config.agencyID + "&uid=" + config.uid + "&output=json"

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

	var data TripUpdateResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		fmt.Println("Failed to decode JSON format:", err)
		//fmt.Println("Response:", string(body))
		os.Exit(1)
	}

	return &data
}
