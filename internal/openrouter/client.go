package openrouter

import (
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
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
	s3Client       *s3.S3
	supabaseBucket string
	s3Endpoint     string
	s3Region       string
}

// Config holds configuration for the OpenRouter client
type Config struct {
	APIKey            string
	ModelID           string
	Timeout           time.Duration
	MaxRetries        int
	S3Endpoint        string
	S3AccessKeyID     string
	S3AccessKeySecret string
	SupabaseBucket    string
	S3Region          string
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

	// Initialize S3 client for Supabase storage
	var s3Client *s3.S3
	if config.S3Endpoint != "" && config.S3AccessKeyID != "" && config.S3AccessKeySecret != "" {
		// Remove /storage/v1/s3 from endpoint if present, as AWS SDK adds paths automatically
		endpoint := config.S3Endpoint

		sess := session.Must(session.NewSession(&aws.Config{
			Region:           aws.String(config.S3Region),
			Endpoint:         aws.String(endpoint + "/storage/v1/s3"),
			Credentials:      credentials.NewStaticCredentials(config.S3AccessKeyID, config.S3AccessKeySecret, ""),
			S3ForcePathStyle: aws.Bool(true),
			DisableSSL:       aws.Bool(false),
		}))
		s3Client = s3.New(sess)
	}

	return &Client{
		apiKey:         config.APIKey,
		apiURL:         "https://openrouter.ai/api/v1/chat/completions",
		modelID:        config.ModelID,
		s3Client:       s3Client,
		supabaseBucket: config.SupabaseBucket,
		s3Endpoint:     config.S3Endpoint,
		s3Region:       config.S3Region,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}
