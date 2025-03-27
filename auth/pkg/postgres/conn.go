package postgres

import (
	"database/sql"
	"fmt"
)

type Conn struct {
	DB *sql.DB
}

func NewPostgresConn(connectionString string) (*Conn, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Conn{DB: db}, nil
}

func (c *Conn) Close() error {
	return c.DB.Close()
}
