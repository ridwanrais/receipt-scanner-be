package repository

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ridwanfathin/invoice-processor-service/internal/domain"
	"github.com/ridwanfathin/invoice-processor-service/internal/openrouter"
)

// SupabaseRepository implements both InvoiceRepository and ReceiptRepository interfaces
// for backward compatibility during the transition
type SupabaseRepository struct {
	openRouterClient *openrouter.Client
	mutex            sync.RWMutex
	invoiceCache     map[string]*domain.Invoice // Invoice cache for old API
	receiptCache     map[string]*domain.Receipt // Receipt cache for new API
}

// NewSupabaseRepository creates a new Supabase-based repository
func NewSupabaseRepository(openRouterClient *openrouter.Client) *SupabaseRepository {
	return &SupabaseRepository{
		openRouterClient: openRouterClient,
		invoiceCache:     make(map[string]*domain.Invoice),
		receiptCache:     make(map[string]*domain.Receipt),
	}
}

//
// InvoiceRepository interface implementation (for backward compatibility)
//

// StoreImage stores an image using the OpenRouter client's Supabase integration
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
	return "supabase-stored", nil
}

// StoreInvoice stores invoice data in memory
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

	// Store in memory cache
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

//
// ReceiptRepository interface implementation (for new API)
//

// CreateReceipt saves a new receipt to memory storage
func (r *SupabaseRepository) CreateReceipt(ctx context.Context, receipt *domain.Receipt) (*domain.Receipt, error) {
	select {
	case <-ctx.Done():
		return nil, &RepositoryError{
			Op:  "create_receipt",
			Err: ctx.Err(),
		}
	default:
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Validate receipt
	if receipt.ID == "" {
		// Generate an ID if not provided
		receipt.ID = fmt.Sprintf("receipt-%d", time.Now().UnixNano())
	}
	
	// Store in memory cache
	r.receiptCache[receipt.ID] = receipt

	return receipt, nil
}

// GetReceiptByID retrieves a receipt by its ID
func (r *SupabaseRepository) GetReceiptByID(ctx context.Context, receiptID string) (*domain.Receipt, error) {
	select {
	case <-ctx.Done():
		return nil, &RepositoryError{
			Op:  "get_receipt",
			Err: ctx.Err(),
		}
	default:
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	receipt, ok := r.receiptCache[receiptID]
	if !ok {
		return nil, &RepositoryError{
			Op:  "get_receipt",
			Err: fmt.Errorf("receipt not found: %s", receiptID),
		}
	}

	return receipt, nil
}

// UpdateReceipt updates an existing receipt
func (r *SupabaseRepository) UpdateReceipt(ctx context.Context, receipt *domain.Receipt) (*domain.Receipt, error) {
	select {
	case <-ctx.Done():
		return nil, &RepositoryError{
			Op:  "update_receipt",
			Err: ctx.Err(),
		}
	default:
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if receipt exists
	if _, ok := r.receiptCache[receipt.ID]; !ok {
		return nil, &RepositoryError{
			Op:  "update_receipt",
			Err: fmt.Errorf("receipt not found: %s", receipt.ID),
		}
	}

	// Update receipt
	receipt.UpdatedAt = time.Now()
	r.receiptCache[receipt.ID] = receipt

	return receipt, nil
}

// DeleteReceipt deletes a receipt by its ID
func (r *SupabaseRepository) DeleteReceipt(ctx context.Context, receiptID string) error {
	select {
	case <-ctx.Done():
		return &RepositoryError{
			Op:  "delete_receipt",
			Err: ctx.Err(),
		}
	default:
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if receipt exists
	if _, ok := r.receiptCache[receiptID]; !ok {
		return &RepositoryError{
			Op:  "delete_receipt",
			Err: fmt.Errorf("receipt not found: %s", receiptID),
		}
	}

	// Delete receipt
	delete(r.receiptCache, receiptID)

	return nil
}

// ListReceipts retrieves receipts with optional filters and pagination
func (r *SupabaseRepository) ListReceipts(ctx context.Context, filter domain.ReceiptFilter) (*domain.PaginatedReceipts, error) {
	select {
	case <-ctx.Done():
		return nil, &RepositoryError{
			Op:  "list_receipts",
			Err: ctx.Err(),
		}
	default:
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Set default pagination values if not provided
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 10
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	var receipts []domain.Receipt
	var filteredReceipts []domain.Receipt

	// Convert map to slice for filtering
	for _, receipt := range r.receiptCache {
		receipts = append(receipts, *receipt)
	}

	// Apply filters
	for _, receipt := range receipts {
		// Apply date filter
		if filter.StartDate != nil && receipt.Date.Before(*filter.StartDate) {
			continue
		}
		if filter.EndDate != nil && receipt.Date.After(*filter.EndDate) {
			continue
		}

		// Apply merchant filter
		if filter.Merchant != "" && receipt.Merchant != filter.Merchant {
			continue
		}

		filteredReceipts = append(filteredReceipts, receipt)
	}

	// Calculate pagination
	totalItems := len(filteredReceipts)
	totalPages := (totalItems + filter.Limit - 1) / filter.Limit // Ceiling division
	
	// Determine current page items
	startIdx := (filter.Page - 1) * filter.Limit
	endIdx := startIdx + filter.Limit
	if endIdx > totalItems {
		endIdx = totalItems
	}
	if startIdx >= totalItems {
		startIdx = 0
		endIdx = 0
	}

	pageItems := []domain.Receipt{}
	if startIdx < endIdx {
		pageItems = filteredReceipts[startIdx:endIdx]
	}

	return &domain.PaginatedReceipts{
		Data: pageItems,
		Pagination: domain.Pagination{
			TotalItems:  totalItems,
			TotalPages:  totalPages,
			CurrentPage: filter.Page,
			Limit:       filter.Limit,
		},
	}, nil
}

// GetReceiptItems retrieves all items from a specific receipt
func (r *SupabaseRepository) GetReceiptItems(ctx context.Context, receiptID string) ([]domain.ReceiptItem, error) {
	select {
	case <-ctx.Done():
		return nil, &RepositoryError{
			Op:  "get_receipt_items",
			Err: ctx.Err(),
		}
	default:
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check if receipt exists
	receipt, ok := r.receiptCache[receiptID]
	if !ok {
		return nil, &RepositoryError{
			Op:  "get_receipt_items",
			Err: fmt.Errorf("receipt not found: %s", receiptID),
		}
	}

	return receipt.Items, nil
}

// The in-memory implementation doesn't support analytics
// These methods return empty results that match the expected format

// GetDashboardSummary returns a minimal dashboard summary
func (r *SupabaseRepository) GetDashboardSummary(ctx context.Context, startDate, endDate *string) (*domain.DashboardSummary, error) {
	return &domain.DashboardSummary{
		TopCategories: []domain.CategorySummary{},
		TopMerchants:  []domain.MerchantSummary{},
	}, nil
}

// GetSpendingTrends returns minimal spending trends
func (r *SupabaseRepository) GetSpendingTrends(ctx context.Context, period string, startDate, endDate *string) (*domain.SpendingTrends, error) {
	return &domain.SpendingTrends{
		Period: period,
		Data:   []domain.SpendingTrendDataItem{},
	}, nil
}

// GetSpendingByCategory returns minimal category spending
func (r *SupabaseRepository) GetSpendingByCategory(ctx context.Context, startDate, endDate *string) (*domain.CategorySpending, error) {
	return &domain.CategorySpending{
		Categories: []domain.CategorySpendingItem{},
	}, nil
}

// GetMerchantFrequency returns minimal merchant frequency
func (r *SupabaseRepository) GetMerchantFrequency(ctx context.Context, startDate, endDate *string, limit int) (*domain.MerchantFrequency, error) {
	return &domain.MerchantFrequency{
		Merchants: []domain.MerchantFrequencyDetail{},
	}, nil
}

// GetMonthlyComparison returns minimal monthly comparison
func (r *SupabaseRepository) GetMonthlyComparison(ctx context.Context, month1, month2 string) (*domain.MonthlyComparison, error) {
	return &domain.MonthlyComparison{
		Month1:     month1,
		Month2:     month2,
		Categories: []domain.MonthlyCategoryComparison{},
	}, nil
}
