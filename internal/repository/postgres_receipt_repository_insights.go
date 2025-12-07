package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ridwanfathin/invoice-processor-service/internal/domain"
)

// GetDashboardSummary retrieves summary data for the dashboard
func (r *PostgresReceiptRepository) GetDashboardSummary(ctx context.Context, userID string, startDateStr, endDateStr *string) (*domain.DashboardSummary, error) {
	// Parse date strings if provided
	var startDate, endDate *time.Time

	if startDateStr != nil {
		parsedDate, err := time.Parse("2006-01-02", *startDateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid start date format: %w", err)
		}
		startDate = &parsedDate
	}

	if endDateStr != nil {
		parsedDate, err := time.Parse("2006-01-02", *endDateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid end date format: %w", err)
		}
		endDate = &parsedDate
	}

	// Build query conditions for date filtering
	conditions := []string{}
	args := []interface{}{}
	argCount := 1

	// Always filter by user ID
	if userID != "" {
		conditions = append(conditions, fmt.Sprintf("r.user_id = $%d", argCount))
		args = append(args, userID)
		argCount++
	}

	if startDate != nil {
		conditions = append(conditions, fmt.Sprintf("r.date >= $%d::date", argCount))
		args = append(args, startDate)
		argCount++
	}
	if endDate != nil {
		conditions = append(conditions, fmt.Sprintf("r.date <= $%d::date", argCount))
		args = append(args, endDate)
		argCount++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Initialize summary with default values
	summary := &domain.DashboardSummary{
		TopCategories: []domain.CategorySummary{},
		TopMerchants:  []domain.MerchantSummary{},
	}

	// Get total spend, receipt count, and average spend
	err := r.db.QueryRow(ctx, fmt.Sprintf(`
		SELECT 
			COALESCE(SUM(total), 0) as total_spend,
			COUNT(*) as receipt_count
		FROM receipts r
		%s
	`, whereClause), args...).Scan(&summary.TotalSpend, &summary.ReceiptCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard summary: %w", err)
	}

	// Calculate average spend
	if summary.ReceiptCount > 0 {
		summary.AverageSpend = summary.TotalSpend / float64(summary.ReceiptCount)
	}

	// Get top categories
	categoryArgs := make([]interface{}, len(args))
	copy(categoryArgs, args)

	categoryRows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT 
			ri.category, 
			COALESCE(SUM(ri.qty * ri.price), 0) as amount,
			COALESCE(SUM(ri.qty * ri.price) / NULLIF((SELECT SUM(total) FROM receipts r %s), 0) * 100, 0) as percentage
		FROM receipt_items ri
		JOIN receipts r ON ri.receipt_id = r.id
		%s
		GROUP BY ri.category
		HAVING ri.category IS NOT NULL
		ORDER BY amount DESC
		LIMIT 5
	`, whereClause, whereClause), categoryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to get top categories: %w", err)
	}
	defer categoryRows.Close()

	for categoryRows.Next() {
		var category domain.CategorySummary
		if err := categoryRows.Scan(&category.Category, &category.Amount, &category.Percentage); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		summary.TopCategories = append(summary.TopCategories, category)
	}

	if err := categoryRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating categories: %w", err)
	}

	// Get top merchants
	merchantArgs := make([]interface{}, len(args))
	copy(merchantArgs, args)

	merchantRows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT 
			r.merchant, 
			COALESCE(SUM(r.total), 0) as amount,
			COALESCE(SUM(r.total) / NULLIF((SELECT SUM(total) FROM receipts r %s), 0) * 100, 0) as percentage
		FROM receipts r
		%s
		GROUP BY r.merchant
		ORDER BY amount DESC
		LIMIT 5
	`, whereClause, whereClause), merchantArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to get top merchants: %w", err)
	}
	defer merchantRows.Close()

	for merchantRows.Next() {
		var merchant domain.MerchantSummary
		if err := merchantRows.Scan(&merchant.Merchant, &merchant.Amount, &merchant.Percentage); err != nil {
			return nil, fmt.Errorf("failed to scan merchant: %w", err)
		}
		summary.TopMerchants = append(summary.TopMerchants, merchant)
	}

	if err := merchantRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating merchants: %w", err)
	}

	return summary, nil
}

