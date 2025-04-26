package openrouter

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/ridwanfathin/invoice-processor-service/internal/domain"
)

// parseOpenRouterResponse parses the JSON response from the OpenRouter API
func (c *Client) parseOpenRouterResponse(respBody []byte) (*domain.Invoice, error) {
	// Define the response structure
	type Choice struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}

	type Response struct {
		Choices []Choice `json:"choices"`
	}

	// Parse the response
	var response Response
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, &OpenRouterError{
			Op:  "parse_response_json",
			Err: fmt.Errorf("failed to unmarshal response: %w", err),
		}
	}

	// Check if we have any choices in the response
	if len(response.Choices) == 0 {
		return nil, &OpenRouterError{
			Op:  "check_response_choices",
			Err: fmt.Errorf("no choices in response"),
		}
	}

	// Get the content from the first choice
	content := response.Choices[0].Message.Content

	// Try to parse the content as JSON directly
	var invoiceDTO struct {
		VendorName     string  `json:"vendor_name"`
		InvoiceNumber  string  `json:"invoice_number"`
		InvoiceDate    string  `json:"invoice_date"`
		DueDate        string  `json:"due_date"`
		Subtotal       float64 `json:"subtotal"`
		TaxRatePercent float64 `json:"tax_rate_percent"`
		TaxAmount      float64 `json:"tax_amount"`
		Discount       float64 `json:"discount"`
		TotalDue       float64 `json:"total_due"`
		Items          []struct {
			Description string   `json:"description"`
			Details     []string `json:"details"`
			Quantity    float64  `json:"quantity"`
			UnitPrice   float64  `json:"unit_price"`
			Total       float64  `json:"total"`
		} `json:"items"`
	}

	// First, try to parse the entire response as JSON
	err := json.Unmarshal([]byte(content), &invoiceDTO)
	if err == nil {
		// Successfully parsed JSON
		invoice := domain.NewInvoice()
		invoice.VendorName = invoiceDTO.VendorName
		invoice.InvoiceNumber = invoiceDTO.InvoiceNumber

		// Parse dates
		if invoiceDTO.InvoiceDate != "" {
			invoiceDate, err := time.Parse("2006-01-02", invoiceDTO.InvoiceDate)
			if err == nil {
				invoice.InvoiceDate = invoiceDate
			}
		}

		if invoiceDTO.DueDate != "" {
			dueDate, err := time.Parse("2006-01-02", invoiceDTO.DueDate)
			if err == nil {
				invoice.DueDate = dueDate
			}
		}

		invoice.Subtotal = invoiceDTO.Subtotal
		invoice.TaxRatePercent = invoiceDTO.TaxRatePercent
		invoice.TaxAmount = invoiceDTO.TaxAmount
		invoice.Discount = invoiceDTO.Discount
		invoice.TotalDue = invoiceDTO.TotalDue

		// Convert line items
		for _, item := range invoiceDTO.Items {
			invoice.AddLineItem(domain.LineItem{
				Description: item.Description,
				Details:     item.Details,
				Quantity:    item.Quantity,
				UnitPrice:   item.UnitPrice,
				Total:       item.Total,
			})
		}

		return invoice, nil
	}

	// If direct JSON parsing fails, try to extract JSON using regex
	log.Printf("Failed to parse response as JSON directly: %v", err)
	log.Printf("Trying to extract JSON using regex")
	return c.extractJSONWithRegex(content)
}

