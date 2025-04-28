package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Get database URL
	dbURL := os.Getenv("POSTGRES_DB_URL")
	if dbURL == "" {
		log.Fatalf("POSTGRES_DB_URL environment variable not set")
	}

	// Connect to database
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	// Read migration file
	migrationFile := "scripts/migrations/001_create_initial_schema.sql"
	migrationSQL, err := ioutil.ReadFile(migrationFile)
	if err != nil {
		log.Fatalf("Unable to read migration file: %v", err)
	}

	// Execute migration
	_, err = pool.Exec(context.Background(), string(migrationSQL))
	if err != nil {
		log.Fatalf("Failed to execute migration: %v", err)
	}

	fmt.Println("Migration successfully executed!")
}
