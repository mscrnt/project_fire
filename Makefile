# FIRE Makefile

# Read version from VERSION file
VERSION := $(shell cat VERSION)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS := -ldflags "-s -w -X main.buildVersion=v$(VERSION) -X main.buildCommit=$(COMMIT) -X 'main.buildTime=$(BUILD_TIME)'"

# Default target
.PHONY: all
all: build

# Build CLI
.PHONY: build
build:
	go build $(LDFLAGS) -o bench ./cmd/fire

# Build GUI (requires CGO)
.PHONY: build-gui
build-gui:
	CGO_ENABLED=1 go build $(LDFLAGS) -o fire-gui ./cmd/fire-gui

# Build all
.PHONY: build-all
build-all: build build-gui

# Build for Windows (from Linux/WSL)
.PHONY: build-windows
build-windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bench.exe ./cmd/fire
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build $(LDFLAGS) -o fire-gui.exe ./cmd/fire-gui

# Run tests
.PHONY: test
test:
	go test -v -race ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	go tool cover -html=coverage.txt -o coverage.html

# Format code
.PHONY: fmt
fmt:
	go fmt ./...

# Run linter
.PHONY: lint
lint:
	golangci-lint run --timeout=5m

# Clean build artifacts
.PHONY: clean
clean:
	rm -f bench bench.exe fire-gui fire-gui.exe
	rm -f coverage.txt coverage.html
	rm -rf dist/ build/ release/

# Install dependencies
.PHONY: deps
deps:
	go mod download
	go mod tidy

# Bump version
.PHONY: bump-patch
bump-patch:
	./scripts/bump-version.sh patch

.PHONY: bump-minor
bump-minor:
	./scripts/bump-version.sh minor

.PHONY: bump-major
bump-major:
	./scripts/bump-version.sh major

# Show current version
.PHONY: version
version:
	@echo "Current version: v$(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build time: $(BUILD_TIME)"

# Build and run CLI
.PHONY: run
run: build
	./bench

# Build and run GUI
.PHONY: run-gui
run-gui: build-gui
	./fire-gui

# Docker build
.PHONY: docker
docker:
	docker build -t fire:$(VERSION) .
	docker tag fire:$(VERSION) fire:latest

# Help
.PHONY: help
help:
	@echo "FIRE Makefile Commands:"
	@echo "  make build        - Build CLI binary"
	@echo "  make build-gui    - Build GUI binary (requires CGO)"
	@echo "  make build-all    - Build both CLI and GUI"
	@echo "  make build-windows - Cross-compile for Windows"
	@echo "  make test         - Run tests"
	@echo "  make test-coverage - Run tests with coverage"
	@echo "  make fmt          - Format code"
	@echo "  make lint         - Run linter"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make deps         - Download dependencies"
	@echo "  make bump-patch   - Bump patch version"
	@echo "  make bump-minor   - Bump minor version"
	@echo "  make bump-major   - Bump major version"
	@echo "  make version      - Show current version"
	@echo "  make run          - Build and run CLI"
	@echo "  make run-gui      - Build and run GUI"
	@echo "  make docker       - Build Docker image"
	@echo "  make help         - Show this help"