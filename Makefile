VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY_NAME := pushover
BUILD_DIR := bin

.PHONY: build clean test lint fmt install help

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME) version $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) .

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)

## test: Run tests
test:
	@echo "Running tests..."
	go test -v ./...

## lint: Run linter
lint:
	@echo "Running linter..."
	golangci-lint run

## fmt: Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

## install: Install binary to /usr/local/bin
install: build
	@echo "Installing to /usr/local/bin..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

## run-example: Run example with test message (requires env vars)
run-example: build
	@echo "Running example..."
	./$(BUILD_DIR)/$(BINARY_NAME) -m "Test notification from improved CLI" -t "Test Title"

## help: Show this help
help:
	@echo "Available targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

# Default target
all: fmt test build
