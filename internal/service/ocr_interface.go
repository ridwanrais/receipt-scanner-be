package service

import (
	"context"

	"github.com/ridwanfathin/invoice-ocr-service/internal/model"
)

// OCRServiceInterface defines the interface for OCR services
type OCRServiceInterface interface {
	// ProcessInvoice processes an invoice image and returns structured data
	ProcessInvoice(ctx context.Context, request *model.OCRRequest) (*model.OCRResponse, error)
}
