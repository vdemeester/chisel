# Makefile for Chisel and Mallet
# Chisel: Dagger-based Tekton executor
# Mallet: Podman-based Tekton executor

.PHONY: all build build-chisel build-mallet test lint clean help

# Default target
all: build

# Build both binaries
build: build-chisel build-mallet

# Build chisel (Dagger backend)
build-chisel:
	@echo "Building chisel..."
	go build -o bin/chisel ./cmd/chisel

# Build mallet (Podman backend)
build-mallet:
	@echo "Building mallet..."
	go build -o bin/mallet ./cmd/mallet

# Run all tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# Install both binaries to GOPATH/bin
install: build
	@echo "Installing chisel and mallet..."
	go install ./cmd/chisel
	go install ./cmd/mallet

# Quick check: build and run dry-run
check: build
	@echo "Checking chisel..."
	./bin/chisel run examples/simple/hello-pipelinerun.yaml --dry-run
	@echo "Checking mallet..."
	./bin/mallet run examples/simple/hello-pipelinerun.yaml --dry-run

# Show help
help:
	@echo "Chisel & Mallet Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make              Build both chisel and mallet"
	@echo "  make build        Build both binaries"
	@echo "  make build-chisel Build chisel only"
	@echo "  make build-mallet Build mallet only"
	@echo "  make test         Run all tests"
	@echo "  make test-coverage Run tests with coverage report"
	@echo "  make lint         Run golangci-lint"
	@echo "  make clean        Remove build artifacts"
	@echo "  make install      Install binaries to GOPATH/bin"
	@echo "  make check        Build and run dry-run tests"
	@echo "  make help         Show this help"
