package repository

import (
	"context"

	"github.com/ridwanfathin/invoice-processor-service/internal/domain"
)

// InvoiceRepository defines the interface for invoice data storage operations
type InvoiceRepository interface {
	// StoreImage stores the raw invoice image and returns an identifier
	StoreImage(ctx context.Context, imageData []byte) (string, error)

	// StoreInvoice stores extracted invoice data and returns an identifier
	StoreInvoice(ctx context.Context, invoice *domain.Invoice) error

	// GetInvoiceByID retrieves an invoice by its ID
	GetInvoiceByID(ctx context.Context, invoiceID string) (*domain.Invoice, error)

	// ListInvoices retrieves a list of invoices with optional pagination
	ListInvoices(ctx context.Context, offset, limit int) ([]*domain.Invoice, error)
}
