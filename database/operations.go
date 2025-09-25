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

// CalculateChaosProbability calcula la probabilidad de chaos injection basada en status codes >= 300
func CalculateChaosProbability(db *sql.DB, url string) (float64, int, error) {
	// Obtener el conteo de requests exitosos (100-299)
	approvedCounts, err := CountApprovedRequests(db)
	if err != nil {
		return 0, 0, fmt.Errorf("error getting approved requests: %v", err)
	}

	// Obtener el conteo de requests con error (>= 300)
	rejectedCounts, err := CountRejectedRequests(db)
	if err != nil {
		return 0, 0, fmt.Errorf("error getting rejected requests: %v", err)
	}

	approvedCount := approvedCounts[url]
	rejectedCount := rejectedCounts[url]
	totalCount := approvedCount + rejectedCount

	if totalCount == 0 {
		return 0, 0, nil // No hay datos para esta URL
	}

	// Calcular probabilidad basada en el porcentaje de errores
	probability := float64(rejectedCount) / float64(totalCount)

	// Obtener el status code más común de error para esta URL
	errorStatusCode, err := getMostCommonErrorStatusCode(db, url)
	if err != nil {
		return 0, 0, fmt.Errorf("error getting most common error status code: %v", err)
	}

	return probability, errorStatusCode, nil
}

// getMostCommonErrorStatusCode obtiene el status code de error más común para una URL
func getMostCommonErrorStatusCode(db *sql.DB, url string) (int, error) {
	query := `
		SELECT res.status_code, COUNT(*) as count
		FROM proxy_requests req
		LEFT JOIN proxy_responses res ON res.request_id = req.id
		WHERE req.url = ? AND res.status_code >= 300
		GROUP BY res.status_code
		ORDER BY count DESC
		LIMIT 1
	`

	var statusCode int
	var count int
	err := db.QueryRow(query, url).Scan(&statusCode, &count)
	if err != nil {
		if err == sql.ErrNoRows {
			return 500, nil // Default error code si no hay errores
		}
		return 0, fmt.Errorf("error querying most common error status code: %v", err)
	}

	return statusCode, nil
}

// CountRejectedRequests cuenta los rechazos por URL
func CountRejectedRequests(db *sql.DB) (map[string]int, error) {
	query := `
		SELECT sub.url, COUNT(sub.url) as count 
		FROM (
			SELECT req.url, COALESCE(req.port, 8080) as port, res.status_code, res.headers, res.body   
			FROM proxy_requests req
			LEFT JOIN proxy_responses res ON res.request_id = req.id
			WHERE res.status_code > 299
		) sub
		GROUP BY sub.url
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error querying rejected requests: %v", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var url string
		var count int
		err := rows.Scan(&url, &count)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %v", err)
		}
		counts[url] = count
	}

	return counts, nil
}

// CountApprovedRequests cuenta los aprobados por URL
func CountApprovedRequests(db *sql.DB) (map[string]int, error) {
	query := `
		SELECT sub.url, COUNT(sub.url) as count 
		FROM (
			SELECT req.url, COALESCE(req.port, 8080) as port, res.status_code, res.headers, res.body   
			FROM proxy_requests req
			LEFT JOIN proxy_responses res ON res.request_id = req.id
			WHERE res.status_code BETWEEN 100 AND 299
		) sub
		GROUP BY sub.url
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error querying approved requests: %v", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var url string
		var count int
		err := rows.Scan(&url, &count)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %v", err)
		}
		counts[url] = count
	}

	return counts, nil
}
