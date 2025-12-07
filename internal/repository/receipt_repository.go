package repository

import (
	"context"
	"time"

	"github.com/ridwanfathin/invoice-processor-service/internal/domain"
)

// ReceiptFilterWithItems is a filter for fetching receipts with items
type ReceiptFilterWithItems struct {
	UserID    string
	StartDate *time.Time
	EndDate   *time.Time
}

// ReceiptRepository defines the interface for receipt data operations
type ReceiptRepository interface {
	// Receipt CRUD operations
	CreateReceipt(ctx context.Context, receipt *domain.Receipt) (*domain.Receipt, error)
	GetReceiptByID(ctx context.Context, receiptID string) (*domain.Receipt, error)
	UpdateReceipt(ctx context.Context, receipt *domain.Receipt) (*domain.Receipt, error)
	DeleteReceipt(ctx context.Context, receiptID string) error

	// Receipt querying operations
	ListReceipts(ctx context.Context, filter domain.ReceiptFilter) (*domain.PaginatedReceipts, error)
	GetReceiptItems(ctx context.Context, receiptID string) ([]domain.ReceiptItem, error)
	GetReceiptsWithItems(ctx context.Context, filter ReceiptFilterWithItems) ([]domain.Receipt, error)

	// Dashboard and insights operations
	GetDashboardSummary(ctx context.Context, userID string, startDate, endDate *string) (*domain.DashboardSummary, error)
	GetSpendingTrends(ctx context.Context, userID string, period string, startDate, endDate *string) (*domain.SpendingTrends, error)
	GetSpendingByCategory(ctx context.Context, userID string, startDate, endDate *string) (*domain.CategorySpending, error)
	GetMerchantFrequency(ctx context.Context, userID string, startDate, endDate *string, limit int) (*domain.MerchantFrequency, error)
	GetMonthlyComparison(ctx context.Context, userID string, month1, month2 string) (*domain.MonthlyComparison, error)
}
