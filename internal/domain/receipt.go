package domain

import (
	"encoding/json"
	"strings"
	"time"
)

// ReceiptItem represents an item on a receipt
type ReceiptItem struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Quantity int     `json:"qty"`
	Price    float64 `json:"price"`
	Currency string  `json:"currency,omitempty"`
	Category string  `json:"category,omitempty"`
}

// Receipt represents a scanned or manually entered receipt
type Receipt struct {
	ID         string        `json:"id"`
	UserID     string        `json:"user_id"`
	Merchant   string        `json:"merchant"`
	Date       FlexibleDate  `json:"date"`
	Total      float64       `json:"total"`
	Tax        float64       `json:"tax,omitempty"`
	Subtotal   float64       `json:"subtotal,omitempty"`
	Items      []ReceiptItem `json:"items"`
	ImageURL   string        `json:"image_url,omitempty"`
	ReceiptURL string        `json:"receipt_url,omitempty"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
}

// FlexibleDate is a custom type that can unmarshal multiple date formats
type FlexibleDate struct {
	time.Time
}

// UnmarshalJSON implements custom JSON unmarshaling for FlexibleDate
func (fd *FlexibleDate) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	if s == "null" || s == "" {
		fd.Time = time.Time{}
		return nil
	}

	// Try multiple date formats
	formats := []string{
		"2006-01-02",          // YYYY-MM-DD
		time.RFC3339,          // 2006-01-02T15:04:05Z07:00
		"2006-01-02T15:04:05", // Without timezone
		time.RFC3339Nano,      // With nanoseconds
	}

	var err error
	for _, format := range formats {
		fd.Time, err = time.Parse(format, s)
		if err == nil {
			return nil
		}
	}

	return err
}

// MarshalJSON implements custom JSON marshaling for FlexibleDate
func (fd FlexibleDate) MarshalJSON() ([]byte, error) {
	if fd.Time.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(fd.Time.Format(time.RFC3339))
}

// ReceiptFilter represents filters for querying receipts
type ReceiptFilter struct {
	StartDate *time.Time
	EndDate   *time.Time
	Merchant  string
	Page      int
	Limit     int
}

// Pagination represents pagination metadata
type Pagination struct {
	TotalItems  int `json:"totalItems"`
	TotalPages  int `json:"totalPages"`
	CurrentPage int `json:"currentPage"`
	Limit       int `json:"limit"`
}

// PaginatedReceipts represents a paginated list of receipts
type PaginatedReceipts struct {
	Data       []Receipt  `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// DashboardSummary represents summary data for the dashboard
type DashboardSummary struct {
	TotalSpend    float64           `json:"totalSpend"`
	ReceiptCount  int               `json:"receiptCount"`
	AverageSpend  float64           `json:"averageSpend"`
	TopCategories []CategorySummary `json:"topCategories"`
	TopMerchants  []MerchantSummary `json:"topMerchants"`
}

// CategorySummary represents summary data for a spending category
type CategorySummary struct {
	Category   string  `json:"category"`
	Amount     float64 `json:"amount"`
	Percentage float64 `json:"percentage"`
}

// MerchantSummary represents summary data for a merchant
type MerchantSummary struct {
	Merchant   string  `json:"merchant"`
	Amount     float64 `json:"amount"`
	Percentage float64 `json:"percentage"`
}

// SpendingTrends represents spending trends over time
type SpendingTrends struct {
	Period string                  `json:"period"`
	Data   []SpendingTrendDataItem `json:"data"`
}

// SpendingTrendDataItem represents a single data point in spending trends
type SpendingTrendDataItem struct {
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
}

// CategorySpending represents spending breakdown by category
type CategorySpending struct {
	Total      float64                `json:"total"`
	Categories []CategorySpendingItem `json:"categories"`
}

// CategorySpendingItem represents spending data for a single category
type CategorySpendingItem struct {
	Name       string                       `json:"name"`
	Amount     float64                      `json:"amount"`
	Percentage float64                      `json:"percentage"`
	Items      []CategorySpendingItemDetail `json:"items"`
}

// CategorySpendingItemDetail represents detailed spending data for items in a category
type CategorySpendingItemDetail struct {
	Name       string  `json:"name"`
	TotalSpent float64 `json:"totalSpent"`
	Count      int     `json:"count"`
}

// MerchantFrequency represents data on frequently visited merchants
type MerchantFrequency struct {
	TotalVisits int                       `json:"totalVisits"`
	Merchants   []MerchantFrequencyDetail `json:"merchants"`
}

// MerchantFrequencyDetail represents detailed data for a merchant's frequency
type MerchantFrequencyDetail struct {
	Name         string  `json:"name"`
	Visits       int     `json:"visits"`
	TotalSpent   float64 `json:"totalSpent"`
	AverageSpent float64 `json:"averageSpent"`
	Percentage   float64 `json:"percentage"`
}

// MonthlyComparison represents a comparison between two months
type MonthlyComparison struct {
	Month1           string                      `json:"month1"`
	Month2           string                      `json:"month2"`
	Month1Total      float64                     `json:"month1Total"`
	Month2Total      float64                     `json:"month2Total"`
	Difference       float64                     `json:"difference"`
	PercentageChange float64                     `json:"percentageChange"`
	Categories       []MonthlyCategoryComparison `json:"categories"`
}

// MonthlyCategoryComparison represents a comparison between two months for a specific category
type MonthlyCategoryComparison struct {
	Name             string  `json:"name"`
	Month1Amount     float64 `json:"month1Amount"`
	Month2Amount     float64 `json:"month2Amount"`
	Difference       float64 `json:"difference"`
	PercentageChange float64 `json:"percentageChange"`
}
