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

	// Supabase S3-compatible storage configuration
	SupabaseS3Endpoint      string
	SupabaseAccessKeyID     string
	SupabaseAccessKeySecret string
	SupabaseBucket          string
	SupabaseRegion          string
	PostgresDBURL           string

	// MLX Service configuration
	UseMLXService bool
	MLXServiceURL string
	MLXTimeout    time.Duration

	// Application configuration
	MaxWorkers  int
	APIBasePath string

	// Logging configuration
	LogFormat string // "json" or "pretty"
	LogLevel  string // "debug", "info", "warn", "error"

	// Authentication configuration
	GoogleClientIDWeb     string // Web OAuth client (for future web support)
	GoogleClientSecretWeb string // Web OAuth client secret
	GoogleRedirectURLWeb  string // Web OAuth redirect URL
	GoogleClientIDAndroid string // Android OAuth client ID
	GoogleClientIDIOS     string // iOS OAuth client ID
	JWTSecret             string
	JWTAccessExpiration   time.Duration
	JWTRefreshExpiration  time.Duration
	FrontendURL           string
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

		SupabaseS3Endpoint:      os.Getenv("SUPABASE_S3_ENDPOINT"),
		SupabaseAccessKeyID:     os.Getenv("SUPABASE_ACCESS_KEY_ID"),
		SupabaseAccessKeySecret: os.Getenv("SUPABASE_ACCESS_KEY_SECRET"),
		SupabaseBucket:          getEnvString("SUPABASE_BUCKET", "invoice-images"),
		SupabaseRegion:          getEnvString("SUPABASE_REGION", "ap-southeast-1"),
		PostgresDBURL:           os.Getenv("POSTGRES_DB_URL"),

		UseMLXService: getEnvString("USE_MLX_SERVICE", "false") == "true",
		MLXServiceURL: getEnvString("MLX_SERVICE_URL", "http://localhost:8000"),
		MLXTimeout:    time.Duration(getEnvInt("MLX_TIMEOUT", 300)) * time.Second,

		MaxWorkers:  getEnvInt("MAX_WORKERS", 5),
		APIBasePath: getEnvString("API_BASE_PATH", "/v1"),

		LogFormat: getEnvString("LOG_FORMAT", "json"),
		LogLevel:  getEnvString("LOG_LEVEL", "info"),

		GoogleClientIDWeb:     os.Getenv("GOOGLE_CLIENT_ID_WEB"),
		GoogleClientSecretWeb: os.Getenv("GOOGLE_CLIENT_SECRET_WEB"),
		GoogleRedirectURLWeb:  getEnvString("GOOGLE_REDIRECT_URL_WEB", "http://localhost:8080/v1/auth/google/callback"),
		GoogleClientIDAndroid: os.Getenv("GOOGLE_CLIENT_ID_ANDROID"),
		GoogleClientIDIOS:     os.Getenv("GOOGLE_CLIENT_ID_IOS"),
		JWTSecret:             getEnvString("JWT_SECRET", "your-secret-key-change-in-production"),
		JWTAccessExpiration:   time.Duration(getEnvInt("JWT_ACCESS_EXPIRATION_HOURS", 24)) * time.Hour,
		JWTRefreshExpiration:  time.Duration(getEnvInt("JWT_REFRESH_EXPIRATION_DAYS", 30)) * 24 * time.Hour,
		FrontendURL:           getEnvString("FRONTEND_URL", "http://localhost:3000"),
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
