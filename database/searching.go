package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Constantes para los scores de prioridad
const (
	ScoreURL          = 100
	ScoreEndpoint     = 80
	ScoreRequestBody  = 60
	ScoreResponseBody = 40
	ScoreHeaders      = 20
	ScoreMethod       = 10
)

const baseQuery = `SELECT r.id, r.method, r.url, r.headers, r.body, r.port, res.status_code, res.body as response_body
				   FROM proxy_requests r LEFT JOIN proxy_responses res ON r.id = res.request_id`

const unifiedQuery = `SELECT uuid, request_method, request_endpoint, request_headers, request_body, 
					  response_status_code, response_body, timestamp, port
					  FROM mock_transactions`

// HierarchicalSearch realiza una búsqueda jerárquica en la bd
func HierarchicalSearch(db *sql.DB, criteria SearchCriteria) (*BestMatch, error) {
	searchers := []func(*sql.DB, SearchCriteria) (*BestMatch, error){
		searchByURL,          // Prioridad 100
		searchByEndpoint,     // Prioridad 80
		searchByRequestBody,  // Prioridad 60
		searchByResponseBody, // Prioridad 40
		searchByHeaders,      // Prioridad 20
		searchByMethod,       // Prioridad 10
	}

	for _, searcher := range searchers {
		result, err := searcher(db, criteria)
		if err != nil {
			return nil, err
		}

		if result != nil {
			return result, nil
		}
	}

	return nil, nil
}

func GetAllRequestsForYAML(db *sql.DB) ([]BestMatch, error) {
	query := unifiedQuery + " WHERE response_status_code IS NOT NULL ORDER BY timestamp DESC"
	return executeMultipleSearchUnified(db, query)
}

func GetSimilarRequests(db *sql.DB, criteria SearchCriteria) ([]BestMatch, error) {
	if criteria.Endpoint == "" || criteria.Method == "" {
		return nil, nil
	}

	query := unifiedQuery + " WHERE request_endpoint LIKE ? AND request_method = ? ORDER BY timestamp DESC LIMIT 10"
	return executeMultipleSearchUnified(db, query, "%"+criteria.Endpoint+"%", criteria.Method)
}

func executeMultipleSearch(db *sql.DB, query string, args ...interface{}) ([]BestMatch, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()

	var matches []BestMatch
	for rows.Next() {
		match, err := scanRowToBestMatch(rows)
		if err != nil {
			continue
		}
		matches = append(matches, *match)
	}

	return matches, nil
}

func executeSearch(db *sql.DB, query string, args ...interface{}) (*BestMatch, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		return scanRowToBestMatch(rows)
	}

	return nil, nil
}

// scanRowToBestMatch convierte una fila de la base de datos a BestMatch
func scanRowToBestMatch(rows *sql.Rows) (*BestMatch, error) {
	var requestID int64
	var method, url, headers, body, responseBody string
	var statusCode, port int

	if err := rows.Scan(&requestID, &method, &url, &headers, &body, &port, &statusCode, &responseBody); err != nil {
		return nil, err
	}

	return &BestMatch{
		Endpoint:   extractEndpoint(url),
		Headers:    parseHeaders(headers),
		Body:       responseBody,
		StatusCode: statusCode,
		Score:      100,
		Method:     method,
		URL:        url,
		Port:       port,
	}, nil
}

func searchGeneric(db *sql.DB, criteria SearchCriteria, condition string, args []interface{}, score int) (*BestMatch, error) {
	query := baseQuery + " WHERE " + condition

	result, err := executeSearch(db, query, args...)
	if result != nil {
		result.Score = score
	}
	return result, err
}

// Funciones de búsqueda ultra-simplificadas
func searchByURL(db *sql.DB, criteria SearchCriteria) (*BestMatch, error) {
	if criteria.URL == "" {
		return nil, nil
	}
	return searchGeneric(db, criteria, "r.url = ?", []interface{}{criteria.URL}, ScoreURL)
}

