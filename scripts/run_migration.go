package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

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

	// Get all migration files
	migrationsDir := "scripts/migrations"
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		log.Fatalf("Unable to read migrations directory: %v", err)
	}

	// Filter and sort SQL files
	var migrationFiles []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".sql" {
			migrationFiles = append(migrationFiles, file.Name())
		}
	}
	sort.Strings(migrationFiles)

	if len(migrationFiles) == 0 {
		log.Println("No migration files found")
		return
	}

	// Execute each migration file
	for _, filename := range migrationFiles {
		migrationPath := filepath.Join(migrationsDir, filename)
		log.Printf("Executing migration: %s", filename)

		migrationSQL, err := os.ReadFile(migrationPath)
		if err != nil {
			log.Fatalf("Unable to read migration file %s: %v", filename, err)
		}

		_, err = pool.Exec(context.Background(), string(migrationSQL))
		if err != nil {
			log.Printf("Warning: Migration %s failed: %v", filename, err)
			log.Printf("Continuing with next migration...")
			continue
		}

		log.Printf("✓ Successfully executed: %s", filename)
	}

	fmt.Println("\nAll migrations completed!")
}
