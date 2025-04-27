package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ridwanfathin/invoice-processor-service/internal/domain"
)

// GetDashboardSummary retrieves summary data for the dashboard
func (r *PostgresReceiptRepository) GetDashboardSummary(ctx context.Context, startDateStr, endDateStr *string) (*domain.DashboardSummary, error) {
	// Parse date strings if provided
	var startDate, endDate *time.Time
	var err error
	
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

	if startDate != nil {
		conditions = append(conditions, fmt.Sprintf("r.date >= $%d", argCount))
		args = append(args, startDate)
		argCount++
	}
	if endDate != nil {
		conditions = append(conditions, fmt.Sprintf("r.date <= $%d", argCount))
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
	err = r.db.QueryRow(ctx, fmt.Sprintf(`
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
func (r *PostgresReceiptRepository) GetSpendingTrends(ctx context.Context, period string, startDateStr, endDateStr *string) (*domain.SpendingTrends, error) {
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

	// Validate period
	if period == "" {
		period = "monthly" // Default
	}
	
	validPeriods := map[string]bool{
		"daily":   true,
		"weekly":  true,
		"monthly": true,
		"yearly":  true,
	}
	
	if !validPeriods[period] {
		return nil, fmt.Errorf("invalid period: %s", period)
	}

	// Build query conditions for date filtering
	conditions := []string{}
	args := []interface{}{}
	argCount := 1

	if startDate != nil {
		conditions = append(conditions, fmt.Sprintf("date >= $%d", argCount))
		args = append(args, startDate)
		argCount++
	}
	if endDate != nil {
		conditions = append(conditions, fmt.Sprintf("date <= $%d", argCount))
		args = append(args, endDate)
		argCount++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Define date grouping based on period
	var dateGrouping string
	switch period {
	case "daily":
		dateGrouping = "DATE(date)"
	case "weekly":
		dateGrouping = "TO_CHAR(date, 'YYYY-WW')"
	case "monthly":
		dateGrouping = "TO_CHAR(date, 'YYYY-MM')"
	case "yearly":
		dateGrouping = "TO_CHAR(date, 'YYYY')"
	}

	// Query spending trends
	query := fmt.Sprintf(`
		SELECT
			%s as period_date,
			COALESCE(SUM(total), 0) as amount
		FROM receipts
		%s
		GROUP BY period_date
		ORDER BY period_date
	`, dateGrouping, whereClause)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query spending trends: %w", err)
	}
	defer rows.Close()

	// Parse results
	trends := &domain.SpendingTrends{
		Period: period,
		Data:   []domain.SpendingTrendDataItem{},
	}

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
func (r *PostgresReceiptRepository) GetSpendingByCategory(ctx context.Context, startDateStr, endDateStr *string) (*domain.CategorySpending, error) {
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

	if startDate != nil {
		conditions = append(conditions, fmt.Sprintf("r.date >= $%d", argCount))
		args = append(args, startDate)
		argCount++
	}
	if endDate != nil {
		conditions = append(conditions, fmt.Sprintf("r.date <= $%d", argCount))
		args = append(args, endDate)
		argCount++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total spending
	var totalSpend float64
	err := r.db.QueryRow(ctx, fmt.Sprintf(`
		SELECT COALESCE(SUM(total), 0) FROM receipts r %s
	`, whereClause), args...).Scan(&totalSpend)
	if err != nil {
		return nil, fmt.Errorf("failed to get total spending: %w", err)
	}

	// Initialize result
	result := &domain.CategorySpending{
		Total:      totalSpend,
		Categories: []domain.CategorySpendingItem{},
	}

	// Get spending by category
	categoryArgs := make([]interface{}, len(args))
	copy(categoryArgs, args)
	
	categoryRows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT
			ri.category,
			COALESCE(SUM(ri.qty * ri.price), 0) as amount,
			COALESCE(SUM(ri.qty * ri.price) / NULLIF($%d, 0) * 100, 0) as percentage
		FROM receipt_items ri
		JOIN receipts r ON ri.receipt_id = r.id
		%s
		GROUP BY ri.category
		HAVING ri.category IS NOT NULL
		ORDER BY amount DESC
	`, argCount+1, whereClause), append(categoryArgs, totalSpend)...)
	if err != nil {
		return nil, fmt.Errorf("failed to query spending by category: %w", err)
	}
	defer categoryRows.Close()

	// Map to keep track of categories for later item detail queries
	categoryMap := make(map[string]*domain.CategorySpendingItem)

	// Parse category results
	for categoryRows.Next() {
		var category domain.CategorySpendingItem
		if err := categoryRows.Scan(&category.Name, &category.Amount, &category.Percentage); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		category.Items = []domain.CategorySpendingItemDetail{}
		categoryMap[category.Name] = &category
		result.Categories = append(result.Categories, category)
	}

	if err := categoryRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating categories: %w", err)
	}

	// For each category, get the top items
	for _, category := range result.Categories {
		itemArgs := make([]interface{}, len(args))
		copy(itemArgs, args)
		itemArgs = append(itemArgs, category.Name)

		itemQuery := fmt.Sprintf(`
			SELECT
				ri.name,
				COALESCE(SUM(ri.qty * ri.price), 0) as total_spent,
				COUNT(*) as count
			FROM receipt_items ri
			JOIN receipts r ON ri.receipt_id = r.id
			%s
			AND ri.category = $%d
			GROUP BY ri.name
			ORDER BY total_spent DESC
			LIMIT 5
		`, whereClause, argCount+1)

		itemRows, err := r.db.Query(ctx, itemQuery, itemArgs...)
		if err != nil {
			return nil, fmt.Errorf("failed to query category items: %w", err)
		}

		for itemRows.Next() {
			var item domain.CategorySpendingItemDetail
			if err := itemRows.Scan(&item.Name, &item.TotalSpent, &item.Count); err != nil {
				itemRows.Close()
				return nil, fmt.Errorf("failed to scan category item: %w", err)
			}
			if cat, ok := categoryMap[category.Name]; ok {
				cat.Items = append(cat.Items, item)
			}
		}

		itemRows.Close()
		if err := itemRows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating category items: %w", err)
		}
	}

	// Update result with the detailed categories
	for i, cat := range result.Categories {
		if detailedCat, ok := categoryMap[cat.Name]; ok {
			result.Categories[i].Items = detailedCat.Items
		}
	}

	return result, nil
}