// GetSpendingTrends retrieves spending trends over time
func (r *PostgresReceiptRepository) GetSpendingTrends(ctx context.Context, userID string, period string, startDateStr, endDateStr *string) (*domain.SpendingTrends, error) {
	// Create the result object
	trends := &domain.SpendingTrends{
		Period: period,
		Data:   []domain.SpendingTrendDataItem{},
	}

	// Validate period
	validPeriods := map[string]bool{
		"daily":   true,
		"weekly":  true,
		"monthly": true,
		"yearly":  true,
	}
	if !validPeriods[period] {
		return nil, fmt.Errorf("invalid period: %s", period)
	}

	// Build WHERE clause with user ID and date strings
	conditions := []string{}
	if userID != "" {
		conditions = append(conditions, fmt.Sprintf("user_id = '%s'", userID))
	}
	if startDateStr != nil {
		conditions = append(conditions, fmt.Sprintf("date >= '%s'::date", *startDateStr))
	}
	if endDateStr != nil {
		conditions = append(conditions, fmt.Sprintf("date <= '%s'::date", *endDateStr))
	}
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Use different queries based on period to avoid TO_CHAR conversion issues
	var query string
	switch period {
	case "daily":
		query = fmt.Sprintf(`
			SELECT 
				TO_CHAR(date, 'YYYY-MM-DD') as date,
				COALESCE(SUM(total), 0) as amount
			FROM receipts
			%s
			GROUP BY date
			ORDER BY date
		`, whereClause)
	case "weekly":
		query = fmt.Sprintf(`
			SELECT 
				TO_CHAR(date, 'YYYY-"W"IW') as date,
				COALESCE(SUM(total), 0) as amount
			FROM receipts
			%s
			GROUP BY TO_CHAR(date, 'YYYY-"W"IW'), date
			ORDER BY MIN(date)
		`, whereClause)
	case "monthly":
		query = fmt.Sprintf(`
			SELECT 
				TO_CHAR(date, 'YYYY-MM') as date,
				COALESCE(SUM(total), 0) as amount
			FROM receipts
			%s
			GROUP BY TO_CHAR(date, 'YYYY-MM'), date
			ORDER BY MIN(date)
		`, whereClause)
	case "yearly":
		query = fmt.Sprintf(`
			SELECT 
				TO_CHAR(date, 'YYYY') as date,
				COALESCE(SUM(total), 0) as amount
			FROM receipts
			%s
			GROUP BY TO_CHAR(date, 'YYYY'), date
			ORDER BY MIN(date)
		`, whereClause)
	}

	// Execute the query
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query spending trends: %w", err)
	}
	defer rows.Close()

	// Process results
	for rows.Next() {
		var item domain.SpendingTrendDataItem
		if err := rows.Scan(&item.Date, &item.Amount); err != nil {
			return nil, fmt.Errorf("failed to scan spending trend: %w", err)
		}
		trends.Data = append(trends.Data, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating spending trends: %w", err)
	}

	return trends, nil
}

