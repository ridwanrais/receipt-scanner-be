package database

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresDB manages the database connection to PostgreSQL
type PostgresDB struct {
	pool *pgxpool.Pool
}

// NewPostgresDB creates a new connection to PostgreSQL
func NewPostgresDB() (*PostgresDB, error) {
	// Get database URL from environment variables
	dbURL := os.Getenv("POSTGRES_DB_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("POSTGRES_DB_URL environment variable is not set")
	}

	// Create a connection pool
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Establish the connection pool
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresDB{pool: pool}, nil
}

// Close closes the database connection pool
func (db *PostgresDB) Close() {
	if db.pool != nil {
		db.pool.Close()
	}
}

// GetPool returns the connection pool for direct use
func (db *PostgresDB) GetPool() *pgxpool.Pool {
	return db.pool
}

// ExecuteTransaction executes a transaction with the provided callback function
func (db *PostgresDB) ExecuteTransaction(ctx context.Context, txFunc func(pgx.Tx) error) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Execute the transaction function
	if err := txFunc(tx); err != nil {
		// Rollback on error
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("error: %v, rollback error: %w", err, rbErr)
		}
		return err
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
