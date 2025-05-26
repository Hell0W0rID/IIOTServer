# EdgeX Foundry Complete Testing & Build Makefile

.PHONY: all build test test-unit test-integration test-coverage clean fmt lint deps

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Binary names
CORE_DATA_BINARY=core-data
CORE_METADATA_BINARY=core-metadata
CORE_COMMAND_BINARY=core-command
SUPPORT_NOTIFICATIONS_BINARY=support-notifications
SUPPORT_SCHEDULER_BINARY=support-scheduler
APP_SERVICE_BINARY=app-service-configurable
DEVICE_VIRTUAL_BINARY=device-virtual

# Directories
BUILD_DIR=build
COVERAGE_DIR=coverage
CMD_DIR=cmd
INTERNAL_DIR=internal
PKG_DIR=pkg
TEST_DIR=test

# Coverage settings
COVERAGE_PROFILE=$(COVERAGE_DIR)/coverage.out
COVERAGE_HTML=$(COVERAGE_DIR)/coverage.html
COVERAGE_THRESHOLD=80

all: clean fmt lint test build

# Build all services
build: build-core-data build-core-metadata build-core-command build-support-notifications build-support-scheduler build-app-service build-device-virtual

build-core-data:
	@echo "Building Core Data Service..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(CORE_DATA_BINARY) $(CMD_DIR)/core-data/main.go

build-core-metadata:
	@echo "Building Core Metadata Service..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(CORE_METADATA_BINARY) $(CMD_DIR)/core-metadata/main.go

build-core-command:
	@echo "Building Core Command Service..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(CORE_COMMAND_BINARY) $(CMD_DIR)/core-command/main.go

build-support-notifications:
	@echo "Building Support Notifications Service..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(SUPPORT_NOTIFICATIONS_BINARY) $(CMD_DIR)/support-notifications/main.go

build-support-scheduler:
	@echo "Building Support Scheduler Service..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(SUPPORT_SCHEDULER_BINARY) $(CMD_DIR)/support-scheduler/main.go

build-app-service:
	@echo "Building Application Service..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(APP_SERVICE_BINARY) $(CMD_DIR)/app-service-configurable/main.go

build-device-virtual:
	@echo "Building Device Virtual Service..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(DEVICE_VIRTUAL_BINARY) $(CMD_DIR)/device-virtual/main.go

# Testing
test: test-unit test-integration

test-unit:
	@echo "Running unit tests..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -v -race -coverprofile=$(COVERAGE_PROFILE) -covermode=atomic ./$(INTERNAL_DIR)/... ./$(PKG_DIR)/...

test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v -tags=integration ./$(TEST_DIR)/integration/...

test-coverage: test-unit
	@echo "Generating coverage report..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOCMD) tool cover -html=$(COVERAGE_PROFILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"
	@$(GOCMD) tool cover -func=$(COVERAGE_PROFILE) | grep total | awk '{print "Total coverage: " $$3}'

test-coverage-check: test-unit
	@echo "Checking coverage threshold..."
	@COVERAGE=$$($(GOCMD) tool cover -func=$(COVERAGE_PROFILE) | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ $$(echo "$$COVERAGE < $(COVERAGE_THRESHOLD)" | bc) -eq 1 ]; then \
		echo "Coverage $$COVERAGE% is below threshold $(COVERAGE_THRESHOLD)%"; \
		exit 1; \
	else \
		echo "Coverage $$COVERAGE% meets threshold $(COVERAGE_THRESHOLD)%"; \
	fi

# Benchmarks
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./$(INTERNAL_DIR)/... ./$(PKG_DIR)/...

# Performance tests
test-performance:
	@echo "Running performance tests..."
	$(GOTEST) -v -tags=performance ./$(TEST_DIR)/...

# Code quality
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

fmt-check:
	@echo "Checking code formatting..."
	@UNFORMATTED=$$($(GOFMT) -l .); \
	if [ -n "$$UNFORMATTED" ]; then \
		echo "The following files need formatting:"; \
		echo "$$UNFORMATTED"; \
		exit 1; \
	fi

lint:
	@echo "Running linter..."
	$(GOLINT) run ./...

# Dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

deps-update:
	@echo "Updating dependencies..."
	$(GOGET) -u ./...
	$(GOMOD) tidy

# Docker
docker-build:
	@echo "Building Docker images..."
	docker-compose build

docker-up:
	@echo "Starting services with Docker Compose..."
	docker-compose up -d

docker-down:
	@echo "Stopping services..."
	docker-compose down

docker-logs:
	@echo "Showing service logs..."
	docker-compose logs -f

# Infrastructure
infrastructure-up:
	@echo "Starting infrastructure services..."
	docker-compose up -d consul redis

infrastructure-down:
	@echo "Stopping infrastructure services..."
	docker-compose stop consul redis

# Health checks
health-check:
	@echo "Running health checks..."
	@./scripts/health-check.sh

# API tests
test-api:
	@echo "Running API tests..."
	$(GOTEST) -v -tags=api ./$(TEST_DIR)/...

# Load tests
test-load:
	@echo "Running load tests..."
	@./scripts/load-test.sh

# Security tests
test-security:
	@echo "Running security tests..."
	@./scripts/security-test.sh

# Clean
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -rf $(COVERAGE_DIR)

# Development helpers
dev-setup: deps
	@echo "Setting up development environment..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/stretchr/testify@latest

dev-run-core-data:
	@echo "Running Core Data Service in development mode..."
	$(GOCMD) run $(CMD_DIR)/core-data/main.go

dev-run-core-metadata:
	@echo "Running Core Metadata Service in development mode..."
	$(GOCMD) run $(CMD_DIR)/core-metadata/main.go

dev-run-core-command:
	@echo "Running Core Command Service in development mode..."
	$(GOCMD) run $(CMD_DIR)/core-command/main.go

# CI/CD helpers
ci: deps fmt-check lint test-coverage-check

# Release
release: clean ci build
	@echo "Creating release..."
	@./scripts/create-release.sh

# Help
help:
	@echo "EdgeX Foundry Complete - Available targets:"
	@echo "  build                  - Build all services"
	@echo "  test                   - Run all tests"
	@echo "  test-unit             - Run unit tests"
	@echo "  test-integration      - Run integration tests"
	@echo "  test-coverage         - Generate coverage report"
	@echo "  test-coverage-check   - Check coverage threshold"
	@echo "  bench                 - Run benchmarks"
	@echo "  fmt                   - Format code"
	@echo "  lint                  - Run linter"
	@echo "  deps                  - Download dependencies"
	@echo "  docker-build          - Build Docker images"
	@echo "  docker-up             - Start services with Docker"
	@echo "  infrastructure-up     - Start infrastructure services"
	@echo "  health-check          - Run health checks"
	@echo "  clean                 - Clean build artifacts"
	@echo "  dev-setup             - Setup development environment"
	@echo "  ci                    - Run CI pipeline"
	@echo "  help                  - Show this help"