func searchByEndpoint(db *sql.DB, criteria SearchCriteria) (*BestMatch, error) {
	if criteria.Endpoint == "" {
		return nil, nil
	}
	return searchGeneric(db, criteria, "r.url LIKE ?", []interface{}{"%" + criteria.Endpoint + "%"}, ScoreEndpoint)
}

func searchByRequestBody(db *sql.DB, criteria SearchCriteria) (*BestMatch, error) {
	if criteria.Body == "" {
		return nil, nil
	}
	return searchGeneric(db, criteria, "r.body LIKE ?", []interface{}{"%" + criteria.Body + "%"}, ScoreRequestBody)
}

func searchByResponseBody(db *sql.DB, criteria SearchCriteria) (*BestMatch, error) {
	if criteria.Body == "" {
		return nil, nil
	}
	return searchGeneric(db, criteria, "res.body LIKE ?", []interface{}{"%" + criteria.Body + "%"}, ScoreResponseBody)
}

func searchByHeaders(db *sql.DB, criteria SearchCriteria) (*BestMatch, error) {
	if len(criteria.Headers) == 0 {
		return nil, nil
	}

	query := baseQuery + " WHERE r.headers LIKE ?"

	for headerKey, headerValue := range criteria.Headers {
		searchPattern := "%\"" + headerKey + "\":\"" + headerValue + "\"%"
		result, err := executeSearch(db, query, searchPattern)
		if result != nil {
			result.Score = ScoreHeaders
			return result, nil
		}
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func searchByMethod(db *sql.DB, criteria SearchCriteria) (*BestMatch, error) {
	if criteria.Method == "" {
		return nil, nil
	}
	return searchGeneric(db, criteria, "r.method = ?", []interface{}{criteria.Method}, ScoreMethod)
}

// Funciones utilitarias simplificadas
func extractEndpoint(url string) string {
	if strings.HasPrefix(url, "/") {
		return url
	}

	// Remover protocolo
	url = strings.TrimPrefix(strings.TrimPrefix(url, "http://"), "https://")

	// Buscar path
	if slashIndex := strings.Index(url, "/"); slashIndex != -1 {
		if path := url[slashIndex:]; path != "" {
			return path
		}
	}

	return "/"
}

func parseHeaders(headersJSON string) map[string]string {
	headers := make(map[string]string)
	if headersJSON != "" {
		json.Unmarshal([]byte(headersJSON), &headers)
	}
	return headers
}

// HierarchicalSearchUnified realiza una búsqueda jerárquica en la tabla unificada
func HierarchicalSearchUnified(db *sql.DB, criteria SearchCriteria) (*BestMatch, error) {
	searchers := []func(*sql.DB, SearchCriteria) (*BestMatch, error){
		searchByURLUnified,          // Prioridad 100
		searchByEndpointUnified,     // Prioridad 80
		searchByRequestBodyUnified,  // Prioridad 60
		searchByResponseBodyUnified, // Prioridad 40
		searchByHeadersUnified,      // Prioridad 20
		searchByMethodUnified,       // Prioridad 10
	}

	for _, searcher := range searchers {
		result, err := searcher(db, criteria)
		if err != nil {
			return nil, err
		}

		if result != nil {
			return result, nil
		}
	}

	return nil, nil
}

func GetAllRequestsForYAMLUnified(db *sql.DB) ([]BestMatch, error) {
	query := unifiedQuery + " WHERE response_status_code IS NOT NULL ORDER BY timestamp DESC"
	return executeMultipleSearchUnified(db, query)
}

func GetSimilarRequestsUnified(db *sql.DB, criteria SearchCriteria) ([]BestMatch, error) {
	if criteria.Endpoint == "" || criteria.Method == "" {
		return nil, nil
	}

	query := unifiedQuery + " WHERE request_endpoint LIKE ? AND request_method = ? ORDER BY timestamp DESC LIMIT 10"
	return executeMultipleSearchUnified(db, query, "%"+criteria.Endpoint+"%", criteria.Method)
}

func executeMultipleSearchUnified(db *sql.DB, query string, args ...interface{}) ([]BestMatch, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()

	var matches []BestMatch
	for rows.Next() {
		match, err := scanRowToBestMatchUnified(rows)
		if err != nil {
			continue
		}
		matches = append(matches, *match)
	}

	return matches, nil
}

func executeSearchUnified(db *sql.DB, query string, args ...interface{}) (*BestMatch, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		return scanRowToBestMatchUnified(rows)
	}

	return nil, nil
}

// scanRowToBestMatchUnified convierte una fila de la tabla unificada a BestMatch
func scanRowToBestMatchUnified(rows *sql.Rows) (*BestMatch, error) {
    var uuid, method, endpoint, headers, body, responseBody string
    var statusCode, port int
    var timestamp time.Time

    if err := rows.Scan(&uuid, &method, &endpoint, &headers, &body, &statusCode, &responseBody, &timestamp, &port); err != nil {
        return nil, err
    }

    return &BestMatch{
        Endpoint:   extractEndpoint(endpoint),
        Headers:    parseHeaders(headers),
        Body:       responseBody,
        StatusCode: statusCode,
        Score:      100,
        Method:     method,
        URL:        endpoint,
        Port:       port,
    }, nil
}

func searchGenericUnified(db *sql.DB, criteria SearchCriteria, condition string, args []interface{}, score int) (*BestMatch, error) {
	query := unifiedQuery + " WHERE " + condition

	result, err := executeSearchUnified(db, query, args...)
	if result != nil {
		result.Score = score
	}
	return result, err
}

// Funciones de búsqueda para tabla unificada
func searchByURLUnified(db *sql.DB, criteria SearchCriteria) (*BestMatch, error) {
	if criteria.URL == "" {
		return nil, nil
	}
	return searchGenericUnified(db, criteria, "request_endpoint = ?", []interface{}{criteria.URL}, ScoreURL)
}

func searchByEndpointUnified(db *sql.DB, criteria SearchCriteria) (*BestMatch, error) {
	if criteria.Endpoint == "" {
		return nil, nil
	}
	return searchGenericUnified(db, criteria, "request_endpoint LIKE ?", []interface{}{"%" + criteria.Endpoint + "%"}, ScoreEndpoint)
}

func searchByRequestBodyUnified(db *sql.DB, criteria SearchCriteria) (*BestMatch, error) {
	if criteria.Body == "" {
		return nil, nil
	}
	return searchGenericUnified(db, criteria, "request_body LIKE ?", []interface{}{"%" + criteria.Body + "%"}, ScoreRequestBody)
}

func searchByResponseBodyUnified(db *sql.DB, criteria SearchCriteria) (*BestMatch, error) {
	if criteria.Body == "" {
		return nil, nil
	}
	return searchGenericUnified(db, criteria, "response_body LIKE ?", []interface{}{"%" + criteria.Body + "%"}, ScoreResponseBody)
}

func searchByHeadersUnified(db *sql.DB, criteria SearchCriteria) (*BestMatch, error) {
	if len(criteria.Headers) == 0 {
		return nil, nil
	}

	query := unifiedQuery + " WHERE request_headers LIKE ?"

	for headerKey, headerValue := range criteria.Headers {
		searchPattern := "%\"" + headerKey + "\":\"" + headerValue + "\"%"
		result, err := executeSearchUnified(db, query, searchPattern)
		if result != nil {
			result.Score = ScoreHeaders
			return result, nil
		}
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func searchByMethodUnified(db *sql.DB, criteria SearchCriteria) (*BestMatch, error) {
	if criteria.Method == "" {
		return nil, nil
	}
	return searchGenericUnified(db, criteria, "request_method = ?", []interface{}{criteria.Method}, ScoreMethod)
}
