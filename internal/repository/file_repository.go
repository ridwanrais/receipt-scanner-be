package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ridwanfathin/invoice-processor-service/internal/domain"
)

// RepositoryError represents an error that occurred within a repository
type RepositoryError struct {
	// Op is the operation that failed
	Op string

	// Err is the underlying error
	Err error
}

// Error returns a string representation of the error
func (e *RepositoryError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Op, e.Err)
	}
	return e.Op
}

// FileRepository implements InvoiceRepository using the local filesystem for storage
type FileRepository struct {
	baseDir string
	mutex   sync.RWMutex
}

// NewFileRepository creates a new file-based invoice repository
func NewFileRepository(baseDir string) (*FileRepository, error) {
	// Ensure base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, &RepositoryError{
			Op:  "create_repository",
			Err: fmt.Errorf("failed to create base directory: %w", err),
		}
	}

	// Create subdirectories
	for _, dir := range []string{"images", "invoices"} {
		subDir := filepath.Join(baseDir, dir)
		if err := os.MkdirAll(subDir, 0755); err != nil {
			return nil, &RepositoryError{
				Op:  "create_repository",
				Err: fmt.Errorf("failed to create %s directory: %w", dir, err),
			}
		}
	}

	return &FileRepository{
		baseDir: baseDir,
	}, nil
}

// StoreImage stores an image in the filesystem and returns its identifier
func (r *FileRepository) StoreImage(ctx context.Context, imageData []byte) (string, error) {
	select {
	case <-ctx.Done():
		return "", &RepositoryError{
			Op:  "store_image",
			Err: ctx.Err(),
		}
	default:
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique identifier based on timestamp
	imageID := fmt.Sprintf("%d", time.Now().UnixNano())

	// Write image to file
	filePath := filepath.Join(r.baseDir, "images", imageID)
	if err := os.WriteFile(filePath, imageData, 0644); err != nil {
		return "", &RepositoryError{
			Op:  "store_image",
			Err: fmt.Errorf("failed to write image file: %w", err),
		}
	}

	return imageID, nil
}

// GetInvoiceByID retrieves an invoice by its ID
func (r *FileRepository) GetInvoiceByID(ctx context.Context, invoiceID string) (*domain.Invoice, error) {
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

	// Construct file path
	filePath := filepath.Join(r.baseDir, "invoices", invoiceID+".json")

	// Read invoice file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, &RepositoryError{
			Op:  "get_invoice",
			Err: fmt.Errorf("failed to read invoice file: %w", err),
		}
	}

	// Deserialize JSON
	var invoice domain.Invoice
	if err := json.Unmarshal(data, &invoice); err != nil {
		return nil, &RepositoryError{
			Op:  "get_invoice",
			Err: fmt.Errorf("failed to deserialize invoice: %w", err),
		}
	}

	return &invoice, nil
}

// StoreInvoice stores invoice data in the repository
func (r *FileRepository) StoreInvoice(ctx context.Context, invoice *domain.Invoice) error {
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

	// Serialize invoice to JSON
	data, err := json.MarshalIndent(invoice, "", "  ")
	if err != nil {
		return &RepositoryError{
			Op:  "store_invoice",
			Err: fmt.Errorf("failed to serialize invoice: %w", err),
		}
	}

	// Write to file
	filePath := filepath.Join(r.baseDir, "invoices", invoice.InvoiceNumber+".json")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return &RepositoryError{
			Op:  "store_invoice",
			Err: fmt.Errorf("failed to write invoice file: %w", err),
		}
	}

	return nil
}

// ListInvoices retrieves a list of invoices with optional pagination
func (r *FileRepository) ListInvoices(ctx context.Context, offset, limit int) ([]*domain.Invoice, error) {
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

	// Get all invoice files
	invoicesDir := filepath.Join(r.baseDir, "invoices")
	files, err := os.ReadDir(invoicesDir)
	if err != nil {
		if os.IsNotExist(err) {
			// If directory doesn't exist yet, return empty list
			return []*domain.Invoice{}, nil
		}
		return nil, &RepositoryError{
			Op:  "list_invoices",
			Err: fmt.Errorf("failed to read invoices directory: %w", err),
		}
	}

	var invoices []*domain.Invoice

	// Apply offset and limit
	end := offset + limit
	if end > len(files) {
		end = len(files)
	}

	if offset >= len(files) {
		return invoices, nil
	}

	// Load invoices from files
	for i := offset; i < end; i++ {
		if files[i].IsDir() {
			continue
		}

		filePath := filepath.Join(invoicesDir, files[i].Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue // Skip files we can't read
		}

		var invoice domain.Invoice
		if err := json.Unmarshal(data, &invoice); err != nil {
			continue // Skip files we can't parse
		}

		invoices = append(invoices, &invoice)
	}

	return invoices, nil
}
