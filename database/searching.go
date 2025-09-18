package database

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"strings"
)

// HierarchicalSearch realiza una búsqueda jerárquica en la base de datos
func HierarchicalSearch(db *sql.DB, criteria SearchCriteria) (*BestMatch, error) {
	var bestMatch *BestMatch
	currentScore := 0

	// 1. Búsqueda por URL completa (mayor prioridad)
	if criteria.URL != "" {
		query := `
			SELECT r.id, r.method, r.url, r.headers, r.body, res.status_code, res.body as response_body
			FROM proxy_requests r
			LEFT JOIN proxy_responses res ON r.id = res.request_id
			WHERE r.url = ?
		`

		rows, err := db.Query(query, criteria.URL)
		if err != nil {
			return nil, fmt.Errorf("error searching by URL: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var requestID int64
			var method, url, headers, body, responseBody string
			var statusCode int

			err := rows.Scan(&requestID, &method, &url, &headers, &body, &statusCode, &responseBody)
			if err != nil {
				continue
			}

			score := 100 // Puntuación máxima para URL completa
			if bestMatch == nil || score > currentScore {
				bestMatch = &BestMatch{
					Endpoint:   extractEndpoint(url),
					Headers:    parseHeaders(headers),
					Body:       responseBody,
					StatusCode: statusCode,
					Score:      score,
					Method:     method,
					URL:        url,
				}
				currentScore = score
			}
		}
	}

	// 2. Búsqueda por endpoint (segunda prioridad)
	if criteria.Endpoint != "" && (bestMatch == nil || currentScore < 80) {
		query := `
			SELECT r.id, r.method, r.url, r.headers, r.body, res.status_code, res.body as response_body
			FROM proxy_requests r
			LEFT JOIN proxy_responses res ON r.id = res.request_id
			WHERE r.url LIKE ?
		`

		rows, err := db.Query(query, "%"+criteria.Endpoint+"%")
		if err != nil {
			return nil, fmt.Errorf("error searching by endpoint: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var requestID int64
			var method, url, headers, body, responseBody string
			var statusCode int

			err := rows.Scan(&requestID, &method, &url, &headers, &body, &statusCode, &responseBody)
			if err != nil {
				continue
			}

			score := 80 // Puntuación para endpoint
			if bestMatch == nil || score > currentScore {
				bestMatch = &BestMatch{
					Endpoint:   extractEndpoint(url),
					Headers:    parseHeaders(headers),
					Body:       responseBody,
					StatusCode: statusCode,
					Score:      score,
					Method:     method,
					URL:        url,
				}
				currentScore = score
			}
		}
	}

	// 3. Búsqueda por body del request (tercera prioridad)
	if criteria.Body != "" && (bestMatch == nil || currentScore < 60) {
		query := `
			SELECT r.id, r.method, r.url, r.headers, r.body, res.status_code, res.body as response_body
			FROM proxy_requests r
			LEFT JOIN proxy_responses res ON r.id = res.request_id
			WHERE r.body LIKE ?
		`

		rows, err := db.Query(query, "%"+criteria.Body+"%")
		if err != nil {
			return nil, fmt.Errorf("error searching by body: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var requestID int64
			var method, url, headers, body, responseBody string
			var statusCode int

			err := rows.Scan(&requestID, &method, &url, &headers, &body, &statusCode, &responseBody)
			if err != nil {
				continue
			}

			score := 60 // Puntuación para body
			if bestMatch == nil || score > currentScore {
				bestMatch = &BestMatch{
					Endpoint:   extractEndpoint(url),
					Headers:    parseHeaders(headers),
					Body:       responseBody,
					StatusCode: statusCode,
					Score:      score,
					Method:     method,
					URL:        url,
				}
				currentScore = score
			}
		}
	}

	// 4. Búsqueda por body del response (cuarta prioridad)
	if criteria.Body != "" && (bestMatch == nil || currentScore < 40) {
		query := `
			SELECT r.id, r.method, r.url, r.headers, r.body, res.status_code, res.body as response_body
			FROM proxy_requests r
			LEFT JOIN proxy_responses res ON r.id = res.request_id
			WHERE res.body LIKE ?
		`

		rows, err := db.Query(query, "%"+criteria.Body+"%")
		if err != nil {
			return nil, fmt.Errorf("error searching by response body: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var requestID int64
			var method, url, headers, body, responseBody string
			var statusCode int

			err := rows.Scan(&requestID, &method, &url, &headers, &body, &statusCode, &responseBody)
			if err != nil {
				continue
			}

			score := 40 // Puntuación para response body
			if bestMatch == nil || score > currentScore {
				bestMatch = &BestMatch{
					Endpoint:   extractEndpoint(url),
					Headers:    parseHeaders(headers),
					Body:       responseBody,
					StatusCode: statusCode,
					Score:      score,
					Method:     method,
					URL:        url,
				}
				currentScore = score
			}
		}
	}

	// 5. Búsqueda por headers (quinta prioridad)
	if len(criteria.Headers) > 0 && (bestMatch == nil || currentScore < 20) {
		for headerKey, headerValue := range criteria.Headers {
			query := `
				SELECT r.id, r.method, r.url, r.headers, r.body, res.status_code, res.body as response_body
				FROM proxy_requests r
				LEFT JOIN proxy_responses res ON r.id = res.request_id
				WHERE r.headers LIKE ?
			`

			searchPattern := "%\"" + headerKey + "\":\"" + headerValue + "\"%"
			rows, err := db.Query(query, searchPattern)
			if err != nil {
				continue
			}
			defer rows.Close()

			for rows.Next() {
				var requestID int64
				var method, url, headers, body, responseBody string
				var statusCode int

				err := rows.Scan(&requestID, &method, &url, &headers, &body, &statusCode, &responseBody)
				if err != nil {
					continue
				}

				score := 20 // Puntuación para headers
				if bestMatch == nil || score > currentScore {
					bestMatch = &BestMatch{
						Endpoint:   extractEndpoint(url),
						Headers:    parseHeaders(headers),
						Body:       responseBody,
						StatusCode: statusCode,
						Score:      score,
						Method:     method,
						URL:        url,
					}
					currentScore = score
				}
			}
		}
	}

	// 6. Búsqueda por method (menor prioridad)
	if criteria.Method != "" && (bestMatch == nil || currentScore < 10) {
		query := `
			SELECT r.id, r.method, r.url, r.headers, r.body, res.status_code, res.body as response_body
			FROM proxy_requests r
			LEFT JOIN proxy_responses res ON r.id = res.request_id
			WHERE r.method = ?
		`

		rows, err := db.Query(query, criteria.Method)
		if err != nil {
			return nil, fmt.Errorf("error searching by method: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var requestID int64
			var method, url, headers, body, responseBody string
			var statusCode int

			err := rows.Scan(&requestID, &method, &url, &headers, &body, &statusCode, &responseBody)
			if err != nil {
				continue
			}

			score := 10 // Puntuación mínima para method
			if bestMatch == nil || score > currentScore {
				bestMatch = &BestMatch{
					Endpoint:   extractEndpoint(url),
					Headers:    parseHeaders(headers),
					Body:       responseBody,
					StatusCode: statusCode,
					Score:      score,
					Method:     method,
					URL:        url,
				}
				currentScore = score
			}
		}
	}

	return bestMatch, nil
}

// extractEndpoint extrae el endpoint de una URL completa o path
func extractEndpoint(url string) string {
	// Si la URL ya empieza con "/", es un path directo
	if strings.HasPrefix(url, "/") {
		return url
	}

	// Remover protocolo si existe
	if strings.HasPrefix(url, "http://") {
		url = strings.TrimPrefix(url, "http://")
	} else if strings.HasPrefix(url, "https://") {
		url = strings.TrimPrefix(url, "https://")
	}

	// Buscar la primera barra después del host:puerto
	slashIndex := strings.Index(url, "/")
	if slashIndex == -1 {
		return "/"
	}

	// Extraer el path completo
	path := url[slashIndex:]

	// Si el path está vacío, retornar "/"
	if path == "" {
		return "/"
	}

	return path
}

// parseHeaders convierte el string JSON de headers a map
func parseHeaders(headersJSON string) map[string]string {
	headers := make(map[string]string)
	if headersJSON != "" {
		json.Unmarshal([]byte(headersJSON), &headers)
	}
	return headers
}
