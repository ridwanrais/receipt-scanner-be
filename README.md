# Invoice OCR Service

A Go-based RESTful API service that extracts structured data from invoice images using OCR technology.

## Features

- Extract vendor name, invoice number, date, line items, and totals from invoice images
- Support for JPEG, PNG, and PDF file formats
- Image preprocessing to improve OCR accuracy
- Concurrent processing with worker pool
- Configurable OCR language support (English and Indonesian)
- Docker support for easy deployment
- Comprehensive test suite

## Prerequisites

- Go 1.22 or higher
- Tesseract OCR 4.0 or higher
- Docker (optional, for containerized deployment)

## Installation

### Install Tesseract OCR

#### Ubuntu/Debian
```bash
sudo apt-get update
sudo apt-get install -y tesseract-ocr libtesseract-dev
# For Indonesian language support
sudo apt-get install -y tesseract-ocr-ind
```

#### macOS
```bash
brew install tesseract
# For Indonesian language support
brew install tesseract-lang
```

### Clone the repository
```bash
git clone https://github.com/ridwanfathin/invoice-ocr-service.git
cd invoice-ocr-service
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
go build -o bin/invoice-ocr-service ./cmd/server
./bin/invoice-ocr-service
```

### Run with Docker
```bash
make docker-build
make docker-run
```

Or using Docker commands directly:
```bash
docker build -t invoice-ocr-service .
docker run -p 8080:8080 invoice-ocr-service
```

## Configuration

The service can be configured using environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| PORT | HTTP server port | 8080 |
| MAX_WORKERS | Maximum number of concurrent OCR workers | 5 |
| OCR_LANGUAGES | Comma-separated list of Tesseract language codes | eng |

Example:
```bash
PORT=9000 MAX_WORKERS=10 OCR_LANGUAGES=eng,ind ./bin/invoice-ocr-service
```

## API Endpoints

### POST /api/v1/ocr/invoice

Process an invoice image and extract structured data.

**Request:**
- Content-Type: `multipart/form-data`
- Form field: `file` (JPEG, PNG, or PDF)

**Response:**
```json
{
  "vendor": "ACME Corporation",
  "invoice_number": "INV-12345",
  "invoice_date": "2025-04-18",
  "items": [
    {
      "description": "Product A",
      "quantity": 2,
      "unit_price": 10.99
    },
    {
      "description": "Service B",
      "quantity": 1,
      "unit_price": 50.00
    }
  ],
  "subtotal": 71.98,
  "tax": 7.20,
  "total": 79.18
}
```

### Example using curl

```bash
curl -X POST \
  http://localhost:8080/api/v1/ocr/invoice \
  -F "file=@/path/to/invoice.jpg"
```

## Customizing Tesseract OCR

### Page Segmentation Modes (PSM)

The default PSM is set to 3 (PSM_AUTO), which works well for most invoices. You can modify this in the `cmd/server/main.go` file:

```go
ocrConfig := &ocr.Config{
    Languages:   languages,
    PageSegMode: 6, // PSM_SINGLE_BLOCK
    MaxRetries:  3,
    RetryDelay:  time.Second,
}
```

Common PSM values:
- 1: Automatic page segmentation with OSD
- 3: Fully automatic page segmentation, but no OSD (default)
- 4: Assume a single column of text of variable sizes
- 6: Assume a single uniform block of text
- 11: Sparse text. Find as much text as possible in no particular order

### Adding New Languages

1. Install the Tesseract language pack for your language
2. Set the `OCR_LANGUAGES` environment variable to include your language code

Example for adding German:
```bash
# Install German language pack
sudo apt-get install tesseract-ocr-deu

# Run the service with German support
OCR_LANGUAGES=eng,deu ./bin/invoice-ocr-service
```

## Development

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
