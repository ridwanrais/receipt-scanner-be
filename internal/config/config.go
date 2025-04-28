package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	// Server configuration
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	// OpenRouter configuration
	OpenRouterAPIKey  string
	OpenRouterModelID string
	OpenRouterTimeout time.Duration

	// Supabase configuration
	SupabaseURL    string
	SupabaseAPIKey string
	SupabaseBucket string
	PostgresDBURL  string

	// Application configuration
	MaxWorkers  int
	APIBasePath string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err == nil {
		fmt.Println("Loaded environment variables from current directory .env file")
	}

	// Load configuration with defaults
	config := &Config{
		Port:         getEnvInt("PORT", 8080),
		ReadTimeout:  time.Duration(getEnvInt("READ_TIMEOUT_SECONDS", 30)) * time.Second,
		WriteTimeout: time.Duration(getEnvInt("WRITE_TIMEOUT_SECONDS", 30)) * time.Second,

		OpenRouterAPIKey:  os.Getenv("OPENROUTER_API_KEY"),
		OpenRouterModelID: getEnvString("OPENROUTER_MODEL_ID", "mistralai/mistral-7b-instruct"),
		OpenRouterTimeout: time.Duration(getEnvInt("OPENROUTER_TIMEOUT", 60)) * time.Second,

		SupabaseURL:    os.Getenv("SUPABASE_URL"),
		SupabaseAPIKey: os.Getenv("SUPABASE_API_KEY"),
		SupabaseBucket: getEnvString("SUPABASE_BUCKET", "receipts"),
		PostgresDBURL:  os.Getenv("POSTGRES_DB_URL"),

		MaxWorkers:  getEnvInt("MAX_WORKERS", 5),
		APIBasePath: getEnvString("API_BASE_PATH", "/v1"),
	}

	return config, nil
}

// getEnvInt gets an environment variable as an integer with a default value
func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return intValue
}

// getEnvString gets an environment variable with a default value
func getEnvString(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
