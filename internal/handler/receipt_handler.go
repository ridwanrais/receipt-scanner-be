package handler

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ridwanfathin/invoice-processor-service/internal/domain"
	"github.com/ridwanfathin/invoice-processor-service/internal/service"
)

// ReceiptHandler handles HTTP requests for receipt-related operations
type ReceiptHandler struct {
	receiptService service.ReceiptService
}

// NewReceiptHandler creates a new receipt handler
func NewReceiptHandler(receiptService service.ReceiptService) *ReceiptHandler {
	return &ReceiptHandler{
		receiptService: receiptService,
	}
}

// ScanReceipt handles the POST /receipts/scan endpoint
func (h *ReceiptHandler) ScanReceipt(c *gin.Context) {
	// Get receipt image from form data
	file, _, err := c.Request.FormFile("receiptImage")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "400",
			"message": "No receipt image provided",
			"details": []gin.H{
				{
					"field":   "receiptImage",
					"message": "Receipt image is required",
				},
			},
		})
		return
	}
	defer file.Close()

	// Read file contents
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "500",
			"message": "Failed to read receipt image",
		})
		return
	}

	// Process receipt image
	receipt, err := h.receiptService.ScanReceipt(c.Request.Context(), fileBytes)
	if err != nil {
		statusCode := http.StatusInternalServerError
		message := "Failed to process receipt"

		// Check for specific error types
		if strings.Contains(fmt.Sprintf("%v", err), "not configured") {
			statusCode = http.StatusBadRequest
			message = fmt.Sprintf("Configuration error: %v", err)
		} else if strings.Contains(fmt.Sprintf("%v", err), "unable to extract") {
			statusCode = http.StatusUnprocessableEntity
			message = "Unable to extract data from receipt image"
		}

		c.JSON(statusCode, gin.H{
			"status":  strconv.Itoa(statusCode),
			"message": message,
		})
		return
	}

	// Format response
	c.JSON(http.StatusOK, formatReceiptResponse(receipt))
}

// CreateReceipt handles the POST /receipts endpoint
func (h *ReceiptHandler) CreateReceipt(c *gin.Context) {
	var input domain.Receipt
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "400",
			"message": "Invalid input format",
		})
		return
	}

	// Validate required fields
	if err := validateReceiptInput(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "400",
			"message": "Invalid input",
			"details": err,
		})
		return
	}

	// Create receipt
	receipt, err := h.receiptService.CreateReceipt(c.Request.Context(), &input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "500",
			"message": fmt.Sprintf("Failed to create receipt: %v", err),
		})
		return
	}

	// Return created receipt
	c.JSON(http.StatusCreated, formatReceiptResponse(receipt))
}

// GetReceipts handles the GET /receipts endpoint
func (h *ReceiptHandler) GetReceipts(c *gin.Context) {
	// Parse query parameters
	filter, err := parseReceiptFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "400",
			"message": "Invalid query parameters",
			"details": []gin.H{
				{
					"field":   "query",
					"message": err.Error(),
				},
			},
		})
		return
	}

	// Get receipts
	paginatedReceipts, err := h.receiptService.ListReceipts(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "500",
			"message": fmt.Sprintf("Failed to retrieve receipts: %v", err),
		})
		return
	}

	// Format response
	response := gin.H{
		"data": formatReceiptsResponse(paginatedReceipts.Data),
		"pagination": gin.H{
			"totalItems":  paginatedReceipts.Pagination.TotalItems,
			"totalPages":  paginatedReceipts.Pagination.TotalPages,
			"currentPage": paginatedReceipts.Pagination.CurrentPage,
			"limit":       paginatedReceipts.Pagination.Limit,
		},
	}
	c.JSON(http.StatusOK, response)
}

// GetReceiptByID handles the GET /receipts/{receiptId} endpoint
func (h *ReceiptHandler) GetReceiptByID(c *gin.Context) {
	receiptID := c.Param("receiptId")
	if receiptID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "400",
			"message": "Receipt ID is required",
		})
		return
	}

	// Get receipt
	receipt, err := h.receiptService.GetReceiptByID(c.Request.Context(), receiptID)
	if err != nil {
		if strings.Contains(fmt.Sprintf("%v", err), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"status":  "404",
				"message": fmt.Sprintf("Receipt not found: %s", receiptID),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "500",
			"message": fmt.Sprintf("Failed to retrieve receipt: %v", err),
		})
		return
	}

	// Return receipt
	c.JSON(http.StatusOK, formatReceiptResponse(receipt))
}

