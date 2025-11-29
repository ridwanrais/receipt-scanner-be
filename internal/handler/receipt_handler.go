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
	"github.com/ridwanfathin/invoice-processor-service/internal/model"
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
// @Summary Scan a receipt image
// @Description Upload and process a receipt image to extract data using AI
// @Tags receipts
// @Accept multipart/form-data
// @Produce json
// @Param receiptImage formData file true "Receipt image file"
// @Success 200 {object} model.ReceiptResponse "Successfully scanned receipt"
// @Failure 400 {object} model.ErrorResponse "Bad request"
// @Failure 422 {object} model.ErrorResponse "Unable to extract data"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /v1/receipts/scan [post]
func (h *ReceiptHandler) ScanReceipt(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		respondUnauthorized(c, "User not authenticated")
		return
	}

	// Get receipt image from form data
	file, _, err := getFormFile(c, "receiptImage")
	if err != nil {
		respondBadRequest(c, err.Error(), newErrorDetail("receiptImage", "Receipt image is required"))
		return
	}
	defer file.Close()

	// Read file contents
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		logError(c, "failed_to_read_file", err, map[string]interface{}{
			"error_type": "file_read_error",
		})
		respondInternalServerError(c, ErrFileProcessing)
		return
	}

	// Process receipt image
	receipt, err := h.receiptService.ScanReceipt(c.Request.Context(), fileBytes, userID.(string))
	if err != nil {
		// Log the actual error with context
		logError(c, "failed_to_scan_receipt", err, map[string]interface{}{
			"error_type":    "service_error",
			"error_message": err.Error(),
			"file_size":     len(fileBytes),
		})

		// Check for specific error types
		if strings.Contains(fmt.Sprintf("%v", err), "not configured") {
			respondBadRequest(c, fmt.Sprintf("Configuration error: %v", err))
		} else if strings.Contains(fmt.Sprintf("%v", err), "unable to extract") {
			respondUnprocessableEntity(c, ErrDataExtraction)
		} else {
			respondInternalServerError(c, ErrFileProcessing)
		}
		return
	}

	respondOK(c, formatReceiptResponse(receipt))
}

// CreateReceipt handles the POST /receipts endpoint
// @Summary Create a new receipt
// @Description Create a new receipt with manual data entry
// @Tags receipts
// @Accept json
// @Produce json
// @Param receipt body domain.Receipt true "Receipt data"
// @Success 201 {object} model.ReceiptResponse "Receipt created successfully"
// @Failure 400 {object} model.ErrorResponse "Invalid input"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /v1/receipts [post]
func (h *ReceiptHandler) CreateReceipt(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		respondUnauthorized(c, "User not authenticated")
		return
	}

	var input domain.Receipt
	if err := bindJSON(c, &input); err != nil {
		respondBadRequest(c, ErrInvalidInput)
		return
	}

	// Set user ID
	input.UserID = userID.(string)

	// Validate required fields
	if validationErrors := validateReceiptInput(&input); len(validationErrors) > 0 {
		respondBadRequest(c, ErrInvalidInput, validationErrors...)
		return
	}

	// Create receipt
	receipt, err := h.receiptService.CreateReceipt(c.Request.Context(), &input)
	if err != nil {
		respondInternalServerError(c, fmt.Sprintf("Failed to create receipt: %v", err))
		return
	}

	respondCreated(c, formatReceiptResponse(receipt))
}

// GetReceipts handles the GET /receipts endpoint
// @Summary List all receipts
// @Description Get a paginated list of receipts with optional filters
// @Tags receipts
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param startDate query string false "Start date filter (YYYY-MM-DD)"
// @Param endDate query string false "End date filter (YYYY-MM-DD)"
// @Param merchant query string false "Merchant name filter"
// @Success 200 {object} model.ReceiptsListResponse "List of receipts"
// @Failure 400 {object} model.ErrorResponse "Invalid query parameters"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /v1/receipts [get]
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
// @Summary Get a receipt by ID
// @Description Retrieve a specific receipt by its ID
// @Tags receipts
// @Accept json
// @Produce json
// @Param receiptId path string true "Receipt ID"
// @Success 200 {object} model.ReceiptResponse "Receipt details"
// @Failure 400 {object} model.ErrorResponse "Invalid receipt ID"
// @Failure 404 {object} model.ErrorResponse "Receipt not found"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /v1/receipts/{receiptId} [get]
func (h *ReceiptHandler) GetReceiptByID(c *gin.Context) {
	receiptID, err := getPathParam(c, "receiptId")
	if err != nil {
		respondBadRequest(c, err.Error())
		return
	}

	// Get receipt
	receipt, err := h.receiptService.GetReceiptByID(c.Request.Context(), receiptID)
	if err != nil {
		if strings.Contains(fmt.Sprintf("%v", err), "not found") {
			respondNotFound(c, fmt.Sprintf("Receipt not found: %s", receiptID))
		} else {
			respondInternalServerError(c, fmt.Sprintf("Failed to retrieve receipt: %v", err))
		}
		return
	}

	respondOK(c, formatReceiptResponse(receipt))
}

