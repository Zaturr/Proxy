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
		SELECT sub.url, sub.port, sub.status_code, sub.headers, sub.body 
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
		SELECT sub.url, sub.port, sub.status_code, sub.headers, sub.body 
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

// GetRequestStats obtiene estadísticas generales de requests
func GetRequestStats(db *sql.DB) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Contar total de requests
	var totalRequests int
	err := db.QueryRow("SELECT COUNT(*) FROM proxy_requests").Scan(&totalRequests)
	if err != nil {
		return nil, fmt.Errorf("error counting total requests: %v", err)
	}
	stats["total_requests"] = totalRequests

	// Contar total de responses
	var totalResponses int
	err = db.QueryRow("SELECT COUNT(*) FROM proxy_responses").Scan(&totalResponses)
	if err != nil {
		return nil, fmt.Errorf("error counting total responses: %v", err)
	}
	stats["total_responses"] = totalResponses

	// Contar requests exitosos
	var successfulRequests int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM proxy_requests req
		LEFT JOIN proxy_responses res ON res.request_id = req.id
		WHERE res.status_code BETWEEN 100 AND 299
	`).Scan(&successfulRequests)
	if err != nil {
		return nil, fmt.Errorf("error counting successful requests: %v", err)
	}
	stats["successful_requests"] = successfulRequests

	// Contar requests con error
	var errorRequests int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM proxy_requests req
		LEFT JOIN proxy_responses res ON res.request_id = req.id
		WHERE res.status_code > 299
	`).Scan(&errorRequests)
	if err != nil {
		return nil, fmt.Errorf("error counting error requests: %v", err)
	}
	stats["error_requests"] = errorRequests

	// Calcular porcentaje de éxito
	if totalResponses > 0 {
		successRate := float64(successfulRequests) / float64(totalResponses) * 100
		stats["success_rate"] = fmt.Sprintf("%.2f%%", successRate)
	} else {
		stats["success_rate"] = "0.00%"
	}

	return stats, nil
}
