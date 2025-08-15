# Makefile for BPL Backend

# Variables
BINARY_NAME=server
MAIN_FILE=main.go
DOCKER_IMAGE=bpl-backend
DOCKER_TAG=latest

# Go related variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Database related
DB_CONTAINER=db-local
MIGRATE_UP=./migrate up head
MIGRATE_DOWN=./migrate down
MIGRATE_VERSION=./migrate up

# Default target
.PHONY: all
all: clean deps test build

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  help          - Show this help message"
	@echo "  build         - Build the application"
	@echo "  run           - Run the application"
	@echo "  dev           - Start development server with hot reload"
	@echo ""
	@echo "Testing & Quality:"
	@echo "  test          - Run tests"
	@echo "  test-verbose  - Run tests with verbose output"
	@echo "  test-coverage - Generate test coverage report"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linter"
	@echo ""
	@echo "Security:"
	@echo "  security      - Run comprehensive security checks (vet + audit + vulncheck)"
	@echo ""
	@echo "Database:"
	@echo "  migrate-up    - Run database migrations up"
	@echo "  migrate-down  - Run database migrations down"
	@echo "  db-shell      - Connect to database shell"
	@echo "  db-logs       - Show database logs"
	@echo ""
	@echo "Documentation:"
	@echo "  swagger       - Generate swagger documentation"
	@echo ""
	@echo "Authentication:"
	@echo "  create-token  - Create JWT token (usage: make create-token ID=1 PERMISSIONS=admin,manager)"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run Docker container"
	@echo "  docker-stop   - Stop Docker container"
	@echo "  docker-logs   - Show container logs"
	@echo ""
	@echo "Infrastructure:"
	@echo "  infra-up      - Start local infrastructure"
	@echo "  infra-down    - Stop local infrastructure"
	@echo "  infra-logs    - Show infrastructure logs"
	@echo ""
	@echo "Setup & Maintenance:"
	@echo "  install-tools - Install development tools"
	@echo "  update        - Update dependencies"

.PHONY: update
update:
	@echo "Updating dependencies..."
	$(GOCMD) get -u
	$(GOMOD) tidy

# Build targets
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	CGO_ENABLED=0 GOOS=linux $(GOBUILD) -o bin/$(BINARY_NAME) -v $(MAIN_FILE)

# Run targets
.PHONY: run
run:
	@echo "Starting application..."
	$(GOCMD) run $(MAIN_FILE)

.PHONY: dev
dev: swagger
	@echo "Starting development server with hot reload..."
	air

# Test targets
.PHONY: test
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

.PHONY: test-verbose
test-verbose:
	@echo "Running tests with verbose output..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

.PHONY: test-coverage
test-coverage: test-verbose
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"

# Code quality targets
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

.PHONY: lint
lint:
	@echo "Running linter..."
	$(GOLINT) run

.PHONY: vet
vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

# Documentation targets
.PHONY: swagger
swagger:
	@echo "Generating Swagger documentation..."
	./generate-spec.sh

# Database targets
.PHONY: migrate-up
migrate-up:
	@echo "Running database migrations up..."
	$(MIGRATE_UP)

.PHONY: migrate-down
migrate-down:
	@echo "Running database migrations down..."
	$(MIGRATE_DOWN) 1


.PHONY: db-shell
db-shell:
	@echo "Connecting to database..."
	docker exec -it $(DB_CONTAINER) psql -U postgres -d postgres

.PHONY: db-logs
db-logs:
	@echo "Showing database logs..."
	docker logs $(DB_CONTAINER)

# Docker targets
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

.PHONY: docker-run
docker-run:
	@echo "Running Docker container..."
	docker run -d --name $(BINARY_NAME) -p 8000:8000 --env-file .env $(DOCKER_IMAGE):$(DOCKER_TAG)

.PHONY: docker-stop
docker-stop:
	@echo "Stopping Docker container..."
	docker stop $(BINARY_NAME) || true
	docker rm $(BINARY_NAME) || true

.PHONY: docker-logs
docker-logs:
	@echo "Showing container logs..."
	docker logs -f $(BINARY_NAME)

# Infrastructure targets
.PHONY: infra-up
infra-up:
	@echo "Starting local infrastructure..."
	@if [ -d "../bpl-2-infrastructure/local" ]; then \
		cd ../bpl-2-infrastructure/local && docker compose up -d; \
	else \
		echo "Infrastructure repo not found. Please clone bpl-2-infrastructure repo."; \
	fi

.PHONY: infra-down
infra-down:
	@echo "Stopping local infrastructure..."
	@if [ -d "../bpl-2-infrastructure/local" ]; then \
		cd ../bpl-2-infrastructure/local && docker compose down; \
	else \
		echo "Infrastructure repo not found."; \
	fi

.PHONY: infra-logs
infra-logs:
	@echo "Showing infrastructure logs..."
	@if [ -d "../bpl-2-infrastructure/local" ]; then \
		cd ../bpl-2-infrastructure/local && docker compose logs -f; \
	else \
		echo "Infrastructure repo not found."; \
	fi

# Installation targets
.PHONY: install-tools
install-tools:
	@echo "Installing development tools..."
	$(GOCMD) install github.com/swaggo/swag/cmd/swag@latest
	$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOCMD) install github.com/air-verse/air@latest
	$(GOCMD) install golang.org/x/vuln/cmd/govulncheck@latest
	$(GOCMD) install github.com/sonatype-nexus-community/nancy@latest


# Security targets
.PHONY: security
security:
	@echo "Running comprehensive security checks..."
	@echo "1. Running go vet for potential issues..."
	$(GOCMD) vet ./...
	@echo "2. Running dependency audit..."
	@which nancy > /dev/null || (echo "Installing nancy..." && $(GOCMD) install github.com/sonatype-nexus-community/nancy@latest)
	$(GOCMD) list -json -m all | nancy sleuth
	@echo "3. Installing/running govulncheck..."
	@which govulncheck > /dev/null || $(GOCMD) install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...
	@echo "All security checks completed!"

# Authentication targets
.PHONY: create-token
create-token:
	@if [ -z "$(ID)" ]; then \
		echo "Error: ID parameter is required. Usage: make create-token ID=1 [PERMISSIONS=admin,manager]"; \
		exit 1; \
	fi
	@if [ -n "$(PERMISSIONS)" ]; then \
		./create-test-token.sh -id=$(ID) -permissions=$(PERMISSIONS); \
	else \
		./create-test-token.sh -id=$(ID); \
	fi
