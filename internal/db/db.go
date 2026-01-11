package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/existflow/irontask/internal/database"
	"github.com/existflow/irontask/internal/logger"
	_ "modernc.org/sqlite"
)

// DB wraps the SQLite database connection
type DB struct {
	*sql.DB
	*database.Queries
}

// DefaultDBPath returns the default database path (~/.irontask/tasks.sqlite)
func DefaultDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".irontask", "tasks.sqlite"), nil
}

// Open opens or creates the SQLite database
func Open(dbPath string) (*DB, error) {
	logger.Info("Opening database", logger.F("path", dbPath))

	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.Error("Failed to create database directory", logger.F("dir", dir), logger.F("error", err))
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database
	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		logger.Error("Failed to open database", logger.F("path", dbPath), logger.F("error", err))
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		logger.Error("Failed to connect to database", logger.F("error", err))
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Enable foreign keys
	if _, err := sqlDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		logger.Error("Failed to enable foreign keys", logger.F("error", err))
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	logger.Debug("Database connection established")

	db := &DB{
		DB:      sqlDB,
		Queries: database.New(sqlDB),
	}

	// Run migrations
	logger.Info("Running database migrations")
	if err := db.migrate(); err != nil {
		logger.Error("Failed to run migrations", logger.F("error", err))
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Info("Database opened successfully")
	return db, nil
}

// OpenDefault opens the database at the default path
func OpenDefault() (*DB, error) {
	path, err := DefaultDBPath()
	if err != nil {
		return nil, err
	}
	return Open(path)
}
