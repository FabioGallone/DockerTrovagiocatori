package database

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"trovagiocatoriAuth/internal/config"
)

type Database struct {
	Conn   *sql.DB
	Config *config.Config
}

func NewDatabase(cfg *config.Config) (*Database, error) {
	db, err := sql.Open("postgres", cfg.GetDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{
		Conn:   db,
		Config: cfg,
	}

	// Esegui le migrazioni
	if err := database.runMigrations(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	fmt.Println("✔ Database connected successfully")
	return database, nil
}

func (db *Database) Close() error {
	if db.Conn != nil {
		return db.Conn.Close()
	}
	return nil
}