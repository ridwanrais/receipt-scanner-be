package openrouter

import (
	"net/http"
	"time"
)

// OpenRouterError represents an error that occurred during OpenRouter API interaction
type OpenRouterError struct {
	Op  string // Operation that caused the error
	Err error  // Original error
}

// Error implements the error interface
func (e *OpenRouterError) Error() string {
	if e.Err == nil {
		return "openrouter error: " + e.Op
	}
	return "openrouter error: " + e.Op + ": " + e.Err.Error()
}

// Unwrap returns the underlying error
func (e *OpenRouterError) Unwrap() error {
	return e.Err
}

// Client represents a client for the OpenRouter API
type Client struct {
	apiKey         string
	apiURL         string
	httpClient     *http.Client
	modelID        string
	supabaseURL    string
	supabaseBucket string
	supabaseAPIKey string
}

// Config holds configuration for the OpenRouter client
type Config struct {
	APIKey         string
	ModelID        string
	Timeout        time.Duration
	MaxRetries     int
	SupabaseURL    string
	SupabaseBucket string
	SupabaseAPIKey string
}

// DefaultConfig returns a default configuration for the OpenRouter client
func DefaultConfig() *Config {
	return &Config{
		ModelID:        "meta-llama/llama-3.2-11b-vision-instruct:free",
		Timeout:        60 * time.Second,
		MaxRetries:     3,
		SupabaseBucket: "invoices",
	}
}

// NewClient creates a new OpenRouter client
func NewClient(config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	return &Client{
		apiKey:         config.APIKey,
		apiURL:         "https://openrouter.ai/api/v1/chat/completions",
		modelID:        config.ModelID,
		supabaseURL:    config.SupabaseURL,
		supabaseBucket: config.SupabaseBucket,
		supabaseAPIKey: config.SupabaseAPIKey,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}
