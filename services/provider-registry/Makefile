# Makefile for Provider Registry Service

BINARY_NAME=provider-registry
CMD_DIR=cmd/server
BUILD_DIR=build
GO_FILES=$(shell find . -type f -name '*.go')

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build flags
BUILD_FLAGS=-v
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# Version and build info
VERSION?=1.0.0
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S_UTC')

.PHONY: all build clean test coverage deps tidy fmt vet lint run dev

all: clean deps tidy fmt vet test build

build: $(BUILD_DIR)/$(BINARY_NAME)

$(BUILD_DIR)/$(BINARY_NAME): $(GO_FILES)
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)

clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)

test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

deps:
	@echo "Downloading dependencies..."
	$(GOGET) ./...

tidy:
	@echo "Tidying modules..."
	$(GOMOD) tidy

fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

lint:
	@echo "Running golint..."
	@which golint > /dev/null || $(GOGET) golang.org/x/lint/golint
	golint ./...

run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

dev: build
	@echo "Running $(BINARY_NAME) in development mode..."
	GIN_MODE=debug LOG_LEVEL=debug ./$(BUILD_DIR)/$(BINARY_NAME) -config config/config.yaml

docker-build:
	@echo "Building Docker image..."
	docker build -t depin/provider-registry:$(VERSION) .

docker-run: docker-build
	@echo "Running Docker container..."
	docker run -p 8080:8080 --env-file .env depin/provider-registry:$(VERSION)

# Development helpers
watch:
	@echo "Watching for changes..."
	@which air > /dev/null || $(GOGET) github.com/cosmtrek/air
	air

install-tools:
	@echo "Installing development tools..."
	$(GOGET) golang.org/x/lint/golint
	$(GOGET) github.com/cosmtrek/air
	$(GOGET) github.com/stretchr/testify

# API documentation
docs:
	@echo "Generating API documentation..."
	@which swagger > /dev/null || $(GOGET) github.com/go-swagger/go-swagger/cmd/swagger
	swagger generate spec -o api/swagger.json --scan-models

serve-docs: docs
	@echo "Serving API documentation..."
	@which swagger > /dev/null || $(GOGET) github.com/go-swagger/go-swagger/cmd/swagger
	swagger serve -F swagger api/swagger.json

help:
	@echo "Available targets:"
	@echo "  build       - Build the binary"
	@echo "  clean       - Clean build artifacts"
	@echo "  test        - Run tests"
	@echo "  coverage    - Run tests with coverage"
	@echo "  deps        - Download dependencies"
	@echo "  tidy        - Tidy modules"
	@echo "  fmt         - Format code"
	@echo "  vet         - Run go vet"
	@echo "  lint        - Run golint"
	@echo "  run         - Build and run"
	@echo "  dev         - Run in development mode"
	@echo "  watch       - Watch for changes and rebuild"
	@echo "  docker-build- Build Docker image"
	@echo "  docker-run  - Build and run Docker container"
	@echo "  docs        - Generate API documentation"
	@echo "  serve-docs  - Serve API documentation"
	@echo "  help        - Show this help"