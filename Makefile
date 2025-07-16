.PHONY: all build build-cli build-gui test lint fmt clean run-gui run-cli help

# Default target
all: build

# Build all binaries
build: build-cli build-gui

# Build CLI
build-cli:
	@echo "Building CLI..."
	go build -v -ldflags "-s -w" -o bench$(shell go env GOEXE) ./cmd/fire

# Build GUI (platform-specific)
build-gui:
ifeq ($(OS),Windows_NT)
	@echo "Building GUI for Windows..."
	set CGO_ENABLED=1 && go build -v -o fire-gui.exe ./cmd/fire-gui
else
	@echo "Building GUI for Unix..."
	CGO_ENABLED=0 go build -v -ldflags "-s -w" -tags=no_glfw -o fire-gui ./cmd/fire-gui
endif

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	go test -v -tags=integration ./pkg/agent -run TestAgentIntegration

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run --timeout=5m

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f bench$(shell go env GOEXE) fire-gui$(shell go env GOEXE) coverage.txt
	rm -rf dist/ build/

# Run GUI
run-gui: build-gui
	@echo "Starting GUI..."
	./fire-gui$(shell go env GOEXE)

# Run GUI without splash
run-gui-nosplash: build-gui
	@echo "Starting GUI (no splash)..."
	./fire-gui$(shell go env GOEXE) --no-splash

# Run CLI
run-cli: build-cli
	@echo "Starting CLI..."
	./bench$(shell go env GOEXE) $(ARGS)

# Generate certificates
certs:
	@echo "Generating certificates..."
	bash scripts/generate-certs.sh

# Create new plugin
new-plugin:
	@if [ -z "$(NAME)" ]; then \
		echo "Usage: make new-plugin NAME=myplugin [CATEGORY=cpu]"; \
		exit 1; \
	fi
	@bash scripts/new-plugin.sh $(NAME) $(CATEGORY)

# Tidy modules
tidy:
	@echo "Tidying modules..."
	go mod tidy

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download

# Run specific plugin test
test-plugin:
	@if [ -z "$(PLUGIN)" ]; then \
		echo "Usage: make test-plugin PLUGIN=cpu"; \
		exit 1; \
	fi
	@echo "Testing plugin: $(PLUGIN)"
	go test -v ./pkg/plugin/$(PLUGIN)/...

# Show coverage report
coverage: test
	@echo "Generating coverage report..."
	go tool cover -html=coverage.txt

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t fire:latest .

# Docker run
docker-run:
	@echo "Running Docker container..."
	docker-compose up

# Help
help:
	@echo "FIRE Project Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all              - Build all binaries (default)"
	@echo "  build            - Build both CLI and GUI"
	@echo "  build-cli        - Build CLI binary"
	@echo "  build-gui        - Build GUI binary"
	@echo "  test             - Run all tests"
	@echo "  test-integration - Run integration tests"
	@echo "  lint             - Run golangci-lint"
	@echo "  fmt              - Format code with gofmt"
	@echo "  clean            - Remove build artifacts"
	@echo "  run-gui          - Build and run GUI"
	@echo "  run-gui-nosplash - Build and run GUI without splash"
	@echo "  run-cli          - Build and run CLI (use ARGS= for arguments)"
	@echo "  certs            - Generate test certificates"
	@echo "  new-plugin       - Create new plugin (use NAME= and CATEGORY=)"
	@echo "  tidy             - Run go mod tidy"
	@echo "  deps             - Download dependencies"
	@echo "  test-plugin      - Test specific plugin (use PLUGIN=)"
	@echo "  coverage         - Generate and open coverage report"
	@echo "  docker-build     - Build Docker image"
	@echo "  docker-run       - Run Docker container"
	@echo "  help             - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make test-plugin PLUGIN=cpu"
	@echo "  make new-plugin NAME=mytest CATEGORY=memory"
	@echo "  make run-cli ARGS='test cpu -duration 30s'"