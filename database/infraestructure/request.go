package database

import (
	"database/sql"
	"fmt"
	"proxy/database"
	"time"
)

// InsertRequest inserta una nueva request en la base de datos
func InsertRequest(db *sql.DB, method, url, headers, body string, port int) (int64, error) {
	query := `
		INSERT INTO proxy_requests (method, url, headers, body, port, timestamp)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	result, err := db.Exec(query, method, url, headers, body, port, time.Now())
	if err != nil {
		return 0, fmt.Errorf("error inserting request: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("error getting last insert id: %v", err)
	}

	return id, nil
}

// GetRequest obtiene una request por ID
func GetRequest(db *sql.DB, id int64) (*database.ProxyRequest, error) {
	query := `
		SELECT id, method, url, headers, body, timestamp
		FROM proxy_requests
		WHERE id = ?
	`

	row := db.QueryRow(query, id)

	var request database.ProxyRequest
	err := row.Scan(&request.ID, &request.Method, &request.URL, &request.Headers, &request.Body, &request.Timestamp)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("request not found")
		}
		return nil, fmt.Errorf("error scanning request: %v", err)
	}

	return &request, nil
}

// GetAllRequests obtiene todas las requests con paginación
func GetAllRequests(db *sql.DB, limit, offset int) ([]database.ProxyRequest, error) {
	query := `
		SELECT id, method, url, headers, body, timestamp
		FROM proxy_requests
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?
	`

	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("error querying requests: %v", err)
	}
	defer rows.Close()

	var requests []database.ProxyRequest
	for rows.Next() {
		var request database.ProxyRequest
		err := rows.Scan(&request.ID, &request.Method, &request.URL, &request.Headers, &request.Body, &request.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("error scanning request: %v", err)
		}
		requests = append(requests, request)
	}

	return requests, nil
}

// GetRequestsByMethod obtiene requests filtradas por método HTTP
func GetRequestsByMethod(db *sql.DB, method string, limit, offset int) ([]database.ProxyRequest, error) {
	query := `
		SELECT id, method, url, headers, body, timestamp
		FROM proxy_requests
		WHERE method = ?
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?
	`

	rows, err := db.Query(query, method, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("error querying requests by method: %v", err)
	}
	defer rows.Close()

	var requests []database.ProxyRequest
	for rows.Next() {
		var request database.ProxyRequest
		err := rows.Scan(&request.ID, &request.Method, &request.URL, &request.Headers, &request.Body, &request.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("error scanning request: %v", err)
		}
		requests = append(requests, request)
	}

	return requests, nil
}

// GetRequestsByEndpointAndMethod obtiene requests filtradas por endpoint y método
func GetRequestsByEndpointAndMethod(db *sql.DB, endpoint, method string, limit, offset int) ([]database.ProxyRequest, error) {
	query := `
		SELECT r.id, r.method, r.url, r.headers, r.body, r.timestamp
		FROM proxy_requests r
		WHERE r.url LIKE ? AND r.method = ?
		ORDER BY r.timestamp DESC
		LIMIT ? OFFSET ?
	`

	rows, err := db.Query(query, "%"+endpoint+"%", method, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("error querying requests by endpoint and method: %v", err)
	}
	defer rows.Close()

	var requests []database.ProxyRequest
	for rows.Next() {
		var request database.ProxyRequest
		err := rows.Scan(&request.ID, &request.Method, &request.URL, &request.Headers, &request.Body, &request.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("error scanning request: %v", err)
		}
		requests = append(requests, request)
	}

	return requests, nil
}
