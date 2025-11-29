package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ridwanfathin/invoice-processor-service/internal/domain"
	"github.com/ridwanfathin/invoice-processor-service/internal/mlxclient"
	"github.com/ridwanfathin/invoice-processor-service/internal/openrouter"
	"github.com/ridwanfathin/invoice-processor-service/internal/repository"
	"github.com/ridwanfathin/invoice-processor-service/internal/storage"
)

// ReceiptServiceError represents an error in the receipt service
type ReceiptServiceError struct {
	Op  string
	Err error
}

func (e *ReceiptServiceError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Op, e.Err)
	}
	return e.Op
}

// ReceiptService defines the interface for receipt-related business logic
type ReceiptService interface {
	// CRUD operations
	ScanReceipt(ctx context.Context, imageData []byte, userID string) (*domain.Receipt, error)
	CreateReceipt(ctx context.Context, receipt *domain.Receipt) (*domain.Receipt, error)
	GetReceiptByID(ctx context.Context, receiptID string) (*domain.Receipt, error)
	UpdateReceipt(ctx context.Context, receipt *domain.Receipt) (*domain.Receipt, error)
	DeleteReceipt(ctx context.Context, receiptID string) error

	// Query operations
	ListReceipts(ctx context.Context, filter domain.ReceiptFilter) (*domain.PaginatedReceipts, error)
	GetReceiptItems(ctx context.Context, receiptID string) ([]domain.ReceiptItem, error)

	// Dashboard and insights operations
	GetDashboardSummary(ctx context.Context, startDate, endDate *string) (*domain.DashboardSummary, error)
	GetSpendingTrends(ctx context.Context, period string, startDate, endDate *string) (*domain.SpendingTrends, error)
	GetSpendingByCategory(ctx context.Context, startDate, endDate *string) (*domain.CategorySpending, error)
	GetMerchantFrequency(ctx context.Context, startDate, endDate *string, limit int) (*domain.MerchantFrequency, error)
	GetMonthlyComparison(ctx context.Context, month1, month2 string) (*domain.MonthlyComparison, error)
}

// ReceiptServiceImpl implements the ReceiptService interface
type ReceiptServiceImpl struct {
	repository    repository.ReceiptRepository
	openAIClient  *openrouter.Client
	mlxClient     *mlxclient.Client
	s3Uploader    *storage.S3Uploader
	useMLXService bool
	workerPool    chan struct{}
}

// NewReceiptService creates a new ReceiptService
func NewReceiptService(repo repository.ReceiptRepository, openAIClient *openrouter.Client, mlxClient *mlxclient.Client, s3Uploader *storage.S3Uploader, useMLXService bool, maxWorkers int) ReceiptService {
	return &ReceiptServiceImpl{
		repository:    repo,
		openAIClient:  openAIClient,
		mlxClient:     mlxClient,
		s3Uploader:    s3Uploader,
		useMLXService: useMLXService,
		workerPool:    make(chan struct{}, maxWorkers),
	}
}