// extractJSONWithRegex tries to extract JSON from text using regex
func (c *Client) extractJSONWithRegex(content string) (*domain.Invoice, error) {
	// Replace all occurrences of ```json and ``` around the JSON content
	content = regexp.MustCompile("```json\\s*").ReplaceAllString(content, "")
	content = regexp.MustCompile("```\\s*").ReplaceAllString(content, "")

	// Try to find a JSON object in the content
	jsonRegex := regexp.MustCompile(`\{[\s\S]*\}`)
	jsonMatch := jsonRegex.FindString(content)

	if jsonMatch != "" {
		// Try to parse the extracted JSON
		var invoiceDTO struct {
			VendorName     string  `json:"vendor_name"`
			InvoiceNumber  string  `json:"invoice_number"`
			InvoiceDate    string  `json:"invoice_date"`
			DueDate        string  `json:"due_date"`
			Subtotal       float64 `json:"subtotal"`
			TaxRatePercent float64 `json:"tax_rate_percent"`
			TaxAmount      float64 `json:"tax_amount"`
			Discount       float64 `json:"discount"`
			TotalDue       float64 `json:"total_due"`
			Items          []struct {
				Description string   `json:"description"`
				Details     []string `json:"details"`
				Quantity    float64  `json:"quantity"`
				UnitPrice   float64  `json:"unit_price"`
				Total       float64  `json:"total"`
			} `json:"items"`
		}

		if err := json.Unmarshal([]byte(jsonMatch), &invoiceDTO); err == nil {
			// Successfully parsed JSON
			invoice := domain.NewInvoice()
			invoice.VendorName = invoiceDTO.VendorName
			invoice.InvoiceNumber = invoiceDTO.InvoiceNumber

			// Parse dates
			if invoiceDTO.InvoiceDate != "" {
				invoiceDate, err := time.Parse("2006-01-02", invoiceDTO.InvoiceDate)
				if err == nil {
					invoice.InvoiceDate = invoiceDate
				}
			}

			if invoiceDTO.DueDate != "" {
				dueDate, err := time.Parse("2006-01-02", invoiceDTO.DueDate)
				if err == nil {
					invoice.DueDate = dueDate
				}
			}

			invoice.Subtotal = invoiceDTO.Subtotal
			invoice.TaxRatePercent = invoiceDTO.TaxRatePercent
			invoice.TaxAmount = invoiceDTO.TaxAmount
			invoice.Discount = invoiceDTO.Discount
			invoice.TotalDue = invoiceDTO.TotalDue

			// Convert line items
			for _, item := range invoiceDTO.Items {
				invoice.AddLineItem(domain.LineItem{
					Description: item.Description,
					Details:     item.Details,
					Quantity:    item.Quantity,
					UnitPrice:   item.UnitPrice,
					Total:       item.Total,
				})
			}

			return invoice, nil
		}
	}

	// If JSON extraction fails, try to extract individual fields using regex
	log.Printf("Failed to extract valid JSON, attempting to extract individual fields with regex")
	invoice := domain.NewInvoice()

	// Extract vendor name
	vendorRegex := regexp.MustCompile(`"vendor_name"\s*:\s*"([^"]+)"`)
	if matches := vendorRegex.FindStringSubmatch(content); len(matches) > 1 {
		invoice.VendorName = matches[1]
	}

	// Extract invoice number
	invoiceNumRegex := regexp.MustCompile(`"invoice_number"\s*:\s*"([^"]+)"`)
	if matches := invoiceNumRegex.FindStringSubmatch(content); len(matches) > 1 {
		invoice.InvoiceNumber = matches[1]
	}

	// Extract invoice date
	invoiceDateRegex := regexp.MustCompile(`"invoice_date"\s*:\s*"([^"]+)"`)
	if matches := invoiceDateRegex.FindStringSubmatch(content); len(matches) > 1 {
		if date, err := time.Parse("2006-01-02", matches[1]); err == nil {
			invoice.InvoiceDate = date
		}
	}

	// Extract due date
	dueDateRegex := regexp.MustCompile(`"due_date"\s*:\s*"([^"]+)"`)
	if matches := dueDateRegex.FindStringSubmatch(content); len(matches) > 1 {
		if date, err := time.Parse("2006-01-02", matches[1]); err == nil {
			invoice.DueDate = date
		}
	}

	// Extract subtotal
	subtotalRegex := regexp.MustCompile(`"subtotal"\s*:\s*(\d+\.?\d*)`)
	if matches := subtotalRegex.FindStringSubmatch(content); len(matches) > 1 {
		if subtotal, err := strconv.ParseFloat(matches[1], 64); err == nil {
			invoice.Subtotal = subtotal
		}
	}

	// Extract tax rate
	taxRateRegex := regexp.MustCompile(`"tax_rate_percent"\s*:\s*(\d+\.?\d*)`)
	if matches := taxRateRegex.FindStringSubmatch(content); len(matches) > 1 {
		if taxRate, err := strconv.ParseFloat(matches[1], 64); err == nil {
			invoice.TaxRatePercent = taxRate
		}
	}

	// Extract tax amount
	taxAmountRegex := regexp.MustCompile(`"tax_amount"\s*:\s*(\d+\.?\d*)`)
	if matches := taxAmountRegex.FindStringSubmatch(content); len(matches) > 1 {
		if taxAmount, err := strconv.ParseFloat(matches[1], 64); err == nil {
			invoice.TaxAmount = taxAmount
		}
	}

	// Extract discount
	discountRegex := regexp.MustCompile(`"discount"\s*:\s*(\d+\.?\d*)`)
	if matches := discountRegex.FindStringSubmatch(content); len(matches) > 1 {
		if discount, err := strconv.ParseFloat(matches[1], 64); err == nil {
			invoice.Discount = discount
		}
	}

	// Extract total due
	totalRegex := regexp.MustCompile(`"total_due"\s*:\s*(\d+\.?\d*)`)
	if matches := totalRegex.FindStringSubmatch(content); len(matches) > 1 {
		if total, err := strconv.ParseFloat(matches[1], 64); err == nil {
			invoice.TotalDue = total
		}
	}

	// Extract line items using a more robust approach
	// Use a regex that can handle multiline content with the DOTALL flag
	itemsPattern := `"items"\s*:\s*\[\s*([\s\S]*?)\s*\]`
	itemsRegex := regexp.MustCompile(itemsPattern)

	if itemsMatch := itemsRegex.FindStringSubmatch(content); len(itemsMatch) > 1 {
		itemsContent := itemsMatch[1]

		// Find each item object using a pattern that can handle multiline content
		itemPattern := `\{\s*([\s\S]*?)\s*\}`
		itemRegex := regexp.MustCompile(itemPattern)
		itemMatches := itemRegex.FindAllStringSubmatch(itemsContent, -1)

		for _, itemMatch := range itemMatches {
			if len(itemMatch) > 0 {
				itemContent := itemMatch[0] // Get the full item object including braces

				// Create a new line item
				lineItem := domain.LineItem{}

				// Extract description
				descRegex := regexp.MustCompile(`"description"\s*:\s*"([^"]+)"`)
				if descMatches := descRegex.FindStringSubmatch(itemContent); len(descMatches) > 1 {
					lineItem.Description = descMatches[1]
				}

				// Extract quantity
				qtyRegex := regexp.MustCompile(`"quantity"\s*:\s*(\d+\.?\d*)`)
				if qtyMatches := qtyRegex.FindStringSubmatch(itemContent); len(qtyMatches) > 1 {
					if qty, err := strconv.ParseFloat(qtyMatches[1], 64); err == nil {
						lineItem.Quantity = qty
					}
				}

				// Extract unit price
				priceRegex := regexp.MustCompile(`"unit_price"\s*:\s*(\d+\.?\d*)`)
				if priceMatches := priceRegex.FindStringSubmatch(itemContent); len(priceMatches) > 1 {
					if price, err := strconv.ParseFloat(priceMatches[1], 64); err == nil {
						lineItem.UnitPrice = price
					}
				}

				// Extract total
				itemTotalRegex := regexp.MustCompile(`"total"\s*:\s*(\d+\.?\d*)`)
				if totalMatches := itemTotalRegex.FindStringSubmatch(itemContent); len(totalMatches) > 1 {
					if total, err := strconv.ParseFloat(totalMatches[1], 64); err == nil {
						lineItem.Total = total
					}
				}

				// Extract details array using a pattern that can handle multiline content
				detailsPattern := `"details"\s*:\s*\[\s*([\s\S]*?)\s*\]`
				detailsRegex := regexp.MustCompile(detailsPattern)

				if detailsMatch := detailsRegex.FindStringSubmatch(itemContent); len(detailsMatch) > 1 {
					detailsContent := detailsMatch[1]

					// Find each detail string
					detailRegex := regexp.MustCompile(`"([^"]+)"`)
					detailMatches := detailRegex.FindAllStringSubmatch(detailsContent, -1)

					for _, detailMatch := range detailMatches {
						if len(detailMatch) > 1 {
							lineItem.Details = append(lineItem.Details, detailMatch[1])
						}
					}
				}

				// Add the line item to the invoice if it has at least a description
				if lineItem.Description != "" {
					invoice.AddLineItem(lineItem)
				}
			}
		}
	}

	// If we couldn't extract anything, return an error
	if invoice.VendorName == "" && invoice.InvoiceNumber == "" && invoice.TotalDue == 0 && len(invoice.Items) == 0 {
		return nil, &OpenRouterError{
			Op:  "extract_json_with_regex",
			Err: fmt.Errorf("failed to extract invoice data from model response"),
		}
	}

	return invoice, nil
}
