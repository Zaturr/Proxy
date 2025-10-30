package internal

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

// DBConfig configuración de conexiones de base de datos
type DBConfig struct {
	MaxOpenConns    int           // Máximo de conexiones abiertas
	MinConn         int           // Mínimo de conexiones inactivas
	ConnMaxLifetime time.Duration // Tiempo máximo de vida de conexión
	ConnMaxIdleTime time.Duration // Tiempo máximo inactivo
}

func InitDB(dbPath string) (*sql.DB, error) {
	return InitDBWithConfig(dbPath, DBConfig{
		MaxOpenConns:    1,
		MinConn:         1,
		ConnMaxLifetime: 0,
		ConnMaxIdleTime: 0,
	})
}

func InitDBWithConfig(dbPath string, config DBConfig) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	// Habilitar WAL mode para mejor concurrencia
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("error setting WAL mode: %v", err)
	}

	if _, err := db.Exec("PRAGMA synchronous=NORMAL"); err != nil {
		return nil, fmt.Errorf("error setting synchronous mode: %v", err)
	}

	// Configurar límites de conexiones para SQLite
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MinConn)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	// Crear tabla unificada para requests y responses (Mockingbird style)
	createUnifiedTable := `
    CREATE TABLE IF NOT EXISTS mock_transactions (
        uuid TEXT PRIMARY KEY,
        port INTEGER,
        request_headers TEXT,
        request_method TEXT NOT NULL,
        request_endpoint TEXT NOT NULL,
        request_body TEXT NOT NULL,
        response_headers TEXT,
        response_body TEXT NOT NULL,
        response_status_code INTEGER,
        timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
    );`

	createIndexes := `


	CREATE INDEX IF NOT EXISTS idx_transactions_method ON mock_transactions(request_method);
	CREATE INDEX IF NOT EXISTS idx_transactions_endpoint ON mock_transactions(request_endpoint);
	CREATE INDEX IF NOT EXISTS idx_transactions_method_endpoint ON mock_transactions(request_method, request_endpoint);
	`

	if _, err := db.Exec(createUnifiedTable); err != nil {
		return nil, fmt.Errorf("error creating unified table: %v", err)
	}

	// Intentar agregar la columna port si la base ya existía y no la tiene (ignorar error si ya existe)
	if _, err := db.Exec("ALTER TABLE mock_transactions ADD COLUMN port INTEGER"); err != nil {
		log.Printf("Info: port column may already exist: %v", err)
	}

	// Asegurar que la columna 'port' quede inmediatamente después de 'uuid' (reordenar si es necesario)
	if err := ensurePortAfterUUID(db); err != nil {
		return nil, fmt.Errorf("error ensuring port column order: %v", err)
	}

	if _, err := db.Exec(createIndexes); err != nil {
		return nil, fmt.Errorf("error creating indexes: %v", err)
	}

	log.Println("Database initialized successfully with unified table (mock_transactions)")
	return db, nil
}

// ensurePortAfterUUID verifica el orden de columnas y, si 'port' no está después de 'uuid',
// recrea la tabla con el orden correcto preservando los datos e índices.
func ensurePortAfterUUID(db *sql.DB) error {
	// Comprobar orden actual de columnas
	rows, err := db.Query("PRAGMA table_info(mock_transactions)")
	if err != nil {
		return err
	}
	defer rows.Close()

	type col struct {
		cid  int
		name string
	}
	cols := []col{}
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt interface{}
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		cols = append(cols, col{cid: cid, name: name})
	}

	if len(cols) == 0 {
		return nil
	}

	// Encontrar posiciones
	uuidIdx, portIdx := -1, -1
	for _, c := range cols {
		if c.name == "uuid" {
			uuidIdx = c.cid
		}
		if c.name == "port" {
			portIdx = c.cid
		}
	}

	// Si no hay columna port aún, no hay nada que reordenar
	if portIdx == -1 || uuidIdx == -1 {
		return nil
	}

	// Si port ya está inmediatamente después de uuid, no hacemos nada
	if portIdx == uuidIdx+1 {
		return nil
	}

	// Reordenar: recrear tabla con orden deseado
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmts := []string{
		"PRAGMA foreign_keys=off;",
		"BEGIN TRANSACTION;",
		`CREATE TABLE mock_transactions_new (
            uuid TEXT PRIMARY KEY,
            port INTEGER,
            request_headers TEXT,
            request_method TEXT NOT NULL,
            request_endpoint TEXT NOT NULL,
            request_body TEXT NOT NULL,
            response_headers TEXT,
            response_body TEXT NOT NULL,
            response_status_code INTEGER,
            timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
        );`,
		`INSERT INTO mock_transactions_new (
            uuid, port, request_headers, request_method, request_endpoint, request_body,
            response_headers, response_body, response_status_code, timestamp
        )
        SELECT uuid, port, request_headers, request_method, request_endpoint, request_body,
               response_headers, response_body, response_status_code, timestamp
        FROM mock_transactions;`,
		`DROP TABLE mock_transactions;`,
		`ALTER TABLE mock_transactions_new RENAME TO mock_transactions;`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_method ON mock_transactions(request_method);`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_endpoint ON mock_transactions(request_endpoint);`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_method_endpoint ON mock_transactions(request_method, request_endpoint);`,
		"COMMIT;",
		"PRAGMA foreign_keys=on;",
	}

	for _, s := range stmts {
		if _, err := tx.Exec(s); err != nil {
			return err
		}
	}

	return tx.Commit()
}
