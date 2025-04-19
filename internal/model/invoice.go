package model

// LineItem represents a single item in an invoice
type LineItem struct {
	Description string   `json:"description"`
	Details     []string `json:"details"`
	Quantity    int      `json:"quantity"`
	UnitPrice   float64  `json:"unit_price"`
	Total       float64  `json:"total"`
}

// Invoice represents the structured data extracted from an invoice image
type Invoice struct {
	VendorName     string     `json:"vendor_name"`
	InvoiceNumber  string     `json:"invoice_number"`
	InvoiceDate    string     `json:"invoice_date"` // Format: YYYY-MM-DD
	DueDate        string     `json:"due_date"`     // Format: YYYY-MM-DD
	Items          []LineItem `json:"items"`
	Subtotal       float64    `json:"subtotal"`
	TaxRatePercent float64    `json:"tax_rate_percent"`
	TaxAmount      float64    `json:"tax_amount"`
	Discount       float64    `json:"discount"`
	TotalDue       float64    `json:"total_due"`
}

// OCRRequest represents an incoming OCR request
type OCRRequest struct {
	File []byte
}

// OCRResponse represents the response to an OCR request
type OCRResponse struct {
	Invoice *Invoice `json:"invoice"`
	Error   string   `json:"error,omitempty"`
}
