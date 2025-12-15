# VStats Makefile
# ================
# Provides convenient commands for development, testing, and building

.PHONY: all build test clean help
.PHONY: go-test go-build go-lint go-fmt
.PHONY: web-test web-build web-lint web-dev
.PHONY: e2e-test e2e-install
.PHONY: docker-build docker-up docker-down
.PHONY: install dev

# Default target
all: help

# ============================================================================
# Help
# ============================================================================

help:
	@echo "VStats Development Commands"
	@echo "============================"
	@echo ""
	@echo "Quick Start:"
	@echo "  make install          Install all dependencies"
	@echo "  make dev              Start development servers"
	@echo "  make test             Run all tests"
	@echo "  make build            Build all components"
	@echo ""
	@echo "Go Backend:"
	@echo "  make go-test          Run Go tests"
	@echo "  make go-test-cover    Run Go tests with coverage"
	@echo "  make go-build         Build Go binaries"
	@echo "  make go-lint          Run Go linter"
	@echo "  make go-fmt           Format Go code"
	@echo ""
	@echo "Frontend:"
	@echo "  make web-test         Run frontend tests"
	@echo "  make web-test-watch   Run frontend tests in watch mode"
	@echo "  make web-test-cover   Run frontend tests with coverage"
	@echo "  make web-build        Build frontend"
	@echo "  make web-lint         Run frontend linter"
	@echo "  make web-dev          Start frontend dev server"
	@echo ""
	@echo "E2E Tests:"
	@echo "  make e2e-install      Install Playwright browsers"
	@echo "  make e2e-test         Run E2E tests"
	@echo "  make e2e-test-ui      Run E2E tests with UI"
	@echo "  make e2e-report       Show E2E test report"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build     Build Docker images"
	@echo "  make docker-up        Start Docker containers"
	@echo "  make docker-down      Stop Docker containers"
	@echo ""
	@echo "Utilities:"
	@echo "  make clean            Clean build artifacts"
	@echo "  make lint             Run all linters"
	@echo "  make fmt              Format all code"

# ============================================================================
# Installation
# ============================================================================

install: install-go install-web install-e2e
	@echo "✅ All dependencies installed"

install-go:
	@echo "📦 Installing Go dependencies..."
	cd server-go && go mod download

install-web:
	@echo "📦 Installing frontend dependencies..."
	cd web && npm ci

install-e2e:
	@echo "📦 Installing E2E dependencies..."
	cd e2e && npm ci

# ============================================================================
# Development
# ============================================================================

dev:
	@echo "🚀 Starting development servers..."
	@echo "Starting backend on :8080 and frontend on :5173"
	@trap 'kill 0' SIGINT; \
	(cd server-go && go run ./cmd/server) & \
	(cd web && npm run dev) & \
	wait

dev-backend:
	@echo "🚀 Starting backend server..."
	cd server-go && go run ./cmd/server

dev-frontend:
	@echo "🚀 Starting frontend dev server..."
	cd web && npm run dev

# ============================================================================
# Testing
# ============================================================================

test: go-test web-test
	@echo "✅ All tests passed"

test-all: go-test web-test e2e-test
	@echo "✅ All tests (including E2E) passed"

# Go tests
go-test:
	@echo "🧪 Running Go tests..."
	cd server-go && go test -v -race ./...

go-test-cover:
	@echo "🧪 Running Go tests with coverage..."
	cd server-go && go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	cd server-go && go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report: server-go/coverage.html"

go-test-short:
	@echo "🧪 Running Go tests (short mode)..."
	cd server-go && go test -short ./...

# Frontend tests
web-test:
	@echo "🧪 Running frontend tests..."
	cd web && npm run test:run

web-test-watch:
	@echo "🧪 Running frontend tests in watch mode..."
	cd web && npm run test:watch

web-test-cover:
	@echo "🧪 Running frontend tests with coverage..."
	cd web && npm run test:coverage
	@echo "📊 Coverage report: web/coverage/index.html"

web-test-ui:
	@echo "🧪 Running frontend tests with UI..."
	cd web && npm run test:ui

# E2E tests
e2e-install:
	@echo "📦 Installing Playwright browsers..."
	cd e2e && npx playwright install --with-deps

e2e-test:
	@echo "🧪 Running E2E tests..."
	cd e2e && npm test