// ScanReceipt processes an image to extract receipt data
func (s *ReceiptServiceImpl) ScanReceipt(ctx context.Context, imageData []byte, userID string) (*domain.Receipt, error) {
	// Acquire worker from pool
	select {
	case s.workerPool <- struct{}{}:
		// Worker acquired, continue processing
		defer func() {
			// Release worker back to pool
			<-s.workerPool
		}()
	case <-ctx.Done():
		// Context cancelled while waiting for worker
		return nil, &ReceiptServiceError{
			Op:  "acquire_worker",
			Err: ctx.Err(),
		}
	}

	// Extract invoice data using MLX or OpenRouter
	var invoiceData *domain.Invoice
	var err error

	if s.useMLXService && s.mlxClient != nil && s.s3Uploader != nil {
		// Upload image to S3 first
		timestamp := time.Now().UnixNano()
		filename := fmt.Sprintf("invoice_%d.png", timestamp)
		imageURL, uploadErr := s.s3Uploader.UploadImage(imageData, filename)
		if uploadErr != nil {
			return nil, &ReceiptServiceError{
				Op:  "upload_image_to_s3",
				Err: uploadErr,
			}
		}

		// Use MLX service with the S3 URL
		invoiceData, err = s.mlxClient.ExtractInvoiceData(imageURL)
		if err != nil {
			return nil, &ReceiptServiceError{
				Op:  "extract_receipt_data_mlx",
				Err: err,
			}
		}
	} else {
		// Use OpenRouter to extract invoice data
		invoiceData, err = s.openAIClient.ExtractInvoiceData(imageData)
		if err != nil {
			return nil, &ReceiptServiceError{
				Op:  "extract_receipt_data_openrouter",
				Err: err,
			}
		}
	}

	// Convert domain.Invoice to domain.Receipt
	receipt := &domain.Receipt{
		UserID:    userID,
		Merchant:  invoiceData.VendorName,
		Date:      invoiceData.InvoiceDate.Time,
		Total:     invoiceData.TotalDue,
		Tax:       invoiceData.TaxAmount,
		Subtotal:  invoiceData.Subtotal,
		Items:     make([]domain.ReceiptItem, 0, len(invoiceData.Items)),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Convert invoice items to receipt items
	for _, item := range invoiceData.Items {
		category := inferCategory(item.Description)
		if item.Category != "" {
			category = item.Category // prefer LLM if present
		}
		receiptItem := domain.ReceiptItem{
			Name:     item.Description,
			Quantity: int(item.Quantity), // Convert float64 to int
			Price:    item.UnitPrice,
			Currency: item.Currency,
			Category: category,
		}
		receipt.Items = append(receipt.Items, receiptItem)
	}

	// Save receipt to database
	storedReceipt, err := s.repository.CreateReceipt(ctx, receipt)
	if err != nil {
		return nil, &ReceiptServiceError{
			Op:  "store_receipt",
			Err: err,
		}
	}

	return storedReceipt, nil
}

// inferCategory maps item descriptions to categories using keywords
func inferCategory(description string) string {
	desc := strings.ToLower(description)
	switch {
	case strings.Contains(desc, "taxi") || strings.Contains(desc, "uber") || strings.Contains(desc, "grab"):
		return "Transport"
	case strings.Contains(desc, "flight") || strings.Contains(desc, "airfare"):
		return "Travel"
	case strings.Contains(desc, "hotel") || strings.Contains(desc, "inn"):
		return "Accommodation"
	case strings.Contains(desc, "meal") || strings.Contains(desc, "food") || strings.Contains(desc, "restaurant"):
		return "Food"
	case strings.Contains(desc, "office") || strings.Contains(desc, "stationery"):
		return "Office Supplies"
	case strings.Contains(desc, "consult") || strings.Contains(desc, "service"):
		return "Professional Services"
	default:
		return "Other"
	}
}

// CreateReceipt saves a new receipt
func (s *ReceiptServiceImpl) CreateReceipt(ctx context.Context, receipt *domain.Receipt) (*domain.Receipt, error) {
	// Set timestamps
	now := time.Now()
	receipt.CreatedAt = now
	receipt.UpdatedAt = now

	// Save to repository
	storedReceipt, err := s.repository.CreateReceipt(ctx, receipt)
	if err != nil {
		return nil, &ReceiptServiceError{
			Op:  "create_receipt",
			Err: err,
		}
	}

	return storedReceipt, nil
}

// GetReceiptByID retrieves a receipt by ID
func (s *ReceiptServiceImpl) GetReceiptByID(ctx context.Context, receiptID string) (*domain.Receipt, error) {
	receipt, err := s.repository.GetReceiptByID(ctx, receiptID)
	if err != nil {
		return nil, &ReceiptServiceError{
			Op:  "get_receipt",
			Err: err,
		}
	}
	return receipt, nil
}

// UpdateReceipt updates an existing receipt
func (s *ReceiptServiceImpl) UpdateReceipt(ctx context.Context, receipt *domain.Receipt) (*domain.Receipt, error) {
	// Update timestamp
	receipt.UpdatedAt = time.Now()

	// Update in repository
	updatedReceipt, err := s.repository.UpdateReceipt(ctx, receipt)
	if err != nil {
		return nil, &ReceiptServiceError{
			Op:  "update_receipt",
			Err: err,
		}
	}

	return updatedReceipt, nil
}

// DeleteReceipt deletes a receipt
func (s *ReceiptServiceImpl) DeleteReceipt(ctx context.Context, receiptID string) error {
	err := s.repository.DeleteReceipt(ctx, receiptID)
	if err != nil {
		return &ReceiptServiceError{
			Op:  "delete_receipt",
			Err: err,
		}
	}
	return nil
}

// ListReceipts retrieves a paginated list of receipts
func (s *ReceiptServiceImpl) ListReceipts(ctx context.Context, filter domain.ReceiptFilter) (*domain.PaginatedReceipts, error) {
	receipts, err := s.repository.ListReceipts(ctx, filter)
	if err != nil {
		return nil, &ReceiptServiceError{
			Op:  "list_receipts",
			Err: err,
		}
	}
	return receipts, nil
}

// GetReceiptItems retrieves items for a specific receipt
func (s *ReceiptServiceImpl) GetReceiptItems(ctx context.Context, receiptID string) ([]domain.ReceiptItem, error) {
	items, err := s.repository.GetReceiptItems(ctx, receiptID)
	if err != nil {
		return nil, &ReceiptServiceError{
			Op:  "get_receipt_items",
			Err: err,
		}
	}
	return items, nil
}

// GetDashboardSummary retrieves summary data for the dashboard
func (s *ReceiptServiceImpl) GetDashboardSummary(ctx context.Context, startDate, endDate *string) (*domain.DashboardSummary, error) {
	summary, err := s.repository.GetDashboardSummary(ctx, startDate, endDate)
	if err != nil {
		return nil, &ReceiptServiceError{
			Op:  "get_dashboard_summary",
			Err: err,
		}
	}
	return summary, nil
}

// GetSpendingTrends retrieves spending trends over time
func (s *ReceiptServiceImpl) GetSpendingTrends(ctx context.Context, period string, startDate, endDate *string) (*domain.SpendingTrends, error) {
	trends, err := s.repository.GetSpendingTrends(ctx, period, startDate, endDate)
	if err != nil {
		return nil, &ReceiptServiceError{
			Op:  "get_spending_trends",
			Err: err,
		}
	}
	return trends, nil
}

// GetSpendingByCategory retrieves spending breakdown by category
func (s *ReceiptServiceImpl) GetSpendingByCategory(ctx context.Context, startDate, endDate *string) (*domain.CategorySpending, error) {
	categorySpending, err := s.repository.GetSpendingByCategory(ctx, startDate, endDate)
	if err != nil {
		return nil, &ReceiptServiceError{
			Op:  "get_spending_by_category",
			Err: err,
		}
	}
	return categorySpending, nil
}

// GetMerchantFrequency retrieves data on frequently visited merchants
func (s *ReceiptServiceImpl) GetMerchantFrequency(ctx context.Context, startDate, endDate *string, limit int) (*domain.MerchantFrequency, error) {
	merchantFrequency, err := s.repository.GetMerchantFrequency(ctx, startDate, endDate, limit)
	if err != nil {
		return nil, &ReceiptServiceError{
			Op:  "get_merchant_frequency",
			Err: err,
		}
	}
	return merchantFrequency, nil
}

// GetMonthlyComparison compares spending between two months
func (s *ReceiptServiceImpl) GetMonthlyComparison(ctx context.Context, month1, month2 string) (*domain.MonthlyComparison, error) {
	comparison, err := s.repository.GetMonthlyComparison(ctx, month1, month2)
	if err != nil {
		return nil, &ReceiptServiceError{
			Op:  "get_monthly_comparison",
			Err: err,
		}
	}
	return comparison, nil
}
