.PHONY: test test-verbose test-coverage build lint fmt vet clean examples benchmark

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet

# Build targets
build:
	$(GOBUILD) -v ./...

# Test targets (following TDD requirements)
test:
	$(GOTEST) -v ./...

test-verbose:
	$(GOTEST) -v -race ./...

test-coverage:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-coverage-check:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@$(GOCMD) tool cover -func=coverage.out | grep "total:" | awk '{if ($$3 < 80.0) {print "ERROR: Test coverage " $$3 " is below 80%"; exit 1} else {print "✓ Test coverage " $$3}}'

# Quality targets
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run

fmt:
	$(GOFMT) -s -w .

vet:
	$(GOVET) ./...

# Dependency management
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Benchmark targets
benchmark:
	$(GOTEST) -bench=. -benchmem ./...

benchmark-verbose:
	$(GOTEST) -bench=. -benchmem -v ./...

# Example targets
examples:
	$(GOBUILD) -v ./examples/...

example-simple:
	$(GOBUILD) -v ./examples/simple

example-advanced:
	$(GOBUILD) -v ./examples/advanced

# Development helpers
dev-setup: deps
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Development setup complete"

# CI targets
ci: fmt vet lint test-coverage-check
	@echo "✓ All CI checks passed"

# Clean targets
clean:
	$(GOCLEAN)
	rm -f coverage.out coverage.html

# Help target
help:
	@echo "Available targets:"
	@echo "  build              - Build the project"
	@echo "  test               - Run tests"
	@echo "  test-verbose       - Run tests with race detection"
	@echo "  test-coverage      - Run tests with coverage report"
	@echo "  test-coverage-check- Check test coverage meets 80% requirement"
	@echo "  lint               - Run linter"
	@echo "  fmt                - Format code"
	@echo "  vet                - Run go vet"
	@echo "  deps               - Download and tidy dependencies"
	@echo "  benchmark          - Run benchmarks"
	@echo "  examples           - Build examples"
	@echo "  dev-setup          - Set up development environment"
	@echo "  ci                 - Run all CI checks"
	@echo "  clean              - Clean build artifacts"