e2e-test-ui:
	@echo "🧪 Running E2E tests with UI..."
	cd e2e && npm run test:ui

e2e-test-headed:
	@echo "🧪 Running E2E tests in headed mode..."
	cd e2e && npm run test:headed

e2e-report:
	@echo "📊 Opening E2E test report..."
	cd e2e && npm run report

e2e-codegen:
	@echo "🔧 Starting Playwright codegen..."
	cd e2e && npm run codegen

# ============================================================================
# Building
# ============================================================================

build: go-build web-build
	@echo "✅ Build complete"

go-build:
	@echo "🔨 Building Go binaries..."
	cd server-go && go build -o ../bin/vstats-server ./cmd/server
	cd server-go && go build -o ../bin/vstats-agent ./cmd/agent
	cd server-go && go build -o ../bin/vstats-cli ./cmd/cli
	@echo "📦 Binaries: bin/vstats-server, bin/vstats-agent, bin/vstats-cli"

go-build-all:
	@echo "🔨 Building Go binaries for all platforms..."
	@mkdir -p bin
	# Linux AMD64
	cd server-go && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ../bin/vstats-server-linux-amd64 ./cmd/server
	cd server-go && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ../bin/vstats-agent-linux-amd64 ./cmd/agent
	# Linux ARM64
	cd server-go && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o ../bin/vstats-server-linux-arm64 ./cmd/server
	cd server-go && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o ../bin/vstats-agent-linux-arm64 ./cmd/agent
	# Darwin AMD64
	cd server-go && CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o ../bin/vstats-server-darwin-amd64 ./cmd/server
	cd server-go && CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o ../bin/vstats-agent-darwin-amd64 ./cmd/agent
	# Darwin ARM64
	cd server-go && CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o ../bin/vstats-server-darwin-arm64 ./cmd/server
	cd server-go && CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o ../bin/vstats-agent-darwin-arm64 ./cmd/agent
	# Windows AMD64
	cd server-go && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o ../bin/vstats-server-windows-amd64.exe ./cmd/server
	cd server-go && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o ../bin/vstats-agent-windows-amd64.exe ./cmd/agent
	@echo "📦 All platform binaries built in bin/"

web-build:
	@echo "🔨 Building frontend..."
	cd web && npm run build
	@echo "📦 Frontend built: web/dist/"

# ============================================================================
# Linting & Formatting
# ============================================================================

lint: go-lint web-lint
	@echo "✅ All linters passed"

go-lint:
	@echo "🔍 Running Go linter..."
	cd server-go && go vet ./...
	@which staticcheck > /dev/null && (cd server-go && staticcheck ./...) || echo "⚠️  staticcheck not installed, skipping"

go-fmt:
	@echo "🎨 Formatting Go code..."
	cd server-go && gofmt -s -w .

web-lint:
	@echo "🔍 Running frontend linter..."
	cd web && npm run lint

fmt: go-fmt
	@echo "✅ Code formatted"

# ============================================================================
# Docker
# ============================================================================

docker-build:
	@echo "🐳 Building Docker images..."
	docker build -t vstats:latest .
	docker build -f Dockerfile.agent -t vstats-agent:latest .

docker-up:
	@echo "🐳 Starting Docker containers..."
	docker-compose up -d

docker-down:
	@echo "🐳 Stopping Docker containers..."
	docker-compose down

docker-logs:
	docker-compose logs -f

# ============================================================================
# Cleanup
# ============================================================================

clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -rf bin/
	rm -rf web/dist/
	rm -rf web/coverage/
	rm -rf e2e/playwright-report/
	rm -rf e2e/test-results/
	rm -rf server-go/coverage.out
	rm -rf server-go/coverage.html
	@echo "✅ Cleaned"

clean-all: clean
	@echo "🧹 Deep cleaning (including node_modules)..."
	rm -rf web/node_modules/
	rm -rf e2e/node_modules/
	rm -rf site/node_modules/
	@echo "✅ Deep cleaned"

# ============================================================================
# Database
# ============================================================================

db-reset:
	@echo "🗄️ Resetting database..."
	rm -f vstats.db vstats.db-wal vstats.db-shm
	@echo "✅ Database reset"

db-backup:
	@echo "🗄️ Backing up database..."
	@mkdir -p backups
	cp vstats.db "backups/vstats-$$(date +%Y%m%d-%H%M%S).db"
	@echo "✅ Database backed up"

