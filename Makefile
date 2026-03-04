.PHONY: all build run test test-cover test-race test-all frontend clean docker

BINARY=seedghost
GO=go
NPM=npm

all: build

# Build frontend
frontend:
	cd frontend && $(NPM) install && $(NPM) run build

# Build Go binary (requires frontend to be built first)
build: frontend
	$(GO) build -o $(BINARY) ./cmd/seedghost/

# Run in development mode
run:
	$(GO) run ./cmd/seedghost/

# Run backend tests
test:
	$(GO) test ./...

# Run tests with coverage
test-cover:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

# Run tests with race detector
test-race:
	$(GO) test -race ./...

# Run frontend tests
test-frontend:
	cd frontend && $(NPM) test -- --run

# Run all tests
test-all: test test-frontend

# Build Docker image
docker:
	docker build -t seedghost .

# Docker compose up
up:
	docker compose up -d

# Docker compose down
down:
	docker compose down

# Clean build artifacts
clean:
	rm -f $(BINARY) coverage.out coverage.html
	rm -rf web/dist
	rm -rf frontend/node_modules
