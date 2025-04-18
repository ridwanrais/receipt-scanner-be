package model

// LineItem represents a single item in an invoice
type LineItem struct {
	Description string  `json:"description"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
}

// Invoice represents the structured data extracted from an invoice image
type Invoice struct {
	Vendor        string     `json:"vendor"`
	InvoiceNumber string     `json:"invoice_number"`
	InvoiceDate   string     `json:"invoice_date"` // Format: YYYY-MM-DD
	Items         []LineItem `json:"items"`
	Subtotal      float64    `json:"subtotal"`
	Tax           float64    `json:"tax"`
	Total         float64    `json:"total"`
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
