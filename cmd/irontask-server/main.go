package main

import (
	"log"
	"os"

	"github.com/existflow/irontask/internal/logger"
	"github.com/existflow/irontask/server"
)

func main() {
	// Initialize logger
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "INFO"
	}

	logConfig := logger.Config{
		Level:      logger.ParseLevel(logLevel),
		FilePath:   "", // No file logging for server
		MaxSize:    0,
		MaxAge:     0,
		MaxBackups: 0,
		Console:    true, // Console only
	}

	if err := logger.Init(logConfig); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Close()

	logger.Info("IronTask sync server starting")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://irontask:irontask@localhost:5432/irontask?sslmode=disable"
	}

	logger.Info("Database configuration", logger.F("url", dbURL))

	srv, err := server.New(dbURL)
	if err != nil {
		logger.Error("Failed to create server", logger.F("error", err))
		log.Fatalf("Failed to create server: %v", err)
	}
	defer func() {
		if err := srv.Close(); err != nil {
			logger.Error("Error closing server", logger.F("error", err))
			log.Printf("Error closing server: %v", err)
		}
	}()

	logger.Info("Server listening", logger.F("port", port))
	log.Printf("IronTask sync server starting on :%s", port)
	if err := srv.Start(":" + port); err != nil {
		logger.Error("Server failed", logger.F("error", err))
		log.Fatalf("Server failed: %v", err)
	}
}
