package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/ridwanfathin/invoice-processor-service/internal/domain"
	"github.com/ridwanfathin/invoice-processor-service/internal/openrouter"
)

// SupabaseRepository implements InvoiceRepository using Supabase for storage
// instead of local file storage
type SupabaseRepository struct {
	openRouterClient *openrouter.Client
	mutex            sync.RWMutex
	invoiceCache     map[string]*domain.Invoice // In-memory cache for development
}

// NewSupabaseRepository creates a new Supabase-based invoice repository
func NewSupabaseRepository(openRouterClient *openrouter.Client) *SupabaseRepository {
	return &SupabaseRepository{
		openRouterClient: openRouterClient,
		invoiceCache:     make(map[string]*domain.Invoice),
	}
}

// StoreImage stores an image using the OpenRouter client's Supabase integration
// This eliminates redundant local file storage
func (r *SupabaseRepository) StoreImage(ctx context.Context, imageData []byte) (string, error) {
	select {
	case <-ctx.Done():
		return "", &RepositoryError{
			Op:  "store_image",
			Err: ctx.Err(),
		}
	default:
	}

	// We don't need to save locally since OpenRouter client already uploads to Supabase
	// We'll just return a placeholder ID that can be used for reference
	// The actual Supabase URL will be generated during the ExtractInvoiceData call
	
	// This is essentially a no-op since the OpenRouter client handles the Supabase upload
	return "supabase-stored", nil
}

// StoreInvoice stores invoice data in memory (temporary implementation)
// This will be replaced with actual database implementation later
func (r *SupabaseRepository) StoreInvoice(ctx context.Context, invoice *domain.Invoice) error {
	select {
	case <-ctx.Done():
		return &RepositoryError{
			Op:  "store_invoice",
			Err: ctx.Err(),
		}
	default:
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Ensure we have an invoice number
	if invoice.InvoiceNumber == "" {
		return &RepositoryError{
			Op:  "store_invoice",
			Err: fmt.Errorf("invoice number is required"),
		}
	}

	// Store in memory cache for now (will be replaced with database)
	r.invoiceCache[invoice.InvoiceNumber] = invoice

	return nil
}

// GetInvoiceByID retrieves an invoice by its ID from memory cache
func (r *SupabaseRepository) GetInvoiceByID(ctx context.Context, invoiceID string) (*domain.Invoice, error) {
	select {
	case <-ctx.Done():
		return nil, &RepositoryError{
			Op:  "get_invoice",
			Err: ctx.Err(),
		}
	default:
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	invoice, ok := r.invoiceCache[invoiceID]
	if !ok {
		return nil, &RepositoryError{
			Op:  "get_invoice",
			Err: fmt.Errorf("invoice not found: %s", invoiceID),
		}
	}

	return invoice, nil
}

// ListInvoices retrieves a list of invoices with optional pagination from memory cache
func (r *SupabaseRepository) ListInvoices(ctx context.Context, offset, limit int) ([]*domain.Invoice, error) {
	select {
	case <-ctx.Done():
		return nil, &RepositoryError{
			Op:  "list_invoices",
			Err: ctx.Err(),
		}
	default:
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var invoices []*domain.Invoice
	var index int = 0

	// Collect all invoices from the cache
	for _, invoice := range r.invoiceCache {
		if index >= offset && (limit <= 0 || len(invoices) < limit) {
			invoices = append(invoices, invoice)
		}
		index++
	}

	return invoices, nil
}
