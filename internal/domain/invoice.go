package domain

import (
	"encoding/json"
	"time"
)

// DateOnly is a custom type for handling date-only strings from JSON
type DateOnly struct {
	time.Time
}

// UnmarshalJSON implements custom unmarshaling for date-only strings
func (d *DateOnly) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	// Handle null/empty dates
	if s == "" || s == "null" {
		d.Time = time.Time{}
		return nil
	}

	// Parse date-only format
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return err
	}
	d.Time = t
	return nil
}

// MarshalJSON implements custom marshaling for date-only strings
func (d DateOnly) MarshalJSON() ([]byte, error) {
	if d.Time.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(d.Time.Format("2006-01-02"))
}

// LineItem represents a single item in an invoice
type LineItem struct {
	Description string   `json:"description"`
	Details     []string `json:"details,omitempty"`
	Quantity    float64  `json:"quantity"`
	UnitPrice   float64  `json:"unit_price"`
	Total       float64  `json:"total"`
	Currency    string   `json:"currency"` // Currency code (e.g., "IDR", "USD")
	Category    string   `json:"category"` // Added for LLM and Go mapping
}

// Invoice represents the core domain entity for an invoice
type Invoice struct {
	VendorName     string     `json:"vendor_name"`
	InvoiceNumber  string     `json:"invoice_number"`
	InvoiceDate    DateOnly   `json:"invoice_date"`
	DueDate        DateOnly   `json:"due_date"`
	Items          []LineItem `json:"items"`
	Subtotal       float64    `json:"subtotal"`
	TaxRatePercent float64    `json:"tax_rate_percent"`
	TaxAmount      float64    `json:"tax_amount"`
	Discount       float64    `json:"discount"`
	TotalDue       float64    `json:"total_due"`
}

// NewInvoice creates a new invoice with default values
func NewInvoice() *Invoice {
	return &Invoice{
		Items: make([]LineItem, 0),
	}
}

// AddLineItem adds a new line item to the invoice
func (i *Invoice) AddLineItem(item LineItem) {
	i.Items = append(i.Items, item)
}

// CalculateSubtotal recalculates the subtotal based on line items
func (i *Invoice) CalculateSubtotal() {
	var subtotal float64
	for _, item := range i.Items {
		subtotal += item.Total
	}
	i.Subtotal = subtotal
}

// CalculateTaxAmount calculates the tax amount based on the tax rate
func (i *Invoice) CalculateTaxAmount() {
	i.TaxAmount = i.Subtotal * (i.TaxRatePercent / 100)
}

// CalculateTotalDue calculates the total due amount
func (i *Invoice) CalculateTotalDue() {
	i.CalculateSubtotal()
	i.CalculateTaxAmount()
	i.TotalDue = i.Subtotal + i.TaxAmount - i.Discount
}
