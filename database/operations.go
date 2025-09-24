package database

import (
	"database/sql"
	"fmt"
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

// InsertResponse inserta una nueva response en la base de datos
func InsertResponse(db *sql.DB, requestID int64, statusCode int, headers, body string) error {
	query := `
		INSERT INTO proxy_responses (request_id, status_code, headers, body, timestamp)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := db.Exec(query, requestID, statusCode, headers, body, time.Now())
	if err != nil {
		return fmt.Errorf("error inserting response: %v", err)
	}

	return nil
}

// GetRequest obtiene una request por ID
func GetRequest(db *sql.DB, id int64) (*ProxyRequest, error) {
	query := `
		SELECT id, method, url, headers, body, timestamp
		FROM proxy_requests
		WHERE id = ?
	`

	row := db.QueryRow(query, id)

	var request ProxyRequest
	err := row.Scan(&request.ID, &request.Method, &request.URL, &request.Headers, &request.Body, &request.Timestamp)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("request not found")
		}
		return nil, fmt.Errorf("error scanning request: %v", err)
	}

	return &request, nil
}

// GetResponse obtiene una response por request ID
func GetResponse(db *sql.DB, requestID int64) (*ProxyResponse, error) {
	query := `
		SELECT id, request_id, status_code, headers, body, timestamp
		FROM proxy_responses
		WHERE request_id = ?
	`

	row := db.QueryRow(query, requestID)

	var response ProxyResponse
	err := row.Scan(&response.ID, &response.RequestID, &response.StatusCode, &response.Headers, &response.Body, &response.Timestamp)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("response not found")
		}
		return nil, fmt.Errorf("error scanning response: %v", err)
	}

	return &response, nil
}

// GetTransaction obtiene una transacción completa (request + response)
func GetTransaction(db *sql.DB, requestID int64) (*ProxyTransaction, error) {
	request, err := GetRequest(db, requestID)
	if err != nil {
		return nil, fmt.Errorf("error getting request: %v", err)
	}

	response, err := GetResponse(db, requestID)
	if err != nil {
		return nil, fmt.Errorf("error getting response: %v", err)
	}

	return &ProxyTransaction{
		Request:  *request,
		Response: *response,
	}, nil
}

// GetAllRequests obtiene todas las requests con paginación
func GetAllRequests(db *sql.DB, limit, offset int) ([]ProxyRequest, error) {
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

	var requests []ProxyRequest
	for rows.Next() {
		var request ProxyRequest
		err := rows.Scan(&request.ID, &request.Method, &request.URL, &request.Headers, &request.Body, &request.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("error scanning request: %v", err)
		}
		requests = append(requests, request)
	}

	return requests, nil
}

// GetRequestsByMethod obtiene requests filtradas por método HTTP
func GetRequestsByMethod(db *sql.DB, method string, limit, offset int) ([]ProxyRequest, error) {
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

	var requests []ProxyRequest
	for rows.Next() {
		var request ProxyRequest
		err := rows.Scan(&request.ID, &request.Method, &request.URL, &request.Headers, &request.Body, &request.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("error scanning request: %v", err)
		}
		requests = append(requests, request)
	}

	return requests, nil
}

func GetRequestsByEndpointAndMethod(db *sql.DB, endpoint, method string, limit, offset int) ([]ProxyRequest, error) {
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

	var requests []ProxyRequest
	for rows.Next() {
		var request ProxyRequest
		err := rows.Scan(&request.ID, &request.Method, &request.URL, &request.Headers, &request.Body, &request.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("error scanning request: %v", err)
		}
		requests = append(requests, request)
	}

	return requests, nil
}