// UpdateReceipt handles the PUT /receipts/{receiptId} endpoint
// @Summary Update a receipt
// @Description Update an existing receipt by ID
// @Tags receipts
// @Accept json
// @Produce json
// @Param receiptId path string true "Receipt ID"
// @Param receipt body domain.Receipt true "Updated receipt data"
// @Success 200 {object} model.ReceiptResponse "Receipt updated successfully"
// @Failure 400 {object} model.ErrorResponse "Invalid input"
// @Failure 404 {object} model.ErrorResponse "Receipt not found"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /v1/receipts/{receiptId} [put]
func (h *ReceiptHandler) UpdateReceipt(c *gin.Context) {
	receiptID, err := getPathParam(c, "receiptId")
	if err != nil {
		respondBadRequest(c, err.Error())
		return
	}

	// Parse input
	var input domain.Receipt
	if err := bindJSON(c, &input); err != nil {
		respondBadRequest(c, ErrInvalidInput)
		return
	}

	// Validate required fields
	if validationErrors := validateReceiptInput(&input); len(validationErrors) > 0 {
		respondBadRequest(c, ErrInvalidInput, validationErrors...)
		return
	}

	// Ensure ID matches path parameter
	input.ID = receiptID

	// Update receipt
	updatedReceipt, err := h.receiptService.UpdateReceipt(c.Request.Context(), &input)
	if err != nil {
		if strings.Contains(fmt.Sprintf("%v", err), "not found") {
			respondNotFound(c, fmt.Sprintf("Receipt not found: %s", receiptID))
		} else {
			respondInternalServerError(c, fmt.Sprintf("Failed to update receipt: %v", err))
		}
		return
	}

	respondOK(c, formatReceiptResponse(updatedReceipt))
}

// DeleteReceipt handles the DELETE /receipts/{receiptId} endpoint
// @Summary Delete a receipt
// @Description Delete a receipt by ID
// @Tags receipts
// @Accept json
// @Produce json
// @Param receiptId path string true "Receipt ID"
// @Success 204 "Receipt deleted successfully"
// @Failure 400 {object} model.ErrorResponse "Invalid receipt ID"
// @Failure 404 {object} model.ErrorResponse "Receipt not found"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /v1/receipts/{receiptId} [delete]
func (h *ReceiptHandler) DeleteReceipt(c *gin.Context) {
	receiptID, err := getPathParam(c, "receiptId")
	if err != nil {
		respondBadRequest(c, err.Error())
		return
	}

	// Delete receipt
	err = h.receiptService.DeleteReceipt(c.Request.Context(), receiptID)
	if err != nil {
		if strings.Contains(fmt.Sprintf("%v", err), "not found") {
			respondNotFound(c, fmt.Sprintf("Receipt not found: %s", receiptID))
		} else {
			respondInternalServerError(c, fmt.Sprintf("Failed to delete receipt: %v", err))
		}
		return
	}

	respondNoContent(c)
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
// @Summary Get dashboard summary
// @Description Get summary statistics for the dashboard
// @Tags dashboard
// @Accept json
// @Produce json
// @Param startDate query string false "Start date filter (YYYY-MM-DD)"
// @Param endDate query string false "End date filter (YYYY-MM-DD)"
// @Success 200 {object} model.DashboardSummaryResponse "Dashboard summary"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /v1/dashboard/summary [get]
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
func validateReceiptInput(receipt *domain.Receipt) []model.ErrorDetail {
	var errors []model.ErrorDetail

	if receipt.Merchant == "" {
		errors = append(errors, newErrorDetail("merchant", "Merchant is required"))
	}

	if receipt.Date.IsZero() {
		errors = append(errors, newErrorDetail("date", "Date is required"))
	}

	if receipt.Total <= 0 {
		errors = append(errors, newErrorDetail("total", "Total must be greater than zero"))
	}

	if len(receipt.Items) == 0 {
		errors = append(errors, newErrorDetail("items", "At least one item is required"))
	} else {
		for i, item := range receipt.Items {
			if item.Name == "" {
				errors = append(errors, newErrorDetail(
					fmt.Sprintf("items[%d].name", i),
					"Item name is required",
				))
			}
			if item.Quantity <= 0 {
				errors = append(errors, newErrorDetail(
					fmt.Sprintf("items[%d].qty", i),
					"Item quantity must be greater than zero",
				))
			}
			if item.Price < 0 {
				errors = append(errors, newErrorDetail(
					fmt.Sprintf("items[%d].price", i),
					"Item price cannot be negative",
				))
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
			"currency": item.Currency,
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
func (h *ReceiptHandler) RegisterRoutes(router *gin.Engine, authMiddleware gin.HandlerFunc) {
	// Create API group with base path
	api := router.Group("/v1")

	// Receipt endpoints - all protected with auth
	receipts := api.Group("/receipts", authMiddleware)
	{
		receipts.POST("/scan", h.ScanReceipt)
		receipts.POST("", h.CreateReceipt)
		receipts.GET("", h.GetReceipts)
		receipts.GET("/:receiptId", h.GetReceiptByID)
		receipts.PUT("/:receiptId", h.UpdateReceipt)
		receipts.DELETE("/:receiptId", h.DeleteReceipt)
		receipts.GET("/:receiptId/items", h.GetReceiptItems)
	}

	// Dashboard endpoints - all protected with auth
	dashboard := api.Group("/dashboard", authMiddleware)
	{
		dashboard.GET("/summary", h.GetDashboardSummary)
		dashboard.GET("/spending-trends", h.GetSpendingTrends)
	}

	// Insights endpoints - all protected with auth
	insights := api.Group("/insights", authMiddleware)
	{
		insights.GET("/spending-by-category", h.GetSpendingByCategory)
		insights.GET("/merchant-frequency", h.GetMerchantFrequency)
		insights.GET("/monthly-comparison", h.GetMonthlyComparison)
	}
}
