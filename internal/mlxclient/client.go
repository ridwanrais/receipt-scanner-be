package mlxclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ridwanfathin/invoice-processor-service/internal/domain"
)

// Client represents a client for the MLX-VLM service
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Config holds configuration for the MLX client
type Config struct {
	BaseURL string
	Timeout time.Duration
}

// NewClient creates a new MLX-VLM client
func NewClient(config *Config) *Client {
	if config.Timeout == 0 {
		config.Timeout = 300 * time.Second // 5 minutes default
	}

	return &Client{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// ExtractInvoiceData extracts structured data from an invoice image URL using MLX-VLM
func (c *Client) ExtractInvoiceData(imageURL string) (*domain.Invoice, error) {
	// Create JSON payload
	payload := map[string]string{
		"image_url": imageURL,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON payload: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/extract", c.baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MLX service error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var invoice domain.Invoice
	if err := json.Unmarshal(respBody, &invoice); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &invoice, nil
}

// HealthCheck checks if the MLX service is healthy
func (c *Client) HealthCheck() error {
	url := fmt.Sprintf("%s/health", c.baseURL)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("health check failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}