// UpdateReceipt handles the PUT /receipts/{receiptId} endpoint
func (h *ReceiptHandler) UpdateReceipt(c *gin.Context) {
	receiptID := c.Param("receiptId")
	if receiptID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "400",
			"message": "Receipt ID is required",
		})
		return
	}

	// Parse input
	var input domain.Receipt
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "400",
			"message": "Invalid input format",
		})
		return
	}

	// Validate required fields
	if err := validateReceiptInput(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "400",
			"message": "Invalid input",
			"details": err,
		})
		return
	}

	// Ensure ID matches path parameter
	input.ID = receiptID

	// Update receipt
	updatedReceipt, err := h.receiptService.UpdateReceipt(c.Request.Context(), &input)
	if err != nil {
		if strings.Contains(fmt.Sprintf("%v", err), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"status":  "404",
				"message": fmt.Sprintf("Receipt not found: %s", receiptID),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "500",
			"message": fmt.Sprintf("Failed to update receipt: %v", err),
		})
		return
	}

	// Return updated receipt
	c.JSON(http.StatusOK, formatReceiptResponse(updatedReceipt))
}

// DeleteReceipt handles the DELETE /receipts/{receiptId} endpoint
func (h *ReceiptHandler) DeleteReceipt(c *gin.Context) {
	receiptID := c.Param("receiptId")
	if receiptID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "400",
			"message": "Receipt ID is required",
		})
		return
	}

	// Delete receipt
	err := h.receiptService.DeleteReceipt(c.Request.Context(), receiptID)
	if err != nil {
		if strings.Contains(fmt.Sprintf("%v", err), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"status":  "404",
				"message": fmt.Sprintf("Receipt not found: %s", receiptID),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "500",
			"message": fmt.Sprintf("Failed to delete receipt: %v", err),
		})
		return
	}

	// Return success with no content
	c.Status(http.StatusNoContent)
}

// GetReceiptItems handles the GET /receipts/{receiptId}/items endpoint
func (h *ReceiptHandler) GetReceiptItems(c *gin.Context) {
	receiptID := c.Param("receiptId")
	if receiptID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "400",
			"message": "Receipt ID is required",
		})
		return
	}

	// Get receipt items
	items, err := h.receiptService.GetReceiptItems(c.Request.Context(), receiptID)
	if err != nil {
		if strings.Contains(fmt.Sprintf("%v", err), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"status":  "404",
				"message": fmt.Sprintf("Receipt not found: %s", receiptID),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "500",
			"message": fmt.Sprintf("Failed to retrieve receipt items: %v", err),
		})
		return
	}

	// Return items
	c.JSON(http.StatusOK, formatReceiptItemsResponse(items))
}

// GetDashboardSummary handles the GET /dashboard/summary endpoint
func (h *ReceiptHandler) GetDashboardSummary(c *gin.Context) {
	// Parse query parameters
	startDate, endDate := parseDateRange(c)

	// Get dashboard summary
	summary, err := h.receiptService.GetDashboardSummary(c.Request.Context(), startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "500",
			"message": fmt.Sprintf("Failed to retrieve dashboard summary: %v", err),
		})
		return
	}

	// Format response
	response := formatDashboardSummaryResponse(summary)
	c.JSON(http.StatusOK, response)
}

// GetSpendingTrends handles the GET /dashboard/spending-trends endpoint
func (h *ReceiptHandler) GetSpendingTrends(c *gin.Context) {
	// Parse query parameters
	period := c.DefaultQuery("period", "monthly")
	startDate, endDate := parseDateRange(c)

	// Validate period
	validPeriods := map[string]bool{
		"daily":   true,
		"weekly":  true,
		"monthly": true,
		"yearly":  true,
	}
	if !validPeriods[period] {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "400",
			"message": "Invalid period parameter",
			"details": []gin.H{
				{
					"field":   "period",
					"message": "Period must be one of: daily, weekly, monthly, yearly",
				},
			},
		})
		return
	}

	// Get spending trends
	trends, err := h.receiptService.GetSpendingTrends(c.Request.Context(), period, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "500",
			"message": fmt.Sprintf("Failed to retrieve spending trends: %v", err),
		})
		return
	}

	// Format response
	response := formatSpendingTrendsResponse(trends)
	c.JSON(http.StatusOK, response)
}

