package main

import (
	"database/sql"
	"fmt"
	"strings"
)

// SELECTクエリを実行し、結果を []map[string]interface{} で返す。
func QueryRows(db *sql.DB, query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("クエリの実行に失敗しました: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("カラム名の取得に失敗しました: %w", err)
	}

	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		values := make([]interface{}, len(columns))
		scanArgs := make([]interface{}, len(columns))
		for i := range values {
			scanArgs[i] = &values[i]
		}

		err = rows.Scan(scanArgs...)
		if err != nil {
			return nil, fmt.Errorf("スキャンに失敗しました: %w", err)
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// []byte 型の場合、string に変換
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("行の反復処理中にエラーが発生しました: %w", err)
	}

	return results, nil
}

func QuerySingleString(db *sql.DB, query string, args ...interface{}) (string, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return "", fmt.Errorf("クエリの実行に失敗しました: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return "", fmt.Errorf("カラム名の取得に失敗しました: %w", err)
	}

	if len(columns) != 1 {
		return "", fmt.Errorf("クエリの結果は1列である必要があります (現在の列数: %d)", len(columns))
	}

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return "", fmt.Errorf("行の反復処理中にエラーが発生しました: %w", err)
		}
		return "", fmt.Errorf("クエリの結果がありません")
	}

	var result string
	err = rows.Scan(&result)
	if err != nil {
		return "", fmt.Errorf("スキャンに失敗しました: %w", err)
	}

	// 2行以上存在する場合はエラー
	if rows.Next() {
		return "", fmt.Errorf("クエリの結果は1行である必要があります")
	}

	return result, nil
}

// ExecuteNonQuery はSELECT以外のクエリ (INSERT, UPDATE, DELETE など) を実行し、sql.Result を返します。
func ExecuteNonQuery(db *sql.DB, query string, args ...interface{}) (sql.Result, error) {
	result, err := db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("クエリの実行に失敗しました: %w", err)
	}
	return result, nil
}

// 2つ以上-を含む文字列から、最後の-以降を削除する。
func removeLastDash(s string) string {
	lastIndex := strings.LastIndex(s, "-")
	if lastIndex == -1 {
		return s // ハイフンがない場合はそのまま返す
	}

	// 最後のハイフンより前のハイフンの数を数える
	hyphenCount := strings.Count(s[:lastIndex], "-")

	if hyphenCount < 1 {
		return s // ハイフンが1つ以下の場合もそのまま返す
	}

	return s[:lastIndex]
}
