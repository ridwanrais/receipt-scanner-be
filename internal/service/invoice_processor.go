package service

import (
	"context"
	"fmt"

	"github.com/ridwanfathin/invoice-processor-service/internal/model"
	"github.com/ridwanfathin/invoice-processor-service/internal/repository"
)

// InvoiceProcessorServicer defines the interface for invoice processing services
type InvoiceProcessorServicer interface {
	// ProcessInvoice processes an invoice image and returns the extracted data
	ProcessInvoice(ctx context.Context, request *model.InvoiceProcessingRequest) (*model.InvoiceProcessingResponse, error)
	
	// SetRepository sets the repository for storing invoice data
	SetRepository(repo repository.InvoiceRepository)
	
	// Shutdown gracefully shuts down the service
	Shutdown()
}

// InvoiceProcessingError represents an error that occurred during invoice processing
type InvoiceProcessingError struct {
	// Op is the operation that failed
	Op string
	
	// Err is the underlying error
	Err error
}

// Error returns a string representation of the error
func (e *InvoiceProcessingError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Op, e.Err)
	}
	return e.Op
}

// Unwrap returns the underlying error
func (e *InvoiceProcessingError) Unwrap() error {
	return e.Err
}
