.PHONY: build run test lint clean docker-build docker-run

# Application name
APP_NAME = invoice-ocr-service
# Docker image name
IMAGE_NAME = invoice-ocr-service
# Docker image tag
IMAGE_TAG = latest

# Build the application
build:
	go build -o bin/$(APP_NAME) ./cmd/server

# Run the application
run: build
	./bin/$(APP_NAME)

# Run tests
test:
	go test -v ./...

# Run linter
lint:
	golangci-lint run

# Clean build artifacts
clean:
	rm -rf bin/

# Build Docker image
docker-build:
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

# Run Docker container
docker-run: docker-build
	docker run -p 8080:8080 $(IMAGE_NAME):$(IMAGE_TAG)

# Install dependencies
deps:
	go mod download

# Generate test coverage report
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
