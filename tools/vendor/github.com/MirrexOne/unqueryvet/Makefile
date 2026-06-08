.PHONY: all test build fmt fmt-check lint clean install help

# Default target
all: fmt test build

# Run tests
test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...

# Build the binary
build:
	@echo "Building unqueryvet..."
	@go build -v ./cmd/unqueryvet

# Format code with gofmt -s
fmt:
	@echo "Formatting code..."
	@find . -name "*.go" -not -path "./vendor/*" -exec gofmt -s -w {} +
	@go fmt ./...

# Check if code is formatted
fmt-check:
	@echo "Checking code formatting..."
	@if [ -n "$$(find . -name '*.go' -not -path './vendor/*' -exec gofmt -s -l {} +)" ]; then \
		echo "The following files need formatting:"; \
		find . -name '*.go' -not -path './vendor/*' -exec gofmt -s -l {} +; \
		exit 1; \
	else \
		echo "All files are properly formatted"; \
	fi

# Run linter
lint:
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null 2>&1; then \
		./lint-local.sh ./...; \
	else \
		echo "golangci-lint not installed. Install it from https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f unqueryvet
	@rm -f coverage.out
	@rm -f .golangci.local.yml
	@go clean

# Install the binary
install:
	@echo "Installing unqueryvet..."
	@go install ./cmd/unqueryvet

# Run unqueryvet on the project itself
check:
	@echo "Running unqueryvet on project..."
	@go run ./cmd/unqueryvet ./...

# Generate coverage report
coverage: test
	@echo "Generating coverage report..."
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./internal/analyzer

# Update dependencies
deps:
	@echo "Updating dependencies..."
	@go mod tidy
	@go mod verify

# Help target
help:
	@echo "Available targets:"
	@echo "  make         - Format, test, and build"
	@echo "  make test    - Run tests with race detection"
	@echo "  make build   - Build the unqueryvet binary"
	@echo "  make fmt     - Format all Go files with gofmt -s"
	@echo "  make fmt-check - Check if files are formatted"
	@echo "  make lint    - Run golangci-lint"
	@echo "  make clean   - Remove build artifacts"
	@echo "  make install - Install unqueryvet binary"
	@echo "  make check   - Run unqueryvet on the project"
	@echo "  make coverage - Generate coverage report"
	@echo "  make bench   - Run benchmarks"
	@echo "  make deps    - Update and verify dependencies"
	@echo "  make help    - Show this help message"
