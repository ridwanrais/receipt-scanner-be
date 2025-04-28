package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReceipt represents a receipt in the API
type TestReceipt struct {
	ID        string       `json:"id,omitempty"`
	Merchant  string       `json:"merchant"`
	Date      string       `json:"date"`
	Total     string       `json:"total"`
	Tax       string       `json:"tax,omitempty"`
	Subtotal  string       `json:"subtotal,omitempty"`
	ImageURL  string       `json:"imageUrl,omitempty"`
	Items     []TestItem   `json:"items"`
	CreatedAt string       `json:"createdAt,omitempty"`
	UpdatedAt string       `json:"updatedAt,omitempty"`
}

// TestItem represents an item in a receipt
type TestItem struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name"`
	Quantity int    `json:"qty"`
	Price    string `json:"price"`
	Category string `json:"category,omitempty"`
}

// TestPagination represents pagination data in API responses
type TestPagination struct {
	TotalItems  int `json:"totalItems"`
	TotalPages  int `json:"totalPages"`
	CurrentPage int `json:"currentPage"`
	Limit       int `json:"limit"`
}

// TestReceiptListResponse represents the response from GET /receipts
type TestReceiptListResponse struct {
	Data       []TestReceipt  `json:"data"`
	Pagination TestPagination `json:"pagination"`
}

