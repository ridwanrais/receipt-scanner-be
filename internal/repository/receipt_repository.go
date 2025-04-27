package repository

import (
	"context"

	"github.com/ridwanfathin/invoice-processor-service/internal/domain"
)

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
	
	// Dashboard and insights operations
	GetDashboardSummary(ctx context.Context, startDate, endDate *string) (*domain.DashboardSummary, error)
	GetSpendingTrends(ctx context.Context, period string, startDate, endDate *string) (*domain.SpendingTrends, error)
	GetSpendingByCategory(ctx context.Context, startDate, endDate *string) (*domain.CategorySpending, error)
	GetMerchantFrequency(ctx context.Context, startDate, endDate *string, limit int) (*domain.MerchantFrequency, error)
	GetMonthlyComparison(ctx context.Context, month1, month2 string) (*domain.MonthlyComparison, error)
}
