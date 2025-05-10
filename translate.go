package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// GetNameByID関数: 指定されたファイルとカラムに対応するデータを返す。
// filePath: テキストファイルのパス
// officeID: 検索するコード
// colIndex: 取得するカラムのインデックス
// 戻り値: 対応する文字列、見つからない場合は空文字列とエラー

func getItemByID(filePath string, ID string, colIndex int) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("%w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, ",")
		if len(fields) > colIndex {
			idStr := strings.TrimSpace(fields[0])
			if idStr == ID {
				return strings.TrimSpace(fields[colIndex]), nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("%w", err)
	}

	return "", nil
}

func getItemByIDs(routeID string, stopSeq int, targetCol int) (string, error) {

	filePath := "static/gtfs/stop_times.txt"

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("%w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, ",")
		if len(fields) > targetCol {
			routeIdStr := strings.TrimSpace(fields[0])
			stopSeqStr := strconv.Itoa(stopSeq)

			if (routeIdStr == routeID) && (stopSeqStr == strings.TrimSpace(fields[4])) {
				return strings.TrimSpace(fields[targetCol]), nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("%w", err)
	}

	return "", nil
}

func getCurrentStopName(routeID string, stopSeq int) (string, error) {
	// CurrentStopSequence を停留所IDに変換
	stopId, err := getItemByIDs(routeID, stopSeq, 3)
	if err != nil {
		fmt.Printf("Error getting stop name for sequence %d: %v\n", stopSeq, err)
		stopId = ""
	}
	println(stopId)

	// 停留所IDを停留所名に変換
	stopName, err := getItemByID("./static/gtfs/stops.txt", stopId, 2)
	if err != nil {
		fmt.Printf("Error getting stop name for sequence %d: %v\n", stopSeq, err)
		stopName = ""
	}
	println(stopName)

	return stopName, nil
}

func populateTimeTable(responseData *ResponseData) *TimeTable {
	var timeTable TimeTable
	timeTable.Entity = []struct {
		Vehicle struct {
			RouteID             string `json:"routeId"`
			CurrentStopSequence string `json:"currentStopSequence"`
			NextStopID          string `json:"nextStopId"`
			NextStopTime        string `json:"nextStopTime"`
		}
	}{} // スライスを初期化

	for _, entity := range responseData.Entity {
		if entity.Vehicle != nil {
			routeID := entity.Vehicle.Trip.RouteID
			stopSeq := entity.Vehicle.CurrentStopSequence

			stopName, err := getCurrentStopName(routeID, int(stopSeq))
			if err != nil {
				fmt.Printf("Error getting current stop name: %v\n", err)
				stopName = ""
			}

			nextStopName, err := getCurrentStopName(routeID, int(stopSeq)+1)
			if err != nil {
				fmt.Printf("Error getting current stop name: %v\n", err)
				nextStopName = ""
			}

			nextStopTime, err := getItemByIDs(routeID, int(stopSeq)+1, 2)
			if err != nil {
				fmt.Printf("Error getting departute time of the next stop: %v\n", err)
				nextStopName = ""
			}

			// RouteIDを路線名に変換
			routeName, err := getItemByID("./static/gtfs/routes.txt", routeID, 3)
			if err != nil {
				fmt.Printf("Error getting route name for ID %s: %v\n", routeID, err)
				routeName = ""
			}

			timeTable.Entity = append(timeTable.Entity, struct {
				Vehicle struct {
					RouteID             string `json:"routeId"`
					CurrentStopSequence string `json:"currentStopSequence"`
					NextStopID          string `json:"nextStopId"`
					NextStopTime        string `json:"nextStopTime"`
				}
			}{
				Vehicle: struct {
					RouteID             string `json:"routeId"`
					CurrentStopSequence string `json:"currentStopSequence"`
					NextStopID          string `json:"nextStopId"`
					NextStopTime        string `json:"nextStopTime"`
				}{
					RouteID:             routeName,
					CurrentStopSequence: stopName,
					NextStopID:          nextStopName, // CurrentStopSequence を NextStopID として利用（仮）
					NextStopTime:        nextStopTime, // NextStopTime はここでは不明
				},
			})
		}
	}
	return &timeTable
}
