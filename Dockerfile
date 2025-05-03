FROM golang:1.24.2-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o app-binary ./cmd/server

# Create a minimal image for distribution
FROM alpine:latest

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/app-binary .

# Set executable permissions
RUN chmod +x /app/app-binary

EXPOSE 8080

# Run the application
CMD ["/app/app-binary"]