// GetSpendingByCategory handles the GET /insights/spending-by-category endpoint
func (h *ReceiptHandler) GetSpendingByCategory(c *gin.Context) {
	// Parse query parameters
	startDate, endDate := parseDateRange(c)

	// Get spending by category
	categorySpending, err := h.receiptService.GetSpendingByCategory(c.Request.Context(), startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "500",
			"message": fmt.Sprintf("Failed to retrieve category spending: %v", err),
		})
		return
	}

	// Format response
	response := formatCategorySpendingResponse(categorySpending)
	c.JSON(http.StatusOK, response)
}

// GetMerchantFrequency handles the GET /insights/merchant-frequency endpoint
func (h *ReceiptHandler) GetMerchantFrequency(c *gin.Context) {
	// Parse query parameters
	startDate, endDate := parseDateRange(c)

	// Parse limit
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	// Get merchant frequency
	merchantFrequency, err := h.receiptService.GetMerchantFrequency(c.Request.Context(), startDate, endDate, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "500",
			"message": fmt.Sprintf("Failed to retrieve merchant frequency: %v", err),
		})
		return
	}

	// Format response
	response := formatMerchantFrequencyResponse(merchantFrequency)
	c.JSON(http.StatusOK, response)
}

// GetMonthlyComparison handles the GET /insights/monthly-comparison endpoint
func (h *ReceiptHandler) GetMonthlyComparison(c *gin.Context) {
	// Parse query parameters
	month1 := c.Query("month1")
	month2 := c.Query("month2")

	// Validate month format
	if !isValidMonth(month1) || !isValidMonth(month2) {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "400",
			"message": "Invalid month format",
			"details": []gin.H{
				{
					"field":   "month1/month2",
					"message": "Months must be in YYYY-MM format (e.g., 2023-01)",
				},
			},
		})
		return
	}

	// Get monthly comparison
	comparison, err := h.receiptService.GetMonthlyComparison(c.Request.Context(), month1, month2)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "500",
			"message": fmt.Sprintf("Failed to retrieve monthly comparison: %v", err),
		})
		return
	}

	// Format response
	response := formatMonthlyComparisonResponse(comparison)
	c.JSON(http.StatusOK, response)
}

// Helper functions

// validateReceiptInput validates required fields in a receipt
func validateReceiptInput(receipt *domain.Receipt) []gin.H {
	var errors []gin.H

	if receipt.Merchant == "" {
		errors = append(errors, gin.H{
			"field":   "merchant",
			"message": "Merchant is required",
		})
	}

	if receipt.Date.IsZero() {
		errors = append(errors, gin.H{
			"field":   "date",
			"message": "Date is required",
		})
	}

	if receipt.Total <= 0 {
		errors = append(errors, gin.H{
			"field":   "total",
			"message": "Total must be greater than zero",
		})
	}

	if len(receipt.Items) == 0 {
		errors = append(errors, gin.H{
			"field":   "items",
			"message": "At least one item is required",
		})
	} else {
		for i, item := range receipt.Items {
			if item.Name == "" {
				errors = append(errors, gin.H{
					"field":   fmt.Sprintf("items[%d].name", i),
					"message": "Item name is required",
				})
			}
			if item.Quantity <= 0 {
				errors = append(errors, gin.H{
					"field":   fmt.Sprintf("items[%d].qty", i),
					"message": "Item quantity must be greater than zero",
				})
			}
			if item.Price < 0 {
				errors = append(errors, gin.H{
					"field":   fmt.Sprintf("items[%d].price", i),
					"message": "Item price cannot be negative",
				})
			}
		}
	}

	return errors
}

