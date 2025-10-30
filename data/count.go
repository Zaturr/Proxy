package yaml

import (
	"database/sql"
	"fmt"
)

// CountResult representa el resultado de un conteo
type CountResult struct {
	URL        string `json:"url"`
	Port       int    `json:"port"`
	StatusCode int    `json:"status_code"`
	Headers    string `json:"headers"`
	Body       string `json:"body"`
	Count      int    `json:"count"`
}

// GetUniquePathsForYAML obtiene los paths únicos exitosos para agregar en el YAML
func GetUniquePathsForYAML(db *sql.DB) ([]CountResult, error) {
	query := `
		SELECT request_endpoint, 8080 as port, response_status_code, response_headers, response_body 
		FROM mock_transactions
		WHERE response_status_code BETWEEN 100 AND 299
		GROUP BY request_endpoint
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error querying unique paths: %v", err)
	}
	defer rows.Close()

	var results []CountResult
	for rows.Next() {
		var result CountResult
		err := rows.Scan(&result.URL, &result.Port, &result.StatusCode, &result.Headers, &result.Body)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %v", err)
		}
		results = append(results, result)
	}

	return results, nil
}

// GetErrorPathsForChaosInjection obtiene los paths con códigos de error para chaos injection
func GetErrorPathsForChaosInjection(db *sql.DB) ([]CountResult, error) {
	query := `
		SELECT request_endpoint, 8080 as port, response_status_code, response_headers, response_body 
		FROM mock_transactions
		WHERE response_status_code > 299
		GROUP BY request_endpoint
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error querying error paths: %v", err)
	}
	defer rows.Close()

	var results []CountResult
	for rows.Next() {
		var result CountResult
		err := rows.Scan(&result.URL, &result.Port, &result.StatusCode, &result.Headers, &result.Body)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %v", err)
		}
		results = append(results, result)
	}

	return results, nil
}

// CountRejectedRequests cuenta los rechazos por URL
func CountRejectedRequests(db *sql.DB) (map[string]int, error) {
	query := `
		SELECT request_endpoint, COUNT(request_endpoint) as count 
		FROM mock_transactions
		WHERE response_status_code > 299
		GROUP BY request_endpoint
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
		SELECT request_endpoint, COUNT(request_endpoint) as count 
		FROM mock_transactions
		WHERE response_status_code BETWEEN 100 AND 299
		GROUP BY request_endpoint
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

func GetRequestStats(db *sql.DB) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var totalTransactions int
	err := db.QueryRow("SELECT COUNT(*) FROM mock_transactions").Scan(&totalTransactions)
	if err != nil {
		return nil, fmt.Errorf("error counting total transactions: %v", err)
	}
	stats["total_transactions"] = totalTransactions

	var successfulRequests int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM mock_transactions
		WHERE response_status_code BETWEEN 100 AND 299
	`).Scan(&successfulRequests)
	if err != nil {
		return nil, fmt.Errorf("error counting successful requests: %v", err)
	}
	stats["successful_requests"] = successfulRequests

	var errorRequests int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM mock_transactions
		WHERE response_status_code > 299
	`).Scan(&errorRequests)
	if err != nil {
		return nil, fmt.Errorf("error counting error requests: %v", err)
	}
	stats["error_requests"] = errorRequests

	if totalTransactions > 0 {
		successRate := float64(successfulRequests) / float64(totalTransactions) * 100
		stats["success_rate"] = successRate
	} else {
		stats["success_rate"] = 0.0
	}

	return stats, nil
}
