package config

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	// Server configuration
	Port       int
	MaxWorkers int

	// AI processor configuration
	OpenRouterAPIKey  string
	OpenRouterModelID string
	OpenRouterTimeout time.Duration

	// Storage configuration
	SupabaseURL    string
	SupabaseBucket string
	SupabaseAPIKey string
}

// LoadConfig loads the application configuration from environment variables
func LoadConfig() (*Config, error) {
	// Get the executable directory
	execPath, err := os.Executable()
	if err != nil {
		log.Printf("Warning: Could not determine executable path: %v", err)
	}

	// Determine project root directory
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(execPath)))
	envPath := filepath.Join(projectRoot, ".env")

	// Load .env file if it exists
	if err := godotenv.Load(envPath); err != nil {
		// Try loading from current directory as fallback
		if err := godotenv.Load(); err != nil {
			log.Println("No .env file found or error loading .env file. Using environment variables.")
		} else {
			log.Println("Loaded environment variables from current directory .env file")
		}
	} else {
		log.Printf("Loaded environment variables from %s", envPath)
	}

	// Create and populate config
	config := &Config{
		// Server configuration
		Port:       getEnvInt("PORT", 8080),
		MaxWorkers: getEnvInt("MAX_WORKERS", 5),

		// AI processor configuration
		OpenRouterAPIKey:  os.Getenv("OPENROUTER_API_KEY"),
		OpenRouterModelID: getEnvString("OPENROUTER_MODEL_ID", "meta-llama/llama-3.2-11b-vision-instruct:free"),
		OpenRouterTimeout: time.Duration(getEnvInt("OPENROUTER_TIMEOUT", 60)) * time.Second,

		// Storage configuration
		SupabaseURL:    os.Getenv("SUPABASE_URL"),
		SupabaseBucket: getEnvString("SUPABASE_BUCKET", "invoices"),
		SupabaseAPIKey: os.Getenv("SUPABASE_API_KEY"),
	}

	// Validate critical configuration
	validateConfig(config)

	return config, nil
}

// validateConfig checks if critical configuration values are set and logs warnings if they're missing
func validateConfig(config *Config) {
	// Check if OpenRouter API key is provided
	if config.OpenRouterAPIKey == "" {
		log.Println("Warning: No OpenRouter API key provided. API requests will fail.")
	}

	// Check if Supabase URL is provided
	if config.SupabaseURL == "" {
		log.Println("Warning: No Supabase URL provided. Image uploads will fail.")
	}

	// Check if Supabase API key is provided
	if config.SupabaseAPIKey == "" {
		log.Println("Warning: No Supabase API key provided. Image uploads will fail.")
	}
}

// getEnvInt gets an integer from an environment variable with a default value
func getEnvInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Invalid value for %s: %s, using default: %d", key, valueStr, defaultValue)
		return defaultValue
	}

	return value
}

// getEnvBool gets a boolean from an environment variable with a default value
func getEnvBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	valueStr = strings.ToLower(valueStr)
	return valueStr == "true" || valueStr == "1" || valueStr == "yes"
}

// getEnvString gets a string from an environment variable with a default value
func getEnvString(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvStringSlice gets a string slice from a comma-separated environment variable
func getEnvStringSlice(key string, defaultValue []string) []string {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	return strings.Split(valueStr, ",")
}
