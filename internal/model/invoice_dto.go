package model

import (
	"time"

	"github.com/ridwanfathin/invoice-processor-service/internal/domain"
)

// LineItemDTO represents a single item in an invoice for data transfer
type LineItemDTO struct {
	Description string   `json:"description"`
	Details     []string `json:"details"`
	Quantity    float64  `json:"quantity"`
	UnitPrice   float64  `json:"unit_price"`
	Total       float64  `json:"total"`
	Category    string   `json:"category"`
}

// InvoiceDTO represents the structured data extracted from an invoice image for data transfer
type InvoiceDTO struct {
	VendorName     string        `json:"vendor_name"`
	InvoiceNumber  string        `json:"invoice_number"`
	InvoiceDate    string        `json:"invoice_date"` // Format: YYYY-MM-DD
	DueDate        string        `json:"due_date"`     // Format: YYYY-MM-DD
	Items          []LineItemDTO `json:"items"`
	Subtotal       float64       `json:"subtotal"`
	TaxRatePercent float64       `json:"tax_rate_percent"`
	TaxAmount      float64       `json:"tax_amount"`
	Discount       float64       `json:"discount"`
	TotalDue       float64       `json:"total_due"`
}

// InvoiceProcessingRequest represents an incoming invoice processing request
type InvoiceProcessingRequest struct {
	File []byte
}

// InvoiceProcessingResponse represents the response to an invoice processing request
type InvoiceProcessingResponse struct {
	Invoice *InvoiceDTO `json:"invoice"`
	Error   string      `json:"error,omitempty"`
}

// InvoiceSuccessResponse represents a successful invoice processing response
type InvoiceSuccessResponse struct {
	Success bool        `json:"success"`
	Invoice *InvoiceDTO `json:"invoice"`
}

// InvoiceErrorResponse represents an error response from invoice processing
type InvoiceErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// FromDomain converts a domain Invoice to an InvoiceDTO
func (dto *InvoiceDTO) FromDomain(invoice *domain.Invoice) {
	dto.VendorName = invoice.VendorName
	dto.InvoiceNumber = invoice.InvoiceNumber
	dto.InvoiceDate = invoice.InvoiceDate.Format("2006-01-02")
	dto.DueDate = invoice.DueDate.Format("2006-01-02")
	dto.Subtotal = invoice.Subtotal
	dto.TaxRatePercent = invoice.TaxRatePercent
	dto.TaxAmount = invoice.TaxAmount
	dto.Discount = invoice.Discount
	dto.TotalDue = invoice.TotalDue

	// Convert line items
	dto.Items = make([]LineItemDTO, len(invoice.Items))
	for i, item := range invoice.Items {
		dto.Items[i] = LineItemDTO{
			Description: item.Description,
			Details:     item.Details,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			Total:       item.Total,
			Category:    item.Category,
		}
	}
}

// ToDomain converts an InvoiceDTO to a domain Invoice
func (dto *InvoiceDTO) ToDomain() (*domain.Invoice, error) {
	invoice := domain.NewInvoice()
	invoice.VendorName = dto.VendorName
	invoice.InvoiceNumber = dto.InvoiceNumber

	// Parse dates
	if dto.InvoiceDate != "" {
		invoiceDate, err := time.Parse("2006-01-02", dto.InvoiceDate)
		if err != nil {
			return nil, err
		}
		invoice.InvoiceDate = invoiceDate
	}

	if dto.DueDate != "" {
		dueDate, err := time.Parse("2006-01-02", dto.DueDate)
		if err != nil {
			return nil, err
		}
		invoice.DueDate = dueDate
	}

	invoice.Subtotal = dto.Subtotal
	invoice.TaxRatePercent = dto.TaxRatePercent
	invoice.TaxAmount = dto.TaxAmount
	invoice.Discount = dto.Discount
	invoice.TotalDue = dto.TotalDue

	// Convert line items
	invoice.Items = make([]domain.LineItem, len(dto.Items))
	for i, item := range dto.Items {
		invoice.Items[i] = domain.LineItem{
			Description: item.Description,
			Details:     item.Details,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			Total:       item.Total,
			Category:    item.Category,
		}
	}

	return invoice, nil
}
