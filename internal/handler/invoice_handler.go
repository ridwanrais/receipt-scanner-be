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
	router.POST("/v1/invoices/process", h.ProcessInvoice)
	router.POST("/v1/invoices/batch", h.ProcessInvoiceBatch)
}

// ProcessInvoice handles a request to process a single invoice image
// @Summary Process an invoice
// @Description Upload and process an invoice image to extract data using AI
// @Tags invoices
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Invoice image file"
// @Success 200 {object} model.InvoiceSuccessResponse "Successfully processed invoice"
// @Failure 400 {object} model.InvoiceErrorResponse "Bad request or configuration error"
// @Failure 500 {object} model.InvoiceErrorResponse "Internal server error"
// @Router /v1/invoices/process [post]
func (h *InvoiceHandler) ProcessInvoice(c *gin.Context) {
	// Parse multipart form data
	if err := c.Request.ParseMultipartForm(h.maxFileSize); err != nil {
		respondBadRequest(c, "Failed to parse form data: "+err.Error())
		return
	}

	// Get the file from the form
	file, header, err := getFormFile(c, "file")
	if err != nil {
		respondBadRequest(c, err.Error())
		return
	}
	defer file.Close()

	// Check file size
	if header.Size > h.maxFileSize {
		respondBadRequest(c, "File size exceeds limit")
		return
	}

	// Read the file data
	fileData := make([]byte, header.Size)
	if _, err := file.Read(fileData); err != nil {
		respondInternalServerError(c, "Failed to read file data: "+err.Error())
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
			respondBadRequest(c, fmt.Sprintf("Configuration error: %v", err.Error()))
		} else {
			respondInternalServerError(c, fmt.Sprintf("Processing failed: %v", err.Error()))
		}
		return
	}

	// Check for application-level errors in the response
	if response.Error != "" {
		respondOK(c, model.InvoiceErrorResponse{
			Success: false,
			Error:   response.Error,
		})
		return
	}

	// Return successful response
	respondOK(c, model.InvoiceSuccessResponse{
		Success: true,
		Invoice: response.Invoice,
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