// GetMerchantFrequency retrieves data on frequently visited merchants
func (r *PostgresReceiptRepository) GetMerchantFrequency(ctx context.Context, startDateStr, endDateStr *string, limit int) (*domain.MerchantFrequency, error) {
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

	// Validate limit
	if limit <= 0 {
		limit = 10 // Default
	}
	if limit > 50 {
		limit = 50 // Max
	}

	// Build query conditions for date filtering
	conditions := []string{}
	args := []interface{}{}
	argCount := 1

	if startDate != nil {
		conditions = append(conditions, fmt.Sprintf("date >= $%d", argCount))
		args = append(args, startDate)
		argCount++
	}
	if endDate != nil {
		conditions = append(conditions, fmt.Sprintf("date <= $%d", argCount))
		args = append(args, endDate)
		argCount++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total visits
	var totalVisits int
	err := r.db.QueryRow(ctx, fmt.Sprintf(`
		SELECT COUNT(*) FROM receipts %s
	`, whereClause), args...).Scan(&totalVisits)
	if err != nil {
		return nil, fmt.Errorf("failed to get total visits: %w", err)
	}

	// Initialize result
	result := &domain.MerchantFrequency{
		TotalVisits: totalVisits,
		Merchants:   []domain.MerchantFrequencyDetail{},
	}

	// Get merchant frequency
	args = append(args, limit)
	
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT
			merchant,
			COUNT(*) as visits,
			COALESCE(SUM(total), 0) as total_spent,
			COALESCE(SUM(total) / COUNT(*), 0) as average_spent,
			COALESCE(COUNT(*) / NULLIF($%d, 0)::float * 100, 0) as percentage
		FROM receipts
		%s
		GROUP BY merchant
		ORDER BY visits DESC
		LIMIT $%d
	`, argCount+1, whereClause, argCount+2), append(args, totalVisits)...)
	if err != nil {
		return nil, fmt.Errorf("failed to query merchant frequency: %w", err)
	}
	defer rows.Close()

	// Parse results
	for rows.Next() {
		var merchant domain.MerchantFrequencyDetail
		if err := rows.Scan(
			&merchant.Name,
			&merchant.Visits,
			&merchant.TotalSpent,
			&merchant.AverageSpent,
			&merchant.Percentage,
		); err != nil {
			return nil, fmt.Errorf("failed to scan merchant: %w", err)
		}
		result.Merchants = append(result.Merchants, merchant)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating merchants: %w", err)
	}

	return result, nil
}

// GetMonthlyComparison compares spending between two months
func (r *PostgresReceiptRepository) GetMonthlyComparison(ctx context.Context, month1, month2 string) (*domain.MonthlyComparison, error) {
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
		WHERE TO_CHAR(date, 'YYYY-MM') = $1
	`, month1).Scan(&result.Month1Total)
	if err != nil {
		return nil, fmt.Errorf("failed to get month1 total: %w", err)
	}

	// Get total spending for month2
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(total), 0)
		FROM receipts
		WHERE TO_CHAR(date, 'YYYY-MM') = $1
	`, month2).Scan(&result.Month2Total)
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
			WHERE TO_CHAR(r.date, 'YYYY-MM') = $1
			GROUP BY ri.category
			HAVING ri.category IS NOT NULL
		),
		month2_categories AS (
			SELECT
				ri.category,
				COALESCE(SUM(ri.qty * ri.price), 0) as amount
			FROM receipt_items ri
			JOIN receipts r ON ri.receipt_id = r.id
			WHERE TO_CHAR(r.date, 'YYYY-MM') = $2
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
	`, month1, month2)
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
