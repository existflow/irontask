package main

import (
	"log"
	"os"

	"github.com/existflow/irontask/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://localhost:5432/irontask?sslmode=disable"
	}

	srv, err := server.New(dbURL)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	defer func() {
		if err := srv.Close(); err != nil {
			log.Printf("Error closing server: %v", err)
		}
	}()

	log.Printf("IronTask sync server starting on :%s", port)
	if err := srv.Start(":" + port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
