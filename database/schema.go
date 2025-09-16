package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	// Crear tabla para requests
	createRequestsTable := `
	CREATE TABLE IF NOT EXISTS proxy_requests (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		method TEXT NOT NULL,
		url TEXT NOT NULL,
		headers TEXT,
		body TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// Crear tabla para responses
	createResponsesTable := `
	CREATE TABLE IF NOT EXISTS proxy_responses (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		request_id INTEGER NOT NULL,
		status_code INTEGER NOT NULL,
		headers TEXT,
		body TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (request_id) REFERENCES proxy_requests (id)
	);`

	// Crear índices para mejorar el rendimiento
	createIndexes := `
	CREATE INDEX IF NOT EXISTS idx_requests_timestamp ON proxy_requests(timestamp);
	CREATE INDEX IF NOT EXISTS idx_responses_request_id ON proxy_responses(request_id);
	CREATE INDEX IF NOT EXISTS idx_responses_timestamp ON proxy_responses(timestamp);
	`

	if _, err := db.Exec(createRequestsTable); err != nil {
		return nil, fmt.Errorf("error creating requests table: %v", err)
	}

	if _, err := db.Exec(createResponsesTable); err != nil {
		return nil, fmt.Errorf("error creating responses table: %v", err)
	}

	if _, err := db.Exec(createIndexes); err != nil {
		return nil, fmt.Errorf("error creating indexes: %v", err)
	}

	log.Println("Database initialized successfully")
	return db, nil
}
