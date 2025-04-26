# Invoice Processor Service

A Go-based RESTful API service that extracts structured data from invoice images using AI-powered processing.

## Features

- Extract vendor name, invoice number, date, line items, and totals from invoice images
- Support for JPEG, PNG, and PDF file formats
- AI-powered invoice processing using OpenRouter
- Concurrent processing with worker pool
- Configurable service settings
- Docker support for easy deployment
- Clean architecture design principles

## Prerequisites

- Go 1.22 or higher
- Docker (optional, for containerized deployment)

## Installation

### Clone the repository
```bash
git clone https://github.com/ridwanfathin/invoice-processor-service.git
cd invoice-processor-service
```

### Install Go dependencies
```bash
go mod download
```

## Usage

### Build and run locally
```bash
make build
make run
```

Or using Go commands directly:
```bash
go build -o bin/invoice-processor-service ./cmd/server
./bin/invoice-processor-service
```

### Run with Docker
```bash
make docker-build
make docker-run
```

Or using Docker commands directly:
```bash
docker build -t invoice-processor-service .
docker run -p 8080:8080 invoice-processor-service
```

## Configuration

The service can be configured using environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| PORT | HTTP server port | 8080 |
| MAX_WORKERS | Maximum number of concurrent processing workers | 5 |
| OPENROUTER_API_KEY | OpenRouter API key for AI processing | (required) |
| OPENROUTER_MODEL_ID | OpenRouter model ID to use | meta-llama/llama-3.2-11b-vision-instruct:free |
| OPENROUTER_TIMEOUT | Timeout for OpenRouter API calls in seconds | 60 |
| SUPABASE_URL | Supabase URL for image storage | (required) |
| SUPABASE_BUCKET | Supabase storage bucket name | invoices |
| SUPABASE_API_KEY | Supabase API key | (required) |

Example:
```bash
PORT=9000 MAX_WORKERS=10 OPENROUTER_API_KEY=your-key ./bin/invoice-processor-service
```

## API Endpoints

### POST /api/v1/invoices/process

Process an invoice image and extract structured data.

**Request:**
- Content-Type: `multipart/form-data`
- Form field: `file` (JPEG, PNG, or PDF)

**Response:**
```json
{
  "success": true,
  "invoice": {
    "vendor_name": "ACME Corporation",
    "invoice_number": "INV-12345",
    "invoice_date": "2025-04-18",
    "due_date": "2025-05-18",
    "items": [
      {
        "description": "Product A",
        "details": ["SKU-123", "Note: Premium edition"],
        "quantity": 2,
        "unit_price": 10.99,
        "total": 21.98
      },
      {
        "description": "Service B",
        "details": [],
        "quantity": 1,
        "unit_price": 50.00,
        "total": 50.00
      }
    ],
    "subtotal": 71.98,
    "tax_rate_percent": 10.0,
    "tax_amount": 7.20,
    "discount": 0.00,
    "total_due": 79.18
  }
}
```

### Example using curl

```bash
curl -X POST \
  http://localhost:8080/api/v1/invoices/process \
  -F "file=@/path/to/invoice.jpg"
```

## Development

### Hot Reload with Air

For development, you can use Air for hot reloading:

```bash
# Install Air first
go install github.com/cosmtrek/air@latest

# Run with Air
air
```

### Running Tests
```bash
make test
```

### Linting
```bash
make lint
```

### Test Coverage
```bash
make coverage
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.