// parseReceiptFilter extracts filtering parameters from request
func parseReceiptFilter(c *gin.Context) (domain.ReceiptFilter, error) {
	filter := domain.ReceiptFilter{}

	// Parse pagination parameters
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		return filter, fmt.Errorf("invalid page number")
	}
	filter.Page = page

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		return filter, fmt.Errorf("invalid limit")
	}
	if limit > 100 {
		limit = 100
	}
	filter.Limit = limit

	// Parse date range
	startDateStr := c.Query("startDate")
	if startDateStr != "" {
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			return filter, fmt.Errorf("invalid startDate format (use YYYY-MM-DD)")
		}
		filter.StartDate = &startDate
	}

	endDateStr := c.Query("endDate")
	if endDateStr != "" {
		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			return filter, fmt.Errorf("invalid endDate format (use YYYY-MM-DD)")
		}
		filter.EndDate = &endDate
	}

	// Parse merchant filter
	filter.Merchant = c.Query("merchant")

	return filter, nil
}

// parseDateRange extracts date range parameters from request
func parseDateRange(c *gin.Context) (*string, *string) {
	startDateStr := c.Query("startDate")
	endDateStr := c.Query("endDate")

	var startDate, endDate *string
	if startDateStr != "" {
		startDate = &startDateStr
	}
	if endDateStr != "" {
		endDate = &endDateStr
	}

	return startDate, endDate
}

// isValidMonth checks if a string is in the format YYYY-MM
func isValidMonth(month string) bool {
	_, err := time.Parse("2006-01", month)
	return err == nil
}

// formatReceiptResponse formats a receipt for response
func formatReceiptResponse(receipt *domain.Receipt) gin.H {
	return gin.H{
		"id":        receipt.ID,
		"merchant":  receipt.Merchant,
		"date":      receipt.Date.Format("2006-01-02"),
		"total":     fmt.Sprintf("%.2f", receipt.Total),
		"tax":       fmt.Sprintf("%.2f", receipt.Tax),
		"subtotal":  fmt.Sprintf("%.2f", receipt.Subtotal),
		"items":     formatReceiptItemsResponse(receipt.Items),
		"createdAt": receipt.CreatedAt.Format(time.RFC3339),
		"updatedAt": receipt.UpdatedAt.Format(time.RFC3339),
	}
}

// formatReceiptsResponse formats a slice of receipts for response
func formatReceiptsResponse(receipts []domain.Receipt) []gin.H {
	formatted := make([]gin.H, len(receipts))
	for i, receipt := range receipts {
		formatted[i] = formatReceiptResponse(&receipt)
	}
	return formatted
}

// formatReceiptItemsResponse formats receipt items for response
func formatReceiptItemsResponse(items []domain.ReceiptItem) []gin.H {
	formatted := make([]gin.H, len(items))
	for i, item := range items {
		formatted[i] = gin.H{
			"id":       item.ID,
			"name":     item.Name,
			"qty":      item.Quantity,
			"price":    fmt.Sprintf("%.2f", item.Price),
			"category": item.Category,
		}
	}
	return formatted
}

// formatDashboardSummaryResponse formats dashboard summary for response
func formatDashboardSummaryResponse(summary *domain.DashboardSummary) gin.H {
	topCategories := make([]gin.H, len(summary.TopCategories))
	for i, category := range summary.TopCategories {
		topCategories[i] = gin.H{
			"category":   category.Category,
			"amount":     fmt.Sprintf("%.2f", category.Amount),
			"percentage": category.Percentage,
		}
	}

	topMerchants := make([]gin.H, len(summary.TopMerchants))
	for i, merchant := range summary.TopMerchants {
		topMerchants[i] = gin.H{
			"merchant":   merchant.Merchant,
			"amount":     fmt.Sprintf("%.2f", merchant.Amount),
			"percentage": merchant.Percentage,
		}
	}

	return gin.H{
		"totalSpend":    fmt.Sprintf("%.2f", summary.TotalSpend),
		"receiptCount":  summary.ReceiptCount,
		"averageSpend":  fmt.Sprintf("%.2f", summary.AverageSpend),
		"topCategories": topCategories,
		"topMerchants":  topMerchants,
	}
}

