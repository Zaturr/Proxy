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
		port INTEGER,
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
		port INTEGER,
		headers TEXT,
		body TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (request_id) REFERENCES proxy_requests (id)
	);`

	createIndexes := `
	CREATE INDEX IF NOT EXISTS idx_responses_request_id ON proxy_responses(request_id);
	CREATE INDEX IF NOT EXISTS idx_responses_port ON proxy_responses(port);
	CREATE INDEX IF NOT EXISTS idx_requests_method ON proxy_requests(method);
	CREATE INDEX IF NOT EXISTS idx_requests_url ON proxy_requests(url);
	CREATE INDEX IF NOT EXISTS idx_requests_port ON proxy_requests(port);
	CREATE INDEX IF NOT EXISTS idx_requests_method_url ON proxy_requests(method, url);
	`

	if _, err := db.Exec(createRequestsTable); err != nil {
		return nil, fmt.Errorf("error creating requests table: %v", err)
	}

	if _, err := db.Exec(createResponsesTable); err != nil {
		return nil, fmt.Errorf("error creating responses table: %v", err)
	}

	addPortColumnRequests := `ALTER TABLE proxy_requests ADD COLUMN port INTEGER;`
	db.Exec(addPortColumnRequests)

	addPortColumnResponses := `ALTER TABLE proxy_responses ADD COLUMN port INTEGER;`
	db.Exec(addPortColumnResponses)
	if _, err := db.Exec(createIndexes); err != nil {
		return nil, fmt.Errorf("error creating indexes: %v", err)
	}

	log.Println("Database initialized successfully")
	return db, nil
}
