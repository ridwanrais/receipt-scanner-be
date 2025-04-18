FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod ./
# Copy go.sum if it exists
COPY go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o invoice-ocr-service ./cmd/server

# Use a smaller image for the final container
FROM alpine:3.19

# Install Tesseract OCR and required dependencies
RUN apk add --no-cache tesseract-ocr tesseract-ocr-data-eng tesseract-ocr-data-ind ca-certificates

# Create a non-root user
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/invoice-ocr-service .

# Use non-root user
USER appuser

# Expose the application port
EXPOSE 8080

# Set environment variables
ENV PORT=8080
ENV MAX_WORKERS=5
ENV OCR_LANGUAGES=eng,ind

# Run the application
CMD ["./invoice-ocr-service"]
