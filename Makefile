# Makefile for invoice-processor-service

# Variables
APP_NAME=invoice-processor-service
CMD_DIR=./cmd/server
BIN_DIR=./bin
BIN_PATH=$(BIN_DIR)/$(APP_NAME)

.PHONY: all build run debug hotreload migrate test clean swagger

all: build

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_PATH) $(CMD_DIR)

run: build
	$(BIN_PATH)

# Hot reload with wgo and Delve for debugging (attach VSCode to port 2345)
hotreload-debug:
	wgo --cmd "dlv" -- debug ./cmd/server --headless --listen=:2345 --api-version=2 --accept-multiclient --log

hotreload:
	@echo "Starting hot reload with Swagger auto-regeneration..."
	@~/go/bin/swag init -g cmd/server/main.go -o docs
	~/go/bin/wgo -file=.go -xfile=_test.go -xdir=tmp,bin,vendor,.git go run cmd/server/main.go

# Run database migration script
migrate:
	go run scripts/run_migration.go

# Run tests
test:
	go test ./...

clean:
	rm -rf $(BIN_DIR) tmp

# Generate Swagger documentation
swagger:
	~/go/bin/swag init -g cmd/server/main.go -o docs