// GetSpendingByCategory retrieves spending breakdown by category
func (r *PostgresReceiptRepository) GetSpendingByCategory(ctx context.Context, userID string, startDateStr, endDateStr *string) (*domain.CategorySpending, error) {
	// Initialize result
	result := &domain.CategorySpending{
		Total:      0,
		Categories: []domain.CategorySpendingItem{},
	}

	// Build WHERE clause with user ID and date strings
	conditions := []string{}
	if userID != "" {
		conditions = append(conditions, fmt.Sprintf("user_id = '%s'", userID))
	}
	if startDateStr != nil {
		conditions = append(conditions, fmt.Sprintf("date >= '%s'::date", *startDateStr))
	}
	if endDateStr != nil {
		conditions = append(conditions, fmt.Sprintf("date <= '%s'::date", *endDateStr))
	}
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Build receipt WHERE clause for joining with receipt_items
	receiptConditions := []string{}
	if userID != "" {
		receiptConditions = append(receiptConditions, fmt.Sprintf("r.user_id = '%s'", userID))
	}
	if startDateStr != nil {
		receiptConditions = append(receiptConditions, fmt.Sprintf("r.date >= '%s'::date", *startDateStr))
	}
	if endDateStr != nil {
		receiptConditions = append(receiptConditions, fmt.Sprintf("r.date <= '%s'::date", *endDateStr))
	}
	receiptWhereClause := ""
	if len(receiptConditions) > 0 {
		receiptWhereClause = "WHERE " + strings.Join(receiptConditions, " AND ")
	}

	// Get total spending
	totalQuery := fmt.Sprintf(`
		SELECT COALESCE(SUM(total), 0) 
		FROM receipts
		%s
	`, whereClause)

	err := r.db.QueryRow(ctx, totalQuery).Scan(&result.Total)
	if err != nil {
		return nil, fmt.Errorf("failed to get total spending: %w", err)
	}

	// Get spending by category
	categoryQuery := fmt.Sprintf(`
		SELECT 
			COALESCE(ri.category, 'Uncategorized') as name, 
			COALESCE(SUM(ri.qty * ri.price), 0) as amount
		FROM receipt_items ri
		JOIN receipts r ON ri.receipt_id = r.id
		%s
		GROUP BY ri.category
		ORDER BY amount DESC
	`, receiptWhereClause)

	categoryRows, err := r.db.Query(ctx, categoryQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query spending by category: %w", err)
	}
	defer categoryRows.Close()

	// Map to store category indices for later updating
	categoryIndices := make(map[string]int)

	// Process categories
	for categoryRows.Next() {
		var category domain.CategorySpendingItem
		if err := categoryRows.Scan(&category.Name, &category.Amount); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}

		// Calculate percentage
		if result.Total > 0 {
			category.Percentage = (category.Amount / result.Total) * 100
		} else {
			category.Percentage = 0
		}

		// Initialize items slice
		category.Items = []domain.CategorySpendingItemDetail{}

		// Store the index for later updating
		categoryIndices[category.Name] = len(result.Categories)

		// Add to results
		result.Categories = append(result.Categories, category)
	}

	if err := categoryRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating categories: %w", err)
	}

	// For each category, get top items
	for _, category := range result.Categories {
		itemQuery := ""
		if receiptWhereClause == "" {
			itemQuery = fmt.Sprintf(`
				SELECT 
					ri.name, 
					COALESCE(SUM(ri.qty * ri.price), 0) as total_spent, 
					COUNT(*) as count
				FROM receipt_items ri
				JOIN receipts r ON ri.receipt_id = r.id
				WHERE ri.category = '%s' OR (ri.category IS NULL AND '%s' = 'Uncategorized')
				GROUP BY ri.name
				ORDER BY total_spent DESC
				LIMIT 10
			`, category.Name, category.Name)
		} else {
			itemQuery = fmt.Sprintf(`
				SELECT 
					ri.name, 
					COALESCE(SUM(ri.qty * ri.price), 0) as total_spent, 
					COUNT(*) as count
				FROM receipt_items ri
				JOIN receipts r ON ri.receipt_id = r.id
				%s AND (ri.category = '%s' OR (ri.category IS NULL AND '%s' = 'Uncategorized'))
				GROUP BY ri.name
				ORDER BY total_spent DESC
				LIMIT 10
			`, receiptWhereClause, category.Name, category.Name)
		}

		itemRows, err := r.db.Query(ctx, itemQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to query category items: %w", err)
		}

		var items []domain.CategorySpendingItemDetail
		for itemRows.Next() {
			var item domain.CategorySpendingItemDetail
			if err := itemRows.Scan(&item.Name, &item.TotalSpent, &item.Count); err != nil {
				itemRows.Close()
				return nil, fmt.Errorf("failed to scan category item: %w", err)
			}
			items = append(items, item)
		}
		itemRows.Close()

		// Update the category with items
		if idx, ok := categoryIndices[category.Name]; ok {
			result.Categories[idx].Items = items
		}
	}

	return result, nil
}

