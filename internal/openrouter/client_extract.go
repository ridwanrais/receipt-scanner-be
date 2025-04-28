package openrouter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ridwanfathin/invoice-processor-service/internal/domain"
)

// ExtractInvoiceData extracts structured data from an invoice image
func (c *Client) ExtractInvoiceData(imageData []byte) (*domain.Invoice, error) {
	// Check for required configuration
	if c.supabaseURL == "" {
		return nil, &OpenRouterError{
			Op:  "validate_configuration",
			Err: fmt.Errorf("Supabase URL is not configured. Please set SUPABASE_URL environment variable"),
		}
	}

	if c.supabaseAPIKey == "" {
		return nil, &OpenRouterError{
			Op:  "validate_configuration",
			Err: fmt.Errorf("Supabase API key is not configured. Please set SUPABASE_API_KEY environment variable"),
		}
	}

	if c.apiKey == "" {
		return nil, &OpenRouterError{
			Op:  "validate_configuration",
			Err: fmt.Errorf("OpenRouter API key is not configured. Please set OPENROUTER_API_KEY environment variable"),
		}
	}

	// Generate a unique filename
	timestamp := time.Now().UnixNano()
	filename := fmt.Sprintf("invoice_%d.png", timestamp)

	// Upload the image to Supabase
	imageURL, err := c.UploadImageToSupabase(imageData, filename)
	if err != nil {
		return nil, &OpenRouterError{
			Op:  "upload_image",
			Err: fmt.Errorf("failed to upload image to Supabase: %w", err),
		}
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
		Type     string    `json:"type"`
		Text     string    `json:"text,omitempty"`
		ImageURL *ImageURL `json:"image_url,omitempty"`
	}

	// Create the system prompt
	systemContent := Content{
		Type: "text",
		Text: `You are an invoice data extraction assistant. Extract the following information from the invoice image:
- Vendor name
- Invoice number
- Invoice date (in YYYY-MM-DD format)
- Due date (in YYYY-MM-DD format)
- Line items (including description, details, quantity, unit price, total, and category for each)
- Subtotal
- Tax rate percentage
- Tax amount
- Discount (if any)
- Total due amount

Format your response as a valid JSON object with the following structure:
{
  "vendor_name": "...",
  "invoice_number": "...",
  "invoice_date": "YYYY-MM-DD",
  "due_date": "YYYY-MM-DD",
  "items": [
    {
      "description": "...",
      "details": ["...", "..."],
      "quantity": 0.0,
      "unit_price": 0.0,
      "total": 0.0,
      "category": "..."
    }
  ],
  "subtotal": 0.0,
  "tax_rate_percent": 0.0,
  "tax_amount": 0.0,
  "discount": 0.0,
  "total_due": 0.0
}

For each line item, if you can infer the category (e.g. "Food", "Office Supplies", "Travel", etc.) from the description, provide it. If not, leave it as an empty string "".

Do not include any other text in your response, only provide the JSON.`,
	}

	// Create the user message with the image
	userContent := []Content{
		{
			Type: "text",
			Text: "Extract the data from this invoice image.",
		},
		{
			Type:     "image_url",
			ImageURL: &ImageURL{URL: imageURL},
		},
	}

	// Convert userContent to []interface{}
	var userContentInterface []interface{}
	for _, item := range userContent {
		userContentInterface = append(userContentInterface, item)
	}

	// Create the request payload
	requestPayload := map[string]interface{}{
		"model": c.modelID,
		"messages": []Message{
			{
				Role:    "system",
				Content: []interface{}{systemContent},
			},
			{
				Role:    "user",
				Content: userContentInterface,
			},
		},
	}

	// Convert the request payload to JSON
	requestData, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, &OpenRouterError{
			Op:  "marshal_request",
			Err: fmt.Errorf("failed to marshal request payload: %w", err),
		}
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", c.apiURL, bytes.NewBuffer(requestData))
	if err != nil {
		return nil, &OpenRouterError{
			Op:  "create_extract_request",
			Err: fmt.Errorf("failed to create request: %w", err),
		}
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("HTTP-Referer", "https://github.com/ridwanfathin/invoice-processor-service")

	// Send the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &OpenRouterError{
			Op:  "send_extract_request",
			Err: fmt.Errorf("failed to send request: %w", err),
		}
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &OpenRouterError{
			Op:  "read_response",
			Err: fmt.Errorf("failed to read response body: %w", err),
		}
	}

	// Check for error status code
	if resp.StatusCode != http.StatusOK {
		return nil, &OpenRouterError{
			Op:  "check_api_response",
			Err: fmt.Errorf("API error: %s - %s", resp.Status, string(respBody)),
		}
	}

	// Parse the response and extract the invoice data
	return c.parseOpenRouterResponse(respBody)
}
