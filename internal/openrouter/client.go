package openrouter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ridwanfathin/invoice-ocr-service/internal/model"
)

// Client represents a client for the OpenRouter API
type Client struct {
	apiKey          string
	apiURL          string
	httpClient      *http.Client
	modelID         string
	supabaseURL     string
	supabaseBucket  string
	supabaseAPIKey  string
}

// Config holds configuration for the OpenRouter client
type Config struct {
	APIKey          string
	ModelID         string
	Timeout         time.Duration
	MaxRetries      int
	SupabaseURL     string
	SupabaseBucket  string
	SupabaseAPIKey  string
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
		apiKey:          config.APIKey,
		apiURL:          "https://openrouter.ai/api/v1/chat/completions",
		modelID:         config.ModelID,
		supabaseURL:     config.SupabaseURL,
		supabaseBucket:  config.SupabaseBucket,
		supabaseAPIKey:  config.SupabaseAPIKey,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// UploadImageToSupabase uploads an image to Supabase storage and returns the public URL
func (c *Client) UploadImageToSupabase(imageData []byte, filename string) (string, error) {
	// Construct the Supabase storage URL
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", c.supabaseURL, c.supabaseBucket, filename)
	
	// Create a new HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(imageData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Authorization", "Bearer "+c.supabaseAPIKey)
	
	// Send the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	// Check for error status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error: %s - %s", resp.Status, string(respBody))
	}
	
	// Return the public URL of the uploaded image
	return fmt.Sprintf("%s/storage/v1/object/public/%s/%s", c.supabaseURL, c.supabaseBucket, filename), nil
}

// ExtractInvoiceData extracts structured data from an invoice image
func (c *Client) ExtractInvoiceData(imageData []byte) (*model.Invoice, error) {
	// Generate a unique filename
	timestamp := time.Now().UnixNano()
	filename := fmt.Sprintf("invoice_%d.png", timestamp)
	
	// Upload the image to Supabase
	imageURL, err := c.UploadImageToSupabase(imageData, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to upload image to Supabase: %w", err)
	}
	
	// Create the OpenRouter API request payload
	type Message struct {
		Role    string        `json:"role"`
		Content []interface{} `json:"content"`
	}
	
	type ImageURL struct {
		URL string `json:"url"`
	}
	
	type Content struct {
		Type     string   `json:"type"`
		Text     string   `json:"text,omitempty"`
		ImageURL *ImageURL `json:"image_url,omitempty"`
	}
	
	textContent := Content{
		Type: "text",
		Text: "Extract the following information from the invoice and return it strictly as a structured JSON object with this format: { \"vendor_name\": \"\", \"invoice_number\": \"\", \"invoice_date\": \"\", \"due_date\": \"\", \"items\": [ { \"description\": \"\", \"details\": [\"\"], \"quantity\": 0, \"unit_price\": 0.0, \"total\": 0.0 } ], \"subtotal\": 0.0, \"tax_rate_percent\": 0.0, \"tax_amount\": 0.0, \"discount\": 0.0, \"total_due\": 0.0 }. Output only valid JSON. Do not include any extra explanation or commentary.",
	}
	
	imageContent := Content{
		Type: "image_url",
		ImageURL: &ImageURL{
			URL: imageURL,
		},
	}
	
	payload := map[string]interface{}{
		"model": c.modelID,
		"messages": []Message{
			{
				Role:    "user",
				Content: []interface{}{textContent, imageContent},
			},
		},
	}
	
	// Marshal the payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}
	
	// Create a new HTTP request
	req, err := http.NewRequest("POST", c.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	
	// Send the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	// Check for error status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(respBody))
	}
	
	// Parse the response
	return c.parseOpenRouterResponse(respBody)
}

// parseOpenRouterResponse parses the JSON response from the OpenRouter API
func (c *Client) parseOpenRouterResponse(respBody []byte) (*model.Invoice, error) {
	// Parse the response JSON
	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	// Check if we have any choices
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no response content received from API")
	}
	
	// Extract the content from the first choice
	content := response.Choices[0].Message.Content
	
	// Log the raw content for debugging
	log.Printf("Raw content from OpenRouter: %s", content)
	
	// Try to extract JSON from the content
	// Sometimes the model might return text with JSON embedded in it
	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}")
	
	if jsonStart >= 0 && jsonEnd > jsonStart {
		// Extract the JSON part
		jsonContent := content[jsonStart : jsonEnd+1]
		log.Printf("Extracted JSON content: %s", jsonContent)
		
		// Parse the content as JSON
		var invoice model.Invoice
		if err := json.Unmarshal([]byte(jsonContent), &invoice); err != nil {
			// If we still can't parse it, try to extract JSON using regex
			return c.extractJSONWithRegex(content)
		}
		return &invoice, nil
	}
	
	// If no JSON braces found, try regex extraction
	return c.extractJSONWithRegex(content)
}

// extractJSONWithRegex tries to extract JSON from text using regex
func (c *Client) extractJSONWithRegex(content string) (*model.Invoice, error) {
	// Create a fallback invoice with minimal information
	invoice := &model.Invoice{
		Items: []model.LineItem{},
	}
	
	// Try to extract key fields using regex
	
	// Vendor name
	vendorRegex := regexp.MustCompile(`"vendor_name"\s*:\s*"([^"]+)"`)
	if matches := vendorRegex.FindStringSubmatch(content); len(matches) > 1 {
		invoice.VendorName = matches[1]
	}
	
	// Invoice number
	invoiceNumRegex := regexp.MustCompile(`"invoice_number"\s*:\s*"([^"]+)"`)
	if matches := invoiceNumRegex.FindStringSubmatch(content); len(matches) > 1 {
		invoice.InvoiceNumber = matches[1]
	}
	
	// Invoice date
	dateRegex := regexp.MustCompile(`"invoice_date"\s*:\s*"([^"]+)"`)
	if matches := dateRegex.FindStringSubmatch(content); len(matches) > 1 {
		invoice.InvoiceDate = matches[1]
	}
	
	// Due date
	dueDateRegex := regexp.MustCompile(`"due_date"\s*:\s*"([^"]+)"`)
	if matches := dueDateRegex.FindStringSubmatch(content); len(matches) > 1 {
		invoice.DueDate = matches[1]
	}
	
	// Total due
	totalRegex := regexp.MustCompile(`"total_due"\s*:\s*(\d+\.?\d*)`)
	if matches := totalRegex.FindStringSubmatch(content); len(matches) > 1 {
		if total, err := strconv.ParseFloat(matches[1], 64); err == nil {
			invoice.TotalDue = total
		}
	}
	
	// If we couldn't extract anything, return the original error
	if invoice.VendorName == "" && invoice.InvoiceNumber == "" && invoice.TotalDue == 0 {
		return nil, fmt.Errorf("failed to extract invoice data from model response: %s", content)
	}
	
	return invoice, nil
}
