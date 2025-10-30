package infraestructure

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
		SELECT response_status_code, COUNT(*) as count
		FROM mock_transactions
		WHERE request_endpoint = ? AND response_status_code >= 300
		GROUP BY response_status_code
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

// GetTransactionUnified obtiene una transacción desde la tabla unificada
func GetTransactionUnified(db *sql.DB, uuid string) (*database.ProxyTransaction, error) {
	return database.GetTransactionUnified(db, uuid)
}

// InsertTransactionUnified inserta una transacción completa en la tabla unificada
func InsertTransactionUnified(db *sql.DB, request *database.ProxyRequest, response *database.ProxyResponse) error {
	return database.InsertTransactionUnified(db, request, response)
}

// GetAllTransactionsUnified obtiene todas las transacciones desde la tabla unificada
func GetAllTransactionsUnified(db *sql.DB, limit, offset int) ([]database.ProxyTransaction, error) {
	return database.GetAllTransactionsUnified(db, limit, offset)
}

// HierarchicalSearchUnified realiza una búsqueda jerárquica en la tabla unificada
func HierarchicalSearchUnified(db *sql.DB, criteria database.SearchCriteria) (*database.BestMatch, error) {
	return database.HierarchicalSearchUnified(db, criteria)
}

// GetAllRequestsForYAMLUnified obtiene todas las requests para YAML desde la tabla unificada
func GetAllRequestsForYAMLUnified(db *sql.DB) ([]database.BestMatch, error) {
	return database.GetAllRequestsForYAMLUnified(db)
}

// GetSimilarRequestsUnified obtiene requests similares desde la tabla unificada
func GetSimilarRequestsUnified(db *sql.DB, criteria database.SearchCriteria) ([]database.BestMatch, error) {
	return database.GetSimilarRequestsUnified(db, criteria)
}
