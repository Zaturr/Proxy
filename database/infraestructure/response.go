package database

import (
	"database/sql"
	"fmt"
	"proxy/database"
	"time"
)

// InsertResponse inserta una nueva response en la base de datos
func InsertResponse(db *sql.DB, requestID int64, statusCode int, headers, body string, port int) error {
	query := `
		INSERT INTO proxy_responses (request_id, status_code, port, headers, body, timestamp)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(query, requestID, statusCode, port, headers, body, time.Now())
	if err != nil {
		return fmt.Errorf("error inserting response: %v", err)
	}

	return nil
}

// GetResponse obtiene una response por request ID
func GetResponse(db *sql.DB, requestID int64) (*database.ProxyResponse, error) {
	query := `
		SELECT id, request_id, status_code, headers, body, timestamp
		FROM proxy_responses
		WHERE request_id = ?
	`

	row := db.QueryRow(query, requestID)

	var response database.ProxyResponse
	err := row.Scan(&response.ID, &response.RequestID, &response.StatusCode, &response.Headers, &response.Body, &response.Timestamp)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("response not found")
		}
		return nil, fmt.Errorf("error scanning response: %v", err)
	}

	return &response, nil
}