// TestReceiptAPI tests the Receipt API endpoints
func TestReceiptAPI(t *testing.T) {
	// Configure base URL - use environment variable or default
	baseURL := os.Getenv("API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080/v1"
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Variables to store data between tests
	var testReceiptID string

	// 1. Test creating a receipt
	t.Run("CreateReceipt", func(t *testing.T) {
		// Create a test receipt
		receiptInput := map[string]interface{}{
			"merchant": "Test Supermarket",
			"date":     time.Now().Format(time.RFC3339),
			"total":    42.86,
			"tax":      3.52,
			"subtotal": 39.34,
			"items": []map[string]interface{}{
				{
					"name":     "Organic Milk",
					"qty":      2,
					"price":    3.99,
					"category": "Groceries",
				},
				{
					"name":     "Whole Wheat Bread",
					"qty":      1,
					"price":    4.29,
					"category": "Groceries",
				},
			},
		}

		// Convert to JSON
		requestBody, err := json.Marshal(receiptInput)
		require.NoError(t, err, "Failed to marshal receipt input")

		// Create request
		url := fmt.Sprintf("%s/receipts", baseURL)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(requestBody))
		require.NoError(t, err, "Failed to create request")
		req.Header.Set("Content-Type", "application/json")

		// Execute request
		resp, err := client.Do(req)
		require.NoError(t, err, "Failed to execute request")
		defer resp.Body.Close()

		// Assert response status code
		assert.Equal(t, http.StatusCreated, resp.StatusCode, "Expected status code 201")

		// Read the response body for error details if status is not 201
		if resp.StatusCode != http.StatusCreated {
			bodyBytes, err := io.ReadAll(resp.Body)
			if err == nil {
				t.Logf("Response body: %s", string(bodyBytes))
			}
			// Create a new reader with the same bytes for the next decoder to use
			resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// Parse response
		var createdReceipt map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&createdReceipt)
		require.NoError(t, err, "Failed to decode response body")

		// Verify receipt data
		assert.NotEmpty(t, createdReceipt["id"], "Receipt ID should not be empty")
		assert.Equal(t, receiptInput["merchant"], createdReceipt["merchant"], "Merchant doesn't match")
		assert.NotEmpty(t, createdReceipt["createdAt"], "createdAt should not be empty")
		assert.NotEmpty(t, createdReceipt["items"], "items should not be empty")

		// Store receipt ID for later tests
		testReceiptID = createdReceipt["id"].(string)
		t.Logf("Created test receipt with ID: %s", testReceiptID)
	})

	// 2. Test scanning a receipt
	t.Run("ScanReceipt", func(t *testing.T) {
		// Skip test if SUPABASE_URL is not configured
		if os.Getenv("SUPABASE_URL") == "" {
			t.Skip("Skipping ScanReceipt test as SUPABASE_URL is not configured")
		}
		
		// Prepare a test image - use a sample receipt image
		// Use the PNG file provided by the user
		imagePath := "../../testdata/sample_receipt.png"
		
		// Skip test if file doesn't exist
		if _, err := os.Stat(imagePath); os.IsNotExist(err) {
			t.Skip("Test image not found, skipping scan receipt test")
			return
		}

		// Create a buffer to write the multipart form
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		// Create form file
		fileWriter, err := writer.CreateFormFile("receiptImage", filepath.Base(imagePath))
		require.NoError(t, err, "Failed to create form file")

		// Open test file
		file, err := os.Open(imagePath)
		require.NoError(t, err, "Failed to open test image")
		defer file.Close()

		// Copy the file data to the form
		_, err = io.Copy(fileWriter, file)
		require.NoError(t, err, "Failed to copy file to form")

		// Close the multipart writer
		err = writer.Close()
		require.NoError(t, err, "Failed to close multipart writer")

		// Create request
		url := fmt.Sprintf("%s/receipts/scan", baseURL)
		req, err := http.NewRequest(http.MethodPost, url, &buf)
		require.NoError(t, err, "Failed to create request")
		req.Header.Set("Content-Type", writer.FormDataContentType())

		// Execute request
		resp, err := client.Do(req)
		require.NoError(t, err, "Failed to execute request")
		defer resp.Body.Close()

		// Check response status (may be 200 OK or 422 if image can't be processed)
		if resp.StatusCode == http.StatusOK {
			// Parse response
			var scannedReceipt map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&scannedReceipt)
			require.NoError(t, err, "Failed to decode response body")

			// Verify receipt data
			assert.NotEmpty(t, scannedReceipt["id"], "Receipt ID should not be empty")
			
			// If we didn't have a test receipt ID yet, store this one
			if testReceiptID == "" {
				testReceiptID = scannedReceipt["id"].(string)
				t.Logf("Using scanned receipt with ID: %s for subsequent tests", testReceiptID)
			}
		} else if resp.StatusCode == http.StatusUnprocessableEntity {
			// If image processing fails, this is acceptable for this test
			// We just want to ensure the endpoint doesn't crash
			t.Log("Receipt image could not be processed (status 422)")
		} else {
			// Any other status is an error
			t.Errorf("Unexpected status code: %d", resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			t.Logf("Response body: %s", body)
		}
	})

	// Skip the remaining tests if we don't have a test receipt ID
	if testReceiptID == "" {
		t.Log("No test receipt ID available, skipping remaining tests")
		return
	}

	// 3. Test listing receipts
	t.Run("GetReceipts", func(t *testing.T) {
		// Create request
		url := fmt.Sprintf("%s/receipts", baseURL)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		require.NoError(t, err, "Failed to create request")

		// Execute request
		resp, err := client.Do(req)
		require.NoError(t, err, "Failed to execute request")
		defer resp.Body.Close()

		// Assert response status code
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")

		// Parse response
		var response TestReceiptListResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err, "Failed to decode response body")

		// Verify data is not empty
		assert.NotEmpty(t, response.Data, "Data should not be empty")

		// Verify pagination
		assert.GreaterOrEqual(t, response.Pagination.TotalItems, 1, "Should have at least one receipt")
		assert.GreaterOrEqual(t, response.Pagination.TotalPages, 1, "Should have at least one page")
		assert.GreaterOrEqual(t, response.Pagination.CurrentPage, 1, "Current page should be at least 1")
		assert.GreaterOrEqual(t, response.Pagination.Limit, 1, "Limit should be at least 1")
	})

	// 4. Test getting a receipt by ID
	t.Run("GetReceiptByID", func(t *testing.T) {
		// Create request
		url := fmt.Sprintf("%s/receipts/%s", baseURL, testReceiptID)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		require.NoError(t, err, "Failed to create request")

		// Execute request
		resp, err := client.Do(req)
		require.NoError(t, err, "Failed to execute request")
		defer resp.Body.Close()

		// Assert response status code
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")

		// Parse response
		var receipt TestReceipt
		err = json.NewDecoder(resp.Body).Decode(&receipt)
		require.NoError(t, err, "Failed to decode response body")

		// Verify receipt data
		assert.Equal(t, testReceiptID, receipt.ID, "Receipt ID doesn't match")
		assert.NotEmpty(t, receipt.Merchant, "Merchant should not be empty")
		assert.NotEmpty(t, receipt.Date, "Date should not be empty")
		assert.NotEmpty(t, receipt.Items, "Items should not be empty")
	})

	// 5. Test getting receipt items
	t.Run("GetReceiptItems", func(t *testing.T) {
		// Create request
		url := fmt.Sprintf("%s/receipts/%s/items", baseURL, testReceiptID)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		require.NoError(t, err, "Failed to create request")

		// Execute request
		resp, err := client.Do(req)
		require.NoError(t, err, "Failed to execute request")
		defer resp.Body.Close()

		// Assert response status code
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")

		// Parse response
		var items []TestItem
		err = json.NewDecoder(resp.Body).Decode(&items)
		require.NoError(t, err, "Failed to decode response body")

		// Verify items
		assert.NotEmpty(t, items, "Items should not be empty")
		assert.NotEmpty(t, items[0].Name, "Item should have a name")
	})

	// 6. Test updating a receipt
	t.Run("UpdateReceipt", func(t *testing.T) {
		// Create an update request
		updateInput := map[string]interface{}{
			"merchant": "Updated Test Market",
			"date":     time.Now().Format(time.RFC3339),
			"total":    45.99,
			"tax":      4.20,
			"subtotal": 41.79,
			"items": []map[string]interface{}{
				{
					"name":     "Organic Eggs",
					"qty":      1,
					"price":    5.99,
					"category": "Groceries",
				},
				{
					"name":     "Fresh Apples",
					"qty":      3,
					"price":    1.29,
					"category": "Produce",
				},
			},
		}

		// Convert to JSON
		requestBody, err := json.Marshal(updateInput)
		require.NoError(t, err, "Failed to marshal update payload")

		// Create request
		url := fmt.Sprintf("%s/receipts/%s", baseURL, testReceiptID)
		req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(requestBody))
		require.NoError(t, err, "Failed to create request")
		req.Header.Set("Content-Type", "application/json")

		// Execute request
		resp, err := client.Do(req)
		require.NoError(t, err, "Failed to execute request")
		defer resp.Body.Close()

		// Assert response status code
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")

		// Parse response
		var updatedReceipt TestReceipt
		err = json.NewDecoder(resp.Body).Decode(&updatedReceipt)
		require.NoError(t, err, "Failed to decode response body")

		// Verify updated data
		assert.Equal(t, testReceiptID, updatedReceipt.ID, "Receipt ID should match")
		assert.Equal(t, updateInput["merchant"], updatedReceipt.Merchant, "Merchant name should be updated")
	})

	// 7. Test dashboard summary
	t.Run("GetDashboardSummary", func(t *testing.T) {
		// Create request
		url := fmt.Sprintf("%s/dashboard/summary", baseURL)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		require.NoError(t, err, "Failed to create request")

		// Execute request
		resp, err := client.Do(req)
		require.NoError(t, err, "Failed to execute request")
		defer resp.Body.Close()

		// Assert response status code
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")

		// Parse response
		var summary map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&summary)
		require.NoError(t, err, "Failed to decode response body")

		// Verify response structure
		assert.Contains(t, summary, "totalSpend", "Summary should contain totalSpend")
		assert.Contains(t, summary, "receiptCount", "Summary should contain receiptCount")
		assert.Contains(t, summary, "averageSpend", "Summary should contain averageSpend")
		assert.Contains(t, summary, "topCategories", "Summary should contain topCategories")
		assert.Contains(t, summary, "topMerchants", "Summary should contain topMerchants")
	})

	// 8. Test spending trends
	t.Run("GetSpendingTrends", func(t *testing.T) {
		// Test different period values
		periods := []string{"daily", "weekly", "monthly", "yearly"}

		for _, period := range periods {
			// Create request with period parameter
			url := fmt.Sprintf("%s/dashboard/spending-trends?period=%s", baseURL, period)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err, "Failed to create request")

			// Execute request
			resp, err := client.Do(req)
			require.NoError(t, err, "Failed to execute request")
			defer resp.Body.Close()

			// Assert response status code
			assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200 for period: "+period)

			// Parse response
			var trends map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&trends)
			require.NoError(t, err, "Failed to decode response body for period: "+period)

			// Verify response structure
			assert.Equal(t, period, trends["period"], "Period should match requested period")
			assert.Contains(t, trends, "data", "Trends should contain data array")
		}
	})

	// 9. Test spending by category
	t.Run("GetSpendingByCategory", func(t *testing.T) {
		// Create request
		url := fmt.Sprintf("%s/insights/spending-by-category", baseURL)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		require.NoError(t, err, "Failed to create request")

		// Execute request
		resp, err := client.Do(req)
		require.NoError(t, err, "Failed to execute request")
		defer resp.Body.Close()

		// Assert response status code
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")

		// Parse response
		var categorySpending map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&categorySpending)
		require.NoError(t, err, "Failed to decode response body")

		// Verify response structure
		assert.Contains(t, categorySpending, "total", "Category spending should contain total")
		assert.Contains(t, categorySpending, "categories", "Category spending should contain categories array")
	})

	// 10. Test merchant frequency
	t.Run("GetMerchantFrequency", func(t *testing.T) {
		// Create request
		url := fmt.Sprintf("%s/insights/merchant-frequency", baseURL)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		require.NoError(t, err, "Failed to create request")

		// Execute request
		resp, err := client.Do(req)
		require.NoError(t, err, "Failed to execute request")
		defer resp.Body.Close()

		// Assert response status code
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")

		// Parse response
		var merchantFrequency map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&merchantFrequency)
		require.NoError(t, err, "Failed to decode response body")

		// Verify response structure
		assert.Contains(t, merchantFrequency, "totalVisits", "Merchant frequency should contain totalVisits")
		assert.Contains(t, merchantFrequency, "merchants", "Merchant frequency should contain merchants array")
	})

	// 11. Test monthly comparison
	t.Run("GetMonthlyComparison", func(t *testing.T) {
		// Create two months to compare (current month and previous month)
		now := time.Now()
		currentMonth := now.Format("2006-01")
		
		// Calculate previous month
		previousMonth := now.AddDate(0, -1, 0).Format("2006-01")

		// Create request
		url := fmt.Sprintf("%s/insights/monthly-comparison?month1=%s&month2=%s", 
			baseURL, previousMonth, currentMonth)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		require.NoError(t, err, "Failed to create request")

		// Execute request
		resp, err := client.Do(req)
		require.NoError(t, err, "Failed to execute request")
		defer resp.Body.Close()

		// Assert response status code
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")

		// Parse response
		var comparison map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&comparison)
		require.NoError(t, err, "Failed to decode response body")

		// Verify response structure
		assert.Equal(t, previousMonth, comparison["month1"], "month1 should match requested month1")
		assert.Equal(t, currentMonth, comparison["month2"], "month2 should match requested month2")
		assert.Contains(t, comparison, "month1Total", "Comparison should contain month1Total")
		assert.Contains(t, comparison, "month2Total", "Comparison should contain month2Total")
		assert.Contains(t, comparison, "difference", "Comparison should contain difference")
		assert.Contains(t, comparison, "percentageChange", "Comparison should contain percentageChange")
		assert.Contains(t, comparison, "categories", "Comparison should contain categories array")
	})

	// 12. Test deleting a receipt
	t.Run("DeleteReceipt", func(t *testing.T) {
		// Create request
		url := fmt.Sprintf("%s/receipts/%s", baseURL, testReceiptID)
		req, err := http.NewRequest(http.MethodDelete, url, nil)
		require.NoError(t, err, "Failed to create request")

		// Execute request
		resp, err := client.Do(req)
		require.NoError(t, err, "Failed to execute request")
		defer resp.Body.Close()

		// Assert response status code
		assert.Equal(t, http.StatusNoContent, resp.StatusCode, "Expected status code 204")

		// Try to fetch the deleted receipt - should return 404
		getReq, err := http.NewRequest(http.MethodGet, url, nil)
		require.NoError(t, err, "Failed to create request")

		getResp, err := client.Do(getReq)
		require.NoError(t, err, "Failed to execute request")
		defer getResp.Body.Close()

		// Assert response status code should be 404 (not found)
		assert.Equal(t, http.StatusNotFound, getResp.StatusCode, "Expected status code 404 after deletion")
	})
}
