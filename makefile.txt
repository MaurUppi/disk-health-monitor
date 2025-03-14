# Makefile for disk-health-monitor

# Go build parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Build parameters
BINARY_NAME=disk-health-monitor
MAIN_PACKAGE=.  # 修改为指向根目录
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE=$(shell date -u '+%Y-%m-%d %H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X 'main.BuildDate=$(BUILD_DATE)'"

# 其余内容保持不变

# Output directories
BIN_DIR=bin
DIST_DIR=dist

.PHONY: all build clean test fmt lint vet tidy deps install dist help

# Default target
all: clean build test

# Build the application
build:
	@echo "Building binary..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Build complete: $(BIN_DIR)/$(BINARY_NAME)"

# Clean up artifacts
clean:
	@echo "Cleaning up..."
	@rm -rf $(BIN_DIR) $(DIST_DIR)
	@echo "Clean complete"

# Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...
	@echo "Tests complete"

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...
	@echo "Formatting complete"

# Check code quality
lint:
	@echo "Linting code..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi
	@echo "Linting complete"

# Run go vet
vet:
	@echo "Vetting code..."
	$(GOVET) ./...
	@echo "Vetting complete"

# Update dependencies
deps:
	@echo "Updating dependencies..."
	$(GOGET) -u ./...
	$(GOMOD) tidy
	@echo "Dependencies updated"

# Just run go mod tidy
tidy:
	@echo "Tidying go modules..."
	$(GOMOD) tidy
	@echo "Tidy complete"

# Install the application
install: build
	@echo "Installing binary..."
	install -m 755 $(BIN_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "Installation complete"

# Create distribution packages
dist: build
	@echo "Creating distribution packages..."
	@mkdir -p $(DIST_DIR)
	tar -czf $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz -C $(BIN_DIR) $(BINARY_NAME)
	@echo "Distribution packages created in $(DIST_DIR)/"

# Display help information
help:
	@echo "Disk Health Monitor - Make targets:"
	@echo "  build      - Build the application"
	@echo "  clean      - Remove build artifacts"
	@echo "  test       - Run tests"
	@echo "  fmt        - Format code"
	@echo "  lint       - Run linter"
	@echo "  vet        - Run go vet"
	@echo "  deps       - Update dependencies"
	@echo "  tidy       - Run go mod tidy"
	@echo "  install    - Install the application"
	@echo "  dist       - Create distribution packages"
	@echo "  help       - Display this help message"
