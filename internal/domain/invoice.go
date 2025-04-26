package domain

import "time"

// LineItem represents a single item in an invoice
type LineItem struct {
	Description string
	Details     []string
	Quantity    float64
	UnitPrice   float64
	Total       float64
}

// Invoice represents the core domain entity for an invoice
type Invoice struct {
	VendorName     string
	InvoiceNumber  string
	InvoiceDate    time.Time
	DueDate        time.Time
	Items          []LineItem
	Subtotal       float64
	TaxRatePercent float64
	TaxAmount      float64
	Discount       float64
	TotalDue       float64
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