// GetMerchantFrequency retrieves data on frequently visited merchants
func (r *PostgresReceiptRepository) GetMerchantFrequency(ctx context.Context, userID string, startDateStr, endDateStr *string, limit int) (*domain.MerchantFrequency, error) {
	// Validate limit
	if limit <= 0 {
		limit = 10 // Default
	}
	if limit > 50 {
		limit = 50 // Max
	}

	// Initialize result
	result := &domain.MerchantFrequency{
		TotalVisits: 0,
		Merchants:   []domain.MerchantFrequencyDetail{},
	}

	// Build WHERE clause with user ID and date strings
	conditions := []string{}
	if userID != "" {
		conditions = append(conditions, fmt.Sprintf("user_id = '%s'", userID))
	}
	if startDateStr != nil {
		conditions = append(conditions, fmt.Sprintf("date >= '%s'::date", *startDateStr))
	}
	if endDateStr != nil {
		conditions = append(conditions, fmt.Sprintf("date <= '%s'::date", *endDateStr))
	}
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total visit count
	visitQuery := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM receipts
		%s
	`, whereClause)

	err := r.db.QueryRow(ctx, visitQuery).Scan(&result.TotalVisits)
	if err != nil {
		return nil, fmt.Errorf("failed to get total visits: %w", err)
	}

	// Get merchant frequency with limit
	merchantQuery := fmt.Sprintf(`
		SELECT 
			COALESCE(merchant, 'Unknown') as name,
			COUNT(*) as visits,
			COALESCE(SUM(total), 0) as total_spent,
			COALESCE(AVG(total), 0) as average_spent
		FROM receipts
		%s
		GROUP BY merchant
		ORDER BY visits DESC, total_spent DESC
		LIMIT %d
	`, whereClause, limit)

	// Execute the query
	rows, err := r.db.Query(ctx, merchantQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query merchant frequency: %w", err)
	}
	defer rows.Close()

	// Process results
	for rows.Next() {
		var merchant domain.MerchantFrequencyDetail
		if err := rows.Scan(
			&merchant.Name,
			&merchant.Visits,
			&merchant.TotalSpent,
			&merchant.AverageSpent,
		); err != nil {
			return nil, fmt.Errorf("failed to scan merchant: %w", err)
		}

		// Calculate percentage
		if result.TotalVisits > 0 {
			merchant.Percentage = float64(merchant.Visits) / float64(result.TotalVisits) * 100
		} else {
			merchant.Percentage = 0
		}

		result.Merchants = append(result.Merchants, merchant)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating merchants: %w", err)
	}

	return result, nil
}

// GetMonthlyComparison compares spending between two months
func (r *PostgresReceiptRepository) GetMonthlyComparison(ctx context.Context, userID string, month1, month2 string) (*domain.MonthlyComparison, error) {
	// Validate month format (YYYY-MM)
	for _, month := range []string{month1, month2} {
		if _, err := time.Parse("2006-01", month); err != nil {
			return nil, fmt.Errorf("invalid month format %s: %w", month, err)
		}
	}

	// Initialize result
	result := &domain.MonthlyComparison{
		Month1:     month1,
		Month2:     month2,
		Categories: []domain.MonthlyCategoryComparison{},
	}

	// Get total spending for month1
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(total), 0)
		FROM receipts
		WHERE TO_CHAR(date, 'YYYY-MM') = $1 AND user_id = $2
	`, month1, userID).Scan(&result.Month1Total)
	if err != nil {
		return nil, fmt.Errorf("failed to get month1 total: %w", err)
	}

	// Get total spending for month2
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(total), 0)
		FROM receipts
		WHERE TO_CHAR(date, 'YYYY-MM') = $1 AND user_id = $2
	`, month2, userID).Scan(&result.Month2Total)
	if err != nil {
		return nil, fmt.Errorf("failed to get month2 total: %w", err)
	}

	// Calculate absolute difference
	result.Difference = result.Month2Total - result.Month1Total

	// Calculate percentage change
	if result.Month1Total > 0 {
		result.PercentageChange = (result.Difference / result.Month1Total) * 100
	} else if result.Month2Total > 0 {
		result.PercentageChange = 100 // If month1 is zero and month2 is positive, 100% increase
	}

	// Get category comparison
	rows, err := r.db.Query(ctx, `
		WITH month1_categories AS (
			SELECT
				ri.category,
				COALESCE(SUM(ri.qty * ri.price), 0) as amount
			FROM receipt_items ri
			JOIN receipts r ON ri.receipt_id = r.id
			WHERE TO_CHAR(r.date, 'YYYY-MM') = $1 AND r.user_id = $3
			GROUP BY ri.category
			HAVING ri.category IS NOT NULL
		),
		month2_categories AS (
			SELECT
				ri.category,
				COALESCE(SUM(ri.qty * ri.price), 0) as amount
			FROM receipt_items ri
			JOIN receipts r ON ri.receipt_id = r.id
			WHERE TO_CHAR(r.date, 'YYYY-MM') = $2 AND r.user_id = $3
			GROUP BY ri.category
			HAVING ri.category IS NOT NULL
		),
		all_categories AS (
			SELECT DISTINCT category FROM (
				SELECT category FROM month1_categories
				UNION
				SELECT category FROM month2_categories
			) as combined
		)
		SELECT
			ac.category,
			COALESCE(m1.amount, 0) as month1_amount,
			COALESCE(m2.amount, 0) as month2_amount
		FROM all_categories ac
		LEFT JOIN month1_categories m1 ON ac.category = m1.category
		LEFT JOIN month2_categories m2 ON ac.category = m2.category
		ORDER BY GREATEST(COALESCE(m1.amount, 0), COALESCE(m2.amount, 0)) DESC
	`, month1, month2, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query category comparison: %w", err)
	}
	defer rows.Close()

	// Parse category results
	for rows.Next() {
		var category domain.MonthlyCategoryComparison
		if err := rows.Scan(&category.Name, &category.Month1Amount, &category.Month2Amount); err != nil {
			return nil, fmt.Errorf("failed to scan category comparison: %w", err)
		}

		// Calculate difference and percentage change
		category.Difference = category.Month2Amount - category.Month1Amount

		if category.Month1Amount > 0 {
			category.PercentageChange = (category.Difference / category.Month1Amount) * 100
		} else if category.Month2Amount > 0 {
			category.PercentageChange = 100 // If month1 is zero and month2 is positive, 100% increase
		}

		result.Categories = append(result.Categories, category)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating category comparison: %w", err)
	}

	return result, nil
}
