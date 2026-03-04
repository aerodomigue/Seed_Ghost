.PHONY: all build run test test-cover test-race test-all frontend clean docker up down help

BINARY=seedghost
GO=go
NPM=npm

all: build ## Build everything (frontend + backend)

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

frontend: ## Build frontend
	cd frontend && $(NPM) install && $(NPM) run build

build: frontend ## Build Go binary (includes frontend)
	$(GO) build -o $(BINARY) ./cmd/seedghost/

run: ## Run in development mode
	$(GO) run ./cmd/seedghost/

dev: ## Run frontend dev server + backend concurrently
	@echo "Start backend:  go run ./cmd/seedghost/"
	@echo "Start frontend: cd frontend && npm run dev"
	@echo "Then open http://localhost:5173"

test: ## Run backend tests
	$(GO) test ./...

test-cover: ## Run tests with coverage report
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

test-race: ## Run tests with race detector
	$(GO) test -race ./...

test-frontend: ## Run frontend tests
	cd frontend && $(NPM) test -- --run

test-all: test test-frontend ## Run all tests (backend + frontend)

docker: ## Build Docker image
	docker build -t seedghost .

up: ## Start with docker compose
	docker compose up -d

down: ## Stop docker compose
	docker compose down

clean: ## Clean build artifacts
	rm -f $(BINARY) coverage.out coverage.html
	rm -rf web/dist
	rm -rf frontend/node_modules