// formatSpendingTrendsResponse formats spending trends for response
func formatSpendingTrendsResponse(trends *domain.SpendingTrends) gin.H {
	data := make([]gin.H, len(trends.Data))
	for i, item := range trends.Data {
		data[i] = gin.H{
			"date":   item.Date,
			"amount": fmt.Sprintf("%.2f", item.Amount),
		}
	}

	return gin.H{
		"period": trends.Period,
		"data":   data,
	}
}

// formatCategorySpendingResponse formats category spending for response
func formatCategorySpendingResponse(spending *domain.CategorySpending) gin.H {
	categories := make([]gin.H, len(spending.Categories))
	for i, category := range spending.Categories {
		items := make([]gin.H, len(category.Items))
		for j, item := range category.Items {
			items[j] = gin.H{
				"name":       item.Name,
				"totalSpent": fmt.Sprintf("%.2f", item.TotalSpent),
				"count":      item.Count,
			}
		}

		categories[i] = gin.H{
			"name":       category.Name,
			"amount":     fmt.Sprintf("%.2f", category.Amount),
			"percentage": category.Percentage,
			"items":      items,
		}
	}

	return gin.H{
		"total":      fmt.Sprintf("%.2f", spending.Total),
		"categories": categories,
	}
}

// formatMerchantFrequencyResponse formats merchant frequency for response
func formatMerchantFrequencyResponse(frequency *domain.MerchantFrequency) gin.H {
	merchants := make([]gin.H, len(frequency.Merchants))
	for i, merchant := range frequency.Merchants {
		merchants[i] = gin.H{
			"name":         merchant.Name,
			"visits":       merchant.Visits,
			"totalSpent":   fmt.Sprintf("%.2f", merchant.TotalSpent),
			"averageSpent": fmt.Sprintf("%.2f", merchant.AverageSpent),
			"percentage":   merchant.Percentage,
		}
	}

	return gin.H{
		"totalVisits": frequency.TotalVisits,
		"merchants":   merchants,
	}
}

// formatMonthlyComparisonResponse formats monthly comparison for response
func formatMonthlyComparisonResponse(comparison *domain.MonthlyComparison) gin.H {
	categories := make([]gin.H, len(comparison.Categories))
	for i, category := range comparison.Categories {
		categories[i] = gin.H{
			"name":             category.Name,
			"month1Amount":     fmt.Sprintf("%.2f", category.Month1Amount),
			"month2Amount":     fmt.Sprintf("%.2f", category.Month2Amount),
			"difference":       fmt.Sprintf("%.2f", category.Difference),
			"percentageChange": category.PercentageChange,
		}
	}

	return gin.H{
		"month1":           comparison.Month1,
		"month2":           comparison.Month2,
		"month1Total":      fmt.Sprintf("%.2f", comparison.Month1Total),
		"month2Total":      fmt.Sprintf("%.2f", comparison.Month2Total),
		"difference":       fmt.Sprintf("%.2f", comparison.Difference),
		"percentageChange": comparison.PercentageChange,
		"categories":       categories,
	}
}

// RegisterRoutes registers the API routes for the receipt handler
func (h *ReceiptHandler) RegisterRoutes(router *gin.Engine) {
	// Create API group with base path
	api := router.Group("/v1")

	// Receipt endpoints
	receipts := api.Group("/receipts")
	{
		receipts.POST("/scan", h.ScanReceipt)
		receipts.POST("", h.CreateReceipt)
		receipts.GET("", h.GetReceipts)
		receipts.GET("/:receiptId", h.GetReceiptByID)
		receipts.PUT("/:receiptId", h.UpdateReceipt)
		receipts.DELETE("/:receiptId", h.DeleteReceipt)
		receipts.GET("/:receiptId/items", h.GetReceiptItems)
	}

	// Dashboard endpoints
	dashboard := api.Group("/dashboard")
	{
		dashboard.GET("/summary", h.GetDashboardSummary)
		dashboard.GET("/spending-trends", h.GetSpendingTrends)
	}

	// Insights endpoints
	insights := api.Group("/insights")
	{
		insights.GET("/spending-by-category", h.GetSpendingByCategory)
		insights.GET("/merchant-frequency", h.GetMerchantFrequency)
		insights.GET("/monthly-comparison", h.GetMonthlyComparison)
	}
}
