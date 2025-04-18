package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/ridwanfathin/invoice-ocr-service/internal/model"
	"github.com/ridwanfathin/invoice-ocr-service/internal/ocr"
	"github.com/ridwanfathin/invoice-ocr-service/internal/util"
)

// OCRService handles the business logic for OCR processing
type OCRService struct {
	ocrEngine       *ocr.OCREngine
	imageProcessor  *util.ImageProcessor
	maxWorkers      int
	processingQueue chan *ocrTask
	wg              sync.WaitGroup
}

type ocrTask struct {
	ctx      context.Context
	request  *model.OCRRequest
	response chan *model.OCRResponse
}

// NewOCRService creates a new OCR service
func NewOCRService(ocrEngine *ocr.OCREngine, imageProcessor *util.ImageProcessor, maxWorkers int) *OCRService {
	if maxWorkers <= 0 {
		maxWorkers = 5 // Default to 5 workers
	}

	service := &OCRService{
		ocrEngine:       ocrEngine,
		imageProcessor:  imageProcessor,
		maxWorkers:      maxWorkers,
		processingQueue: make(chan *ocrTask, maxWorkers*2), // Buffer size is twice the number of workers
	}

	// Start worker pool
	service.startWorkers()

	return service
}

// startWorkers initializes the worker pool
func (s *OCRService) startWorkers() {
	for i := 0; i < s.maxWorkers; i++ {
		s.wg.Add(1)
		go func(workerID int) {
			defer s.wg.Done()
			s.worker(workerID)
		}(i)
	}
}

// worker processes tasks from the queue
func (s *OCRService) worker(id int) {
	for task := range s.processingQueue {
		// Check if context is canceled
		if task.ctx.Err() != nil {
			task.response <- &model.OCRResponse{
				Error: "request canceled",
			}
			continue
		}

		// Process the OCR task
		invoice, err := s.processOCR(task.request.File)
		if err != nil {
			task.response <- &model.OCRResponse{
				Error: err.Error(),
			}
		} else {
			task.response <- &model.OCRResponse{
				Invoice: invoice,
			}
		}
	}
}

