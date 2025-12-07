package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ridwanfathin/invoice-processor-service/internal/currency"
	"github.com/ridwanfathin/invoice-processor-service/internal/repository"
)

// AnalyticsHandler handles analytics endpoints with currency conversion
type AnalyticsHandler struct {
	receiptRepo    repository.ReceiptRepository
	currencyClient *currency.Client
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(receiptRepo repository.ReceiptRepository, currencyClient *currency.Client) *AnalyticsHandler {
	return &AnalyticsHandler{
		receiptRepo:    receiptRepo,
		currencyClient: currencyClient,
	}
}

// AnalyticsSummary represents the analytics summary response
type AnalyticsSummary struct {
	TotalSpent   float64                `json:"totalSpent"`
	ReceiptCount int                    `json:"receiptCount"`
	Average      float64                `json:"average"`
	Highest      float64                `json:"highest"`
	Currency     string                 `json:"currency"`
	ByCategory   []CategoryAmount       `json:"byCategory"`
	ByPeriod     []PeriodAmount         `json:"byPeriod"`
}

// CategoryAmount represents spending by category
type CategoryAmount struct {
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
}

// PeriodAmount represents spending by time period
type PeriodAmount struct {
	Period string  `json:"period"`
	Amount float64 `json:"amount"`
	Count  int     `json:"count"`
}

// GetAnalytics handles GET /v1/analytics endpoint
// @Summary Get analytics with currency conversion
// @Description Get spending analytics with all amounts converted to target currency
// @Tags analytics
// @Accept json
// @Produce json
// @Param currency query string false "Target currency (default: USD)"
// @Param period query string false "Period type: weekly, monthly, yearly (default: monthly)"
// @Param startDate query string false "Start date filter (YYYY-MM-DD)"
// @Param endDate query string false "End date filter (YYYY-MM-DD)"
// @Success 200 {object} AnalyticsSummary "Analytics summary"
// @Failure 401 {object} model.ErrorResponse "Unauthorized"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /v1/analytics [get]
func (h *AnalyticsHandler) GetAnalytics(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "401",
			"message": "User not authenticated",
		})
		return
	}

	// Parse parameters
	targetCurrency := c.DefaultQuery("currency", "USD")
	periodType := c.DefaultQuery("period", "monthly")
	startDateStr := c.Query("startDate")
	endDateStr := c.Query("endDate")

	// Get exchange rates for target currency
	rates, err := h.currencyClient.GetLatestRates(c.Request.Context(), targetCurrency)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "500",
			"message": "Failed to fetch exchange rates: " + err.Error(),
		})
		return
	}

	// Fetch all receipts for user with items
	filter := repository.ReceiptFilterWithItems{
		UserID:    userID.(string),
		StartDate: parseDateParam(startDateStr),
		EndDate:   parseDateParam(endDateStr),
	}

	receipts, err := h.receiptRepo.GetReceiptsWithItems(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "500",
			"message": "Failed to fetch receipts: " + err.Error(),
		})
		return
	}

	// Calculate analytics with currency conversion
	summary := AnalyticsSummary{
		Currency:   targetCurrency,
		ByCategory: []CategoryAmount{},
		ByPeriod:   []PeriodAmount{},
	}

	categoryTotals := make(map[string]float64)
	periodTotals := make(map[string]*PeriodAmount)
	var highest float64

	for _, receipt := range receipts {
		var receiptTotal float64

		// Sum up items with currency conversion
		for _, item := range receipt.Items {
			itemTotal := float64(item.Quantity) * item.Price
			convertedAmount := convertToTarget(itemTotal, item.Currency, targetCurrency, rates)
			receiptTotal += convertedAmount

			// Track by category
			category := item.Category
			if category == "" {
				category = "Uncategorized"
			}
			categoryTotals[category] += convertedAmount
		}

		// If no items, use receipt total (assume USD if no currency info)
		if len(receipt.Items) == 0 {
			receiptTotal = receipt.Total // Already in some currency, assume target
		}

		summary.TotalSpent += receiptTotal
		summary.ReceiptCount++

		if receiptTotal > highest {
			highest = receiptTotal
		}

		// Track by period
		periodKey := getPeriodKey(receipt.Date.Time, periodType)
		if _, ok := periodTotals[periodKey]; !ok {
			periodTotals[periodKey] = &PeriodAmount{Period: periodKey}
		}
		periodTotals[periodKey].Amount += receiptTotal
		periodTotals[periodKey].Count++
	}

	summary.Highest = highest
	if summary.ReceiptCount > 0 {
		summary.Average = summary.TotalSpent / float64(summary.ReceiptCount)
	}

	// Convert maps to slices
	for category, amount := range categoryTotals {
		summary.ByCategory = append(summary.ByCategory, CategoryAmount{
			Category: category,
			Amount:   amount,
		})
	}

	for _, period := range periodTotals {
		summary.ByPeriod = append(summary.ByPeriod, *period)
	}

	c.JSON(http.StatusOK, summary)
}

// convertToTarget converts an amount from source currency to target currency
func convertToTarget(amount float64, sourceCurrency, targetCurrency string, rates *currency.ExchangeRates) float64 {
	if sourceCurrency == "" {
		sourceCurrency = "USD" // Default assumption
	}
	if sourceCurrency == targetCurrency {
		return amount
	}

	// rates.Rates contains rates FROM targetCurrency TO other currencies
	// So to convert FROM sourceCurrency TO targetCurrency, we divide by the rate
	rate, ok := rates.Rates[sourceCurrency]
	if !ok {
		return amount // Can't convert, return as-is
	}

	return amount / rate
}

// getPeriodKey returns a period key for grouping
func getPeriodKey(date time.Time, periodType string) string {
	switch periodType {
	case "weekly":
		year, week := date.ISOWeek()
		return time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, (week-1)*7).Format("2006-01-02")
	case "yearly":
		return date.Format("2006")
	default: // monthly
		return date.Format("2006-01")
	}
}

// parseDateParam parses a date string parameter
func parseDateParam(dateStr string) *time.Time {
	if dateStr == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil
	}
	return &t
}

// RegisterAnalyticsRoutes registers analytics routes
func (h *AnalyticsHandler) RegisterAnalyticsRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	analytics := router.Group("/analytics", authMiddleware)
	{
		analytics.GET("", h.GetAnalytics)
	}
}
