package repository

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ridwanfathin/invoice-processor-service/internal/domain"
)

// PostgresReceiptRepository implements ReceiptRepository interface using PostgreSQL
type PostgresReceiptRepository struct {
	db *pgxpool.Pool
}

// NewPostgresReceiptRepository creates a new PostgreSQL receipt repository
func NewPostgresReceiptRepository(db *pgxpool.Pool) *PostgresReceiptRepository {
	return &PostgresReceiptRepository{
		db: db,
	}
}

// CreateReceipt saves a new receipt to the database
func (r *PostgresReceiptRepository) CreateReceipt(ctx context.Context, receipt *domain.Receipt) (*domain.Receipt, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // Rollback if not committed

	// Insert receipt
	var receiptID string
	err = tx.QueryRow(ctx, `
		INSERT INTO receipts (merchant, date, total, tax, subtotal, image_url)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`, receipt.Merchant, receipt.Date, receipt.Total, receipt.Tax, receipt.Subtotal, receipt.ImageURL).Scan(
		&receiptID, &receipt.CreatedAt, &receipt.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert receipt: %w", err)
	}

	receipt.ID = receiptID

	// Insert receipt items
	for i := range receipt.Items {
		item := &receipt.Items[i]
		err = tx.QueryRow(ctx, `
			INSERT INTO receipt_items (receipt_id, name, qty, price, category)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, created_at, updated_at
		`, receiptID, item.Name, item.Quantity, item.Price, item.Category).Scan(
			&item.ID, &time.Time{}, &time.Time{}, // We don't need the timestamps for items, but they're returned
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert receipt item: %w", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return receipt, nil
}

// GetReceiptByID retrieves a receipt by its ID
func (r *PostgresReceiptRepository) GetReceiptByID(ctx context.Context, receiptID string) (*domain.Receipt, error) {
	// Query receipt
	var receipt domain.Receipt
	err := r.db.QueryRow(ctx, `
		SELECT id, merchant, date, total, tax, subtotal, image_url, created_at, updated_at
		FROM receipts
		WHERE id = $1
	`, receiptID).Scan(
		&receipt.ID, &receipt.Merchant, &receipt.Date, &receipt.Total, &receipt.Tax,
		&receipt.Subtotal, &receipt.ImageURL, &receipt.CreatedAt, &receipt.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("receipt not found: %s", receiptID)
		}
		return nil, fmt.Errorf("failed to get receipt: %w", err)
	}

	// Query receipt items
	rows, err := r.db.Query(ctx, `
		SELECT id, name, qty, price, category
		FROM receipt_items
		WHERE receipt_id = $1
		ORDER BY id
	`, receiptID)
	if err != nil {
		return nil, fmt.Errorf("failed to query receipt items: %w", err)
	}
	defer rows.Close()

	// Parse rows
	receipt.Items = []domain.ReceiptItem{}
	for rows.Next() {
		var item domain.ReceiptItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Quantity, &item.Price, &item.Category); err != nil {
			return nil, fmt.Errorf("failed to scan receipt item: %w", err)
		}
		receipt.Items = append(receipt.Items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating receipt items: %w", err)
	}

	return &receipt, nil
}

// UpdateReceipt updates an existing receipt
func (r *PostgresReceiptRepository) UpdateReceipt(ctx context.Context, receipt *domain.Receipt) (*domain.Receipt, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // Rollback if not committed

	// Update receipt
	var updatedAt time.Time
	err = tx.QueryRow(ctx, `
		UPDATE receipts
		SET merchant = $1, date = $2, total = $3, tax = $4, subtotal = $5, image_url = $6
		WHERE id = $7
		RETURNING updated_at
	`, receipt.Merchant, receipt.Date, receipt.Total, receipt.Tax, receipt.Subtotal, receipt.ImageURL, receipt.ID).Scan(&updatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to update receipt: %w", err)
	}
	
	receipt.UpdatedAt = updatedAt

	// Delete existing items
	_, err = tx.Exec(ctx, `DELETE FROM receipt_items WHERE receipt_id = $1`, receipt.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete receipt items: %w", err)
	}

	// Insert updated items
	for i := range receipt.Items {
		item := &receipt.Items[i]
		err = tx.QueryRow(ctx, `
			INSERT INTO receipt_items (receipt_id, name, qty, price, category)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`, receipt.ID, item.Name, item.Quantity, item.Price, item.Category).Scan(&item.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to insert receipt item: %w", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return receipt, nil
}

// DeleteReceipt deletes a receipt by its ID
func (r *PostgresReceiptRepository) DeleteReceipt(ctx context.Context, receiptID string) error {
	// Delete receipt (cascade will delete items)
	commandTag, err := r.db.Exec(ctx, `DELETE FROM receipts WHERE id = $1`, receiptID)
	if err != nil {
		return fmt.Errorf("failed to delete receipt: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("receipt not found: %s", receiptID)
	}

	return nil
}

// ListReceipts retrieves receipts with optional filters and pagination
func (r *PostgresReceiptRepository) ListReceipts(ctx context.Context, filter domain.ReceiptFilter) (*domain.PaginatedReceipts, error) {
	result := &domain.PaginatedReceipts{
		Data:       []domain.Receipt{},
		Pagination: domain.Pagination{},
	}

	// Set default pagination values if not provided
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 10
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	// Build query conditions
	conditions := []string{}
	args := []interface{}{}
	argCount := 1

	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("date >= $%d", argCount))
		args = append(args, filter.StartDate)
		argCount++
	}
	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("date <= $%d", argCount))
		args = append(args, filter.EndDate)
		argCount++
	}
	if filter.Merchant != "" {
		conditions = append(conditions, fmt.Sprintf("merchant ILIKE $%d", argCount))
		args = append(args, "%"+filter.Merchant+"%") // Case-insensitive partial match
		argCount++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total items
	var totalItems int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM receipts %s`, whereClause)
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&totalItems)
	if err != nil {
		return nil, fmt.Errorf("failed to count receipts: %w", err)
	}

	// Calculate pagination values
	result.Pagination.TotalItems = totalItems
	result.Pagination.Limit = filter.Limit
	result.Pagination.CurrentPage = filter.Page
	result.Pagination.TotalPages = int(math.Ceil(float64(totalItems) / float64(filter.Limit)))

	// If no results, return empty array
	if totalItems == 0 {
		return result, nil
	}

	// Calculate offset
	offset := (filter.Page - 1) * filter.Limit
	args = append(args, filter.Limit, offset)

	// Query receipts with pagination
	query := fmt.Sprintf(`
		SELECT id, merchant, date, total, tax, subtotal, image_url, created_at, updated_at
		FROM receipts
		%s
		ORDER BY date DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argCount, argCount+1)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query receipts: %w", err)
	}
	defer rows.Close()

	// Parse receipt rows
	receiptMap := make(map[string]*domain.Receipt)
	var receiptIDs []string

	for rows.Next() {
		var receipt domain.Receipt
		if err := rows.Scan(
			&receipt.ID, &receipt.Merchant, &receipt.Date, &receipt.Total, &receipt.Tax,
			&receipt.Subtotal, &receipt.ImageURL, &receipt.CreatedAt, &receipt.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan receipt: %w", err)
		}
		receipt.Items = []domain.ReceiptItem{}
		receiptMap[receipt.ID] = &receipt
		receiptIDs = append(receiptIDs, receipt.ID)
		result.Data = append(result.Data, receipt)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating receipts: %w", err)
	}

	// If no receipts, return empty result
	if len(receiptIDs) == 0 {
		return result, nil
	}

	// Get items for all receipts in a single query
	// This is more efficient than querying items for each receipt separately
	placeholders := make([]string, len(receiptIDs))
	itemArgs := make([]interface{}, len(receiptIDs))
	for i, id := range receiptIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		itemArgs[i] = id
	}

	itemQuery := fmt.Sprintf(`
		SELECT receipt_id, id, name, qty, price, category
		FROM receipt_items
		WHERE receipt_id IN (%s)
		ORDER BY id
	`, strings.Join(placeholders, ", "))

	itemRows, err := r.db.Query(ctx, itemQuery, itemArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query receipt items: %w", err)
	}
	defer itemRows.Close()

	// Populate items for each receipt
	for itemRows.Next() {
		var receiptID string
		var item domain.ReceiptItem
		if err := itemRows.Scan(
			&receiptID, &item.ID, &item.Name, &item.Quantity, &item.Price, &item.Category,
		); err != nil {
			return nil, fmt.Errorf("failed to scan receipt item: %w", err)
		}
		if receipt, ok := receiptMap[receiptID]; ok {
			receipt.Items = append(receipt.Items, item)
		}
	}

	if err := itemRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating receipt items: %w", err)
	}

	// Update the result data with the populated receipts
	for i, id := range receiptIDs {
		if receipt, ok := receiptMap[id]; ok {
			result.Data[i] = *receipt
		}
	}

	return result, nil
}

// GetReceiptItems retrieves all items from a specific receipt
func (r *PostgresReceiptRepository) GetReceiptItems(ctx context.Context, receiptID string) ([]domain.ReceiptItem, error) {
	// First, check if receipt exists
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM receipts WHERE id = $1)`, receiptID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check receipt existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("receipt not found: %s", receiptID)
	}

	// Query receipt items
	rows, err := r.db.Query(ctx, `
		SELECT id, name, qty, price, category
		FROM receipt_items
		WHERE receipt_id = $1
		ORDER BY id
	`, receiptID)
	if err != nil {
		return nil, fmt.Errorf("failed to query receipt items: %w", err)
	}
	defer rows.Close()

	// Parse rows
	items := []domain.ReceiptItem{}
	for rows.Next() {
		var item domain.ReceiptItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Quantity, &item.Price, &item.Category); err != nil {
			return nil, fmt.Errorf("failed to scan receipt item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating receipt items: %w", err)
	}

	return items, nil
}

// The remaining methods will be implemented in separate files for better organization
