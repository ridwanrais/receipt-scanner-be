package openrouter

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

// UploadImageToSupabase uploads an image to Supabase storage and returns the public URL
func (c *Client) UploadImageToSupabase(imageData []byte, filename string) (string, error) {
	// Check if Supabase URL is configured
	if c.supabaseURL == "" {
		return "", &OpenRouterError{
			Op:  "check_supabase_config",
			Err: fmt.Errorf("Supabase URL is not configured"),
		}
	}

	// Check if Supabase API key is configured
	if c.supabaseAPIKey == "" {
		return "", &OpenRouterError{
			Op:  "check_supabase_config",
			Err: fmt.Errorf("Supabase API key is not configured"),
		}
	}

	// Construct the Supabase storage URL
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", c.supabaseURL, c.supabaseBucket, filename)

	// Create a new HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(imageData))
	if err != nil {
		return "", &OpenRouterError{
			Op:  "create_upload_request",
			Err: fmt.Errorf("failed to create request: %w", err),
		}
	}

	// Set headers
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Authorization", "Bearer "+c.supabaseAPIKey)

	// Send the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", &OpenRouterError{
			Op:  "send_upload_request",
			Err: fmt.Errorf("failed to send request: %w", err),
		}
	}
	defer resp.Body.Close()

	// Check for error status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", &OpenRouterError{
			Op:  "check_upload_response",
			Err: fmt.Errorf("API error: %s - %s", resp.Status, string(respBody)),
		}
	}

	// Return the public URL of the uploaded image
	return fmt.Sprintf("%s/storage/v1/object/public/%s/%s", c.supabaseURL, c.supabaseBucket, filename), nil
}
