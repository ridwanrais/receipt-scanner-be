package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ridwanfathin/invoice-ocr-service/internal/model"
	"github.com/ridwanfathin/invoice-ocr-service/internal/service"
)

// OCRHandler handles HTTP requests for OCR processing
type OCRHandler struct {
	ocrService *service.OCRService
	maxFileSize int64
}

// NewOCRHandler creates a new OCR handler
func NewOCRHandler(ocrService *service.OCRService) *OCRHandler {
	return &OCRHandler{
		ocrService: ocrService,
		maxFileSize: 10 * 1024 * 1024, // 10MB default
	}
}

// SetMaxFileSize sets the maximum file size for uploads
func (h *OCRHandler) SetMaxFileSize(maxBytes int64) {
	h.maxFileSize = maxBytes
}

// RegisterRoutes registers the handler routes with the given router
func (h *OCRHandler) RegisterRoutes(router *gin.Engine) {
	v1 := router.Group("/api/v1")
	{
		v1.POST("/ocr/invoice", h.ProcessInvoice)
	}
}

// ProcessInvoice handles the invoice OCR request
func (h *OCRHandler) ProcessInvoice(c *gin.Context) {
	// Set request size limit
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxFileSize)

	// Parse multipart form
	if err := c.Request.ParseMultipartForm(h.maxFileSize); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("File too large or invalid multipart form: %v", err),
		})
		return
	}

	// Get file from form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Failed to get file from form: %v", err),
		})
		return
	}
	defer file.Close()

	// Check file size
	if header.Size > h.maxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("File size exceeds the limit of %d bytes", h.maxFileSize),
		})
		return
	}

	// Check file type
	fileType := header.Header.Get("Content-Type")
	if !isValidFileType(fileType) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Unsupported file type: %s. Only JPEG, PNG, and PDF are supported", fileType),
		})
		return
	}

	// Read file content
	fileData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to read file: %v", err),
		})
		return
	}

	// Create OCR request
	request := &model.OCRRequest{
		File: fileData,
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	// Process the invoice
	response, err := h.ocrService.ProcessInvoice(ctx, request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to process invoice: %v", err),
		})
		return
	}

	// Check for errors in the response
	if response.Error != "" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": response.Error,
		})
		return
	}

	// Return the result
	c.JSON(http.StatusOK, response.Invoice)
}

// isValidFileType checks if the file type is supported
func isValidFileType(fileType string) bool {
	fmt.Println("Debug: Checking file type:", fileType)
	validTypes := map[string]bool{
		"image/jpeg":      true,
		"image/jpg":       true,
		"image/pjpeg":     true, // Progressive JPEG
		"image/png":       true,
		"application/pdf": true,
	}
	return validTypes[fileType]
}
