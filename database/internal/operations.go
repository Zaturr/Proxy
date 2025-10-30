package internal

import (
	"database/sql"
	"time"
)

// Mockdata representa los datos de una transacción mock
type Mockdata struct {
	UUID               string    `json:"uuid" db:"uuid"`
	RequestHeaders     string    `json:"request_headers" db:"request_headers"`
	RequestMethod      string    `json:"request_method" db:"request_method"`
	RequestEndpoint    string    `json:"request_endpoint" db:"request_endpoint"`
	RequestBody        string    `json:"request_body" db:"request_body"`
	ResponseHeaders    string    `json:"response_headers" db:"response_headers"`
	ResponseBody       string    `json:"response_body" db:"response_body"`
	ResponseStatusCode int       `json:"response_status_code" db:"response_status_code"`
	Timestamp          time.Time `json:"timestamp" db:"timestamp"`
	Port               int       `json:"port" db:"port"`
}

// InsertOperation inserta una nueva operación en la base de datos
func InsertOperation(db *sql.DB, operation *Mockdata) error {
	query := `
    INSERT INTO mock_transactions (
        uuid, request_headers, request_method, 
        request_endpoint, request_body, response_headers, response_body, 
        response_status_code, timestamp, port
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.Exec(query,
		operation.UUID,
		operation.RequestHeaders,
		operation.RequestMethod,
		operation.RequestEndpoint,
		operation.RequestBody,
		operation.ResponseHeaders,
		operation.ResponseBody,
		operation.ResponseStatusCode,
		operation.Timestamp,
		operation.Port,
	)

	return err
}

// UpdateOperationResponse actualiza solo la respuesta de una operación
func UpdateOperationResponse(db *sql.DB, uuid string, responseHeaders, responseBody string, statusCode int) error {
	query := `UPDATE mock_transactions SET 
		response_headers = ?, response_body = ?, response_status_code = ?
		WHERE uuid = ?`

	_, err := db.Exec(query, responseHeaders, responseBody, statusCode, uuid)
	return err
}

// GetOperationByUUID obtiene una operación por UUID
func GetOperationByUUID(db *sql.DB, uuid string) (*Mockdata, error) {
	query := `
        SELECT uuid, request_headers, request_method, 
               request_endpoint, request_body, response_headers, response_body, 
               response_status_code, timestamp, port
        FROM mock_transactions
        WHERE uuid = ?`

	row := db.QueryRow(query, uuid)

	var operation Mockdata
	err := row.Scan(
		&operation.UUID,
		&operation.RequestHeaders,
		&operation.RequestMethod,
		&operation.RequestEndpoint,
		&operation.RequestBody,
		&operation.ResponseHeaders,
		&operation.ResponseBody,
		&operation.ResponseStatusCode,
		&operation.Timestamp,
		&operation.Port,
	)

	if err != nil {
		return nil, err
	}

	return &operation, nil
}

// GetAllOperations obtiene todas las operaciones con paginación
func GetAllOperations(db *sql.DB, limit, offset int) ([]Mockdata, error) {
	query := `
        SELECT uuid, request_headers, request_method, 
               request_endpoint, request_body, response_headers, response_body, 
               response_status_code, timestamp, port
        FROM mock_transactions
        ORDER BY timestamp DESC
        LIMIT ? OFFSET ?`

	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var operations []Mockdata
	for rows.Next() {
		var operation Mockdata
		err := rows.Scan(
			&operation.UUID,
			&operation.RequestHeaders,
			&operation.RequestMethod,
			&operation.RequestEndpoint,
			&operation.RequestBody,
			&operation.ResponseHeaders,
			&operation.ResponseBody,
			&operation.ResponseStatusCode,
			&operation.Timestamp,
			&operation.Port,
		)
		if err != nil {
			return nil, err
		}
		operations = append(operations, operation)
	}

	return operations, nil
}