// ProcessInvoice processes an invoice image and returns structured data
func (s *OCRService) ProcessInvoice(ctx context.Context, request *model.OCRRequest) (*model.OCRResponse, error) {
	if len(request.File) == 0 {
		return nil, errors.New("empty file data")
	}

	// Create response channel
	responseChan := make(chan *model.OCRResponse, 1)

	// Create task
	task := &ocrTask{
		ctx:      ctx,
		request:  request,
		response: responseChan,
	}

	// Send task to processing queue
	select {
	case s.processingQueue <- task:
		// Task accepted
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Wait for response or context cancellation
	select {
	case response := <-responseChan:
		return response, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Shutdown gracefully shuts down the service
func (s *OCRService) Shutdown() {
	close(s.processingQueue)
	s.wg.Wait()
}

// processOCR handles the OCR processing for an invoice image
func (s *OCRService) processOCR(imageData []byte) (*model.Invoice, error) {
	// Preprocess the image
	processedImage, err := s.imageProcessor.PreprocessImage(imageData)
	if err != nil {
		return nil, fmt.Errorf("image preprocessing failed: %w", err)
	}

	// Extract text using OCR
	text, err := s.ocrEngine.ExtractText(processedImage)
	if err != nil {
		return nil, fmt.Errorf("OCR extraction failed: %w", err)
	}

	// Parse the extracted text into structured invoice data
	invoice, err := s.parseInvoiceText(text)
	if err != nil {
		return nil, fmt.Errorf("invoice parsing failed: %w", err)
	}

	return invoice, nil
}

// parseInvoiceText parses the OCR text into structured invoice data
func (s *OCRService) parseInvoiceText(text string) (*model.Invoice, error) {
	if text == "" {
		return nil, errors.New("empty OCR text")
	}

	invoice := &model.Invoice{
		Items: []model.LineItem{},
	}

	// Extract vendor name (usually at the top of the invoice)
	invoice.Vendor = extractVendor(text)

	// Extract invoice number
	invoice.InvoiceNumber = extractInvoiceNumber(text)

	// Extract invoice date
	invoice.InvoiceDate = extractInvoiceDate(text)

	// Extract line items
	invoice.Items = extractLineItems(text)

	// Extract totals
	invoice.Subtotal = extractSubtotal(text)
	invoice.Tax = extractTax(text)
	invoice.Total = extractTotal(text)

	// If total is not found, calculate it from subtotal and tax
	if invoice.Total == 0 && invoice.Subtotal > 0 {
		invoice.Total = invoice.Subtotal + invoice.Tax
	}

	return invoice, nil
}

// extractVendor extracts the vendor name from the OCR text
func extractVendor(text string) string {
	lines := strings.Split(text, "\n")
	if len(lines) > 0 {
		// Assume the vendor name is in the first few lines
		// This is a simplified approach and might need refinement
		for i := 0; i < min(3, len(lines)); i++ {
			line := strings.TrimSpace(lines[i])
			if line != "" && !strings.Contains(strings.ToLower(line), "invoice") {
				return line
			}
		}
	}
	return ""
}

// extractInvoiceNumber extracts the invoice number from the OCR text
func extractInvoiceNumber(text string) string {
	// Look for patterns like "Invoice #: 12345" or "Invoice Number: 12345"
	patterns := []string{
		`(?i)invoice\s*#?\s*:?\s*([A-Za-z0-9-]+)`,
		`(?i)invoice\s*number\s*:?\s*([A-Za-z0-9-]+)`,
		`(?i)inv\s*#?\s*:?\s*([A-Za-z0-9-]+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(text)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}

	return ""
}

// extractInvoiceDate extracts the invoice date from the OCR text
func extractInvoiceDate(text string) string {
	// Look for date patterns
	patterns := []string{
		// MM/DD/YYYY
		`(?i)(?:invoice|date).*?(\d{1,2}[/-]\d{1,2}[/-]\d{2,4})`,
		// YYYY-MM-DD
		`(?i)(?:invoice|date).*?(\d{4}[/-]\d{1,2}[/-]\d{1,2})`,
		// Month DD, YYYY
		`(?i)(?:invoice|date).*?([A-Za-z]+\s+\d{1,2},?\s+\d{4})`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(text)
		if len(matches) > 1 {
			// Try to convert to YYYY-MM-DD format
			dateStr := strings.TrimSpace(matches[1])
			return formatDate(dateStr)
		}
	}

	return ""
}

// formatDate attempts to convert various date formats to YYYY-MM-DD
func formatDate(dateStr string) string {
	// This is a simplified implementation
	// In a production environment, you would use a more robust date parsing library

	// Try to parse MM/DD/YYYY
	if re := regexp.MustCompile(`(\d{1,2})[/-](\d{1,2})[/-](\d{2,4})`); re.MatchString(dateStr) {
		matches := re.FindStringSubmatch(dateStr)
		if len(matches) == 4 {
			month, _ := strconv.Atoi(matches[1])
			day, _ := strconv.Atoi(matches[2])
			year, _ := strconv.Atoi(matches[3])
			
			// Handle 2-digit years
			if year < 100 {
				if year < 50 {
					year += 2000
				} else {
					year += 1900
				}
			}
			
			return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		}
	}

	// Try to parse YYYY-MM-DD
	if re := regexp.MustCompile(`(\d{4})[/-](\d{1,2})[/-](\d{1,2})`); re.MatchString(dateStr) {
		matches := re.FindStringSubmatch(dateStr)
		if len(matches) == 4 {
			year, _ := strconv.Atoi(matches[1])
			month, _ := strconv.Atoi(matches[2])
			day, _ := strconv.Atoi(matches[3])
			return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		}
	}

	// For other formats, return as is
	return dateStr
}

// extractLineItems extracts line items from the OCR text
func extractLineItems(text string) []model.LineItem {
	var items []model.LineItem

	// Look for patterns that might indicate line items
	// This is a simplified approach and might need refinement
	lines := strings.Split(text, "\n")
	
	inItemsSection := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if we're entering the items section
		if strings.Contains(strings.ToLower(line), "item") && 
		   (strings.Contains(strings.ToLower(line), "description") || 
		    strings.Contains(strings.ToLower(line), "qty") || 
		    strings.Contains(strings.ToLower(line), "price")) {
			inItemsSection = true
			continue
		}

		// Check if we're leaving the items section
		if inItemsSection && (strings.Contains(strings.ToLower(line), "subtotal") || 
		                     strings.Contains(strings.ToLower(line), "total")) {
			inItemsSection = false
			continue
		}

		// Process line items
		if inItemsSection {
			// Try to extract quantity and price
			// This pattern looks for a description followed by numbers (quantity and price)
			re := regexp.MustCompile(`(.*?)\s+(\d+)\s+(\d+(?:\.\d+)?)`)
			matches := re.FindStringSubmatch(line)
			
			if len(matches) >= 4 {
				description := strings.TrimSpace(matches[1])
				quantity, _ := strconv.Atoi(matches[2])
				unitPrice, _ := strconv.ParseFloat(matches[3], 64)
				
				items = append(items, model.LineItem{
					Description: description,
					Quantity:    quantity,
					UnitPrice:   unitPrice,
				})
			} else {
				// Try another pattern
				re = regexp.MustCompile(`(.*?)\s+(\d+(?:\.\d+)?)`)
				matches = re.FindStringSubmatch(line)
				
				if len(matches) >= 3 {
					description := strings.TrimSpace(matches[1])
					price, _ := strconv.ParseFloat(matches[2], 64)
					
					items = append(items, model.LineItem{
						Description: description,
						Quantity:    1, // Default to 1 if quantity is not specified
						UnitPrice:   price,
					})
				}
			}
		}
	}

	return items
}

// extractSubtotal extracts the subtotal from the OCR text
func extractSubtotal(text string) float64 {
	return extractAmount(text, []string{
		`(?i)subtotal\s*:?\s*\$?\s*(\d+(?:,\d+)*(?:\.\d+)?)`,
		`(?i)sub\s*total\s*:?\s*\$?\s*(\d+(?:,\d+)*(?:\.\d+)?)`,
		`(?i)sub-total\s*:?\s*\$?\s*(\d+(?:,\d+)*(?:\.\d+)?)`,
	})
}

// extractTax extracts the tax amount from the OCR text
func extractTax(text string) float64 {
	return extractAmount(text, []string{
		`(?i)tax\s*:?\s*\$?\s*(\d+(?:,\d+)*(?:\.\d+)?)`,
		`(?i)vat\s*:?\s*\$?\s*(\d+(?:,\d+)*(?:\.\d+)?)`,
		`(?i)gst\s*:?\s*\$?\s*(\d+(?:,\d+)*(?:\.\d+)?)`,
	})
}

// extractTotal extracts the total amount from the OCR text
func extractTotal(text string) float64 {
	return extractAmount(text, []string{
		`(?i)total\s*:?\s*\$?\s*(\d+(?:,\d+)*(?:\.\d+)?)`,
		`(?i)amount\s*due\s*:?\s*\$?\s*(\d+(?:,\d+)*(?:\.\d+)?)`,
		`(?i)grand\s*total\s*:?\s*\$?\s*(\d+(?:,\d+)*(?:\.\d+)?)`,
	})
}

// extractAmount extracts an amount using the given patterns
func extractAmount(text string, patterns []string) float64 {
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(text)
		if len(matches) > 1 {
			// Remove commas
			amountStr := strings.ReplaceAll(matches[1], ",", "")
			amount, err := strconv.ParseFloat(amountStr, 64)
			if err == nil {
				return amount
			}
		}
	}
	return 0
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
