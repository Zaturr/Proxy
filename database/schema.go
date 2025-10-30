package database

import (
	"database/sql"
	"proxy/database/internal"
)

// InitDB inicializa la base de datos usando la función interna
func InitDB(dbPath string) (*sql.DB, error) {
	return internal.InitDB(dbPath)
}
