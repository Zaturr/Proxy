package database

import (
	"database/sql"
	"fmt"
	"proxy/database"
)

// GetTransaction obtiene una transacción completa (request + response)
func GetTransaction(db *sql.DB, requestID int64) (*database.ProxyTransaction, error) {
	request, err := GetRequest(db, requestID)
	if err != nil {
		return nil, fmt.Errorf("error getting request: %v", err)
	}

	response, err := GetResponse(db, requestID)
	if err != nil {
		return nil, fmt.Errorf("error getting response: %v", err)
	}

	return &database.ProxyTransaction{
		Request:  *request,
		Response: *response,
	}, nil
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
	probability := float64(int(float64(rejectedCount)/float64(totalCount)*100*100)) / 100

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
