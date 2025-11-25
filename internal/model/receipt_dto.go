package model

// ReceiptResponse represents the response for a single receipt
type ReceiptResponse struct {
	ID        string                `json:"id"`
	Merchant  string                `json:"merchant"`
	Date      string                `json:"date"`
	Total     string                `json:"total"`
	Tax       string                `json:"tax"`
	Subtotal  string                `json:"subtotal"`
	Items     []ReceiptItemResponse `json:"items"`
	CreatedAt string                `json:"createdAt"`
	UpdatedAt string                `json:"updatedAt"`
}

// ReceiptItemResponse represents a single receipt item
type ReceiptItemResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Qty      int    `json:"qty"`
	Price    string `json:"price"`
	Currency string `json:"currency,omitempty"`
	Category string `json:"category"`
}

// ReceiptsListResponse represents paginated list of receipts
type ReceiptsListResponse struct {
	Data       []ReceiptResponse  `json:"data"`
	Pagination PaginationResponse `json:"pagination"`
}

// PaginationResponse represents pagination metadata
type PaginationResponse struct {
	TotalItems  int `json:"totalItems"`
	TotalPages  int `json:"totalPages"`
	CurrentPage int `json:"currentPage"`
	Limit       int `json:"limit"`
}

// DashboardSummaryResponse represents dashboard summary statistics
type DashboardSummaryResponse struct {
	TotalSpend    string            `json:"totalSpend"`
	ReceiptCount  int               `json:"receiptCount"`
	AverageSpend  string            `json:"averageSpend"`
	TopCategories []CategorySummary `json:"topCategories"`
	TopMerchants  []MerchantSummary `json:"topMerchants"`
}

// CategorySummary represents category spending summary
type CategorySummary struct {
	Category   string  `json:"category"`
	Amount     string  `json:"amount"`
	Percentage float64 `json:"percentage"`
}

// MerchantSummary represents merchant spending summary
type MerchantSummary struct {
	Merchant   string  `json:"merchant"`
	Amount     string  `json:"amount"`
	Percentage float64 `json:"percentage"`
}

// SpendingTrendsResponse represents spending trends over time
type SpendingTrendsResponse struct {
	Period string           `json:"period"`
	Data   []TrendDataPoint `json:"data"`
}

// TrendDataPoint represents a single data point in spending trends
type TrendDataPoint struct {
	Date   string `json:"date"`
	Amount string `json:"amount"`
}

// CategorySpendingResponse represents spending breakdown by category
type CategorySpendingResponse struct {
	Total      string                   `json:"total"`
	Categories []CategorySpendingDetail `json:"categories"`
}

// CategorySpendingDetail represents detailed spending for a category
type CategorySpendingDetail struct {
	Name       string               `json:"name"`
	Amount     string               `json:"amount"`
	Percentage float64              `json:"percentage"`
	Items      []CategoryItemDetail `json:"items"`
}

// CategoryItemDetail represents item-level spending within a category
type CategoryItemDetail struct {
	Name       string `json:"name"`
	TotalSpent string `json:"totalSpent"`
	Count      int    `json:"count"`
}

// MerchantFrequencyResponse represents merchant visit frequency
type MerchantFrequencyResponse struct {
	TotalVisits int                       `json:"totalVisits"`
	Merchants   []MerchantFrequencyDetail `json:"merchants"`
}

// MerchantFrequencyDetail represents detailed merchant frequency data
type MerchantFrequencyDetail struct {
	Name         string  `json:"name"`
	Visits       int     `json:"visits"`
	TotalSpent   string  `json:"totalSpent"`
	AverageSpent string  `json:"averageSpent"`
	Percentage   float64 `json:"percentage"`
}

// MonthlyComparisonResponse represents comparison between two months
type MonthlyComparisonResponse struct {
	Month1           string                      `json:"month1"`
	Month2           string                      `json:"month2"`
	Month1Total      string                      `json:"month1Total"`
	Month2Total      string                      `json:"month2Total"`
	Difference       string                      `json:"difference"`
	PercentageChange float64                     `json:"percentageChange"`
	Categories       []MonthlyComparisonCategory `json:"categories"`
}

// MonthlyComparisonCategory represents category comparison between months
type MonthlyComparisonCategory struct {
	Name             string  `json:"name"`
	Month1Amount     string  `json:"month1Amount"`
	Month2Amount     string  `json:"month2Amount"`
	Difference       string  `json:"difference"`
	PercentageChange float64 `json:"percentageChange"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Status  string        `json:"status"`
	Message string        `json:"message"`
	Details []ErrorDetail `json:"details,omitempty"`
}

// ErrorDetail represents detailed error information
type ErrorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
