package handler

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ridwanfathin/invoice-processor-service/internal/model"
	"github.com/ridwanfathin/invoice-processor-service/internal/service"
)

// InvoiceHandler handles HTTP requests for invoice processing
type InvoiceHandler struct {
	processor   service.InvoiceProcessorServicer
	maxFileSize int64
}

// NewInvoiceHandler creates a new invoice processing handler
func NewInvoiceHandler(processor service.InvoiceProcessorServicer) *InvoiceHandler {
	return &InvoiceHandler{
		processor:   processor,
		maxFileSize: 10 * 1024 * 1024, // 10MB default
	}
}

// RegisterRoutes registers the handler's routes with the given router
func (h *InvoiceHandler) RegisterRoutes(router *gin.Engine) {
	router.POST("/api/v1/invoices/process", h.ProcessInvoice)
	router.POST("/api/v1/invoices/batch", h.ProcessInvoiceBatch)
}

// ProcessInvoice handles a request to process a single invoice image
func (h *InvoiceHandler) ProcessInvoice(c *gin.Context) {
	// Parse multipart form data
	if err := c.Request.ParseMultipartForm(h.maxFileSize); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Failed to parse form data: " + err.Error(),
		})
		return
	}

	// Get the file from the form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "No file uploaded or invalid file field",
		})
		return
	}
	defer file.Close()

	// Check file size
	if header.Size > h.maxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "File size exceeds limit",
		})
		return
	}

	// Read the file data
	fileData := make([]byte, header.Size)
	if _, err := file.Read(fileData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to read file data: " + err.Error(),
		})
		return
	}

	// Create request model
	request := &model.InvoiceProcessingRequest{
		File: fileData,
	}

	// Process the invoice
	log.Printf("Processing invoice: %s (%d bytes)", header.Filename, header.Size)
	response, err := h.processor.ProcessInvoice(c.Request.Context(), request)
	if err != nil {
		// Check if it's a configuration error
		if strings.Contains(err.Error(), "not configured") {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   fmt.Sprintf("Configuration error: %v", err.Error()),
			})
			return
		}

		// Otherwise, it's an internal server error
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Processing failed: %v", err.Error()),
		})
		return
	}

	// Check for application-level errors in the response
	if response.Error != "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   response.Error,
		})
		return
	}

	// Return successful response
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"invoice": response.Invoice,
	})
}

// ProcessInvoiceBatch handles a request to process multiple invoice images
func (h *InvoiceHandler) ProcessInvoiceBatch(c *gin.Context) {
	// This is a placeholder for batch processing functionality
	// In a real implementation, this would handle multiple files
	c.JSON(http.StatusNotImplemented, gin.H{
		"success": false,
		"error":   "Batch processing not yet implemented",
	})
}
