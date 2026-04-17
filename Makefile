# Spotnik Makefile
# All commands should be run from the project root.
# Usage: make <target>

# Load .env if present (for SPOTIFY_CLIENT_ID and other secrets).
-include .env
export

BINARY_NAME      = spotnik
BINARY_DIR       = bin
MODULE           = github.com/initgrep-apps/spotnik
GO               = go
GOFLAGS          = -trimpath
SPOTIFY_CLIENT_ID ?=
LDFLAGS          = -s -w
VERSION          ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")


# Build platforms for release
PLATFORMS = \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64

.PHONY: all build run test test-integration test-coverage lint fmt clean install release help

## Default target
all: lint test build

## Build the binary for the current platform
build:
	@echo "→ Building $(BINARY_NAME)..."
	@mkdir -p $(BINARY_DIR)
	$(GO) build $(GOFLAGS) \
		-ldflags="$(LDFLAGS) -X main.version=$(VERSION) -X $(MODULE)/cmd.spotifyClientID=$(SPOTIFY_CLIENT_ID)" \
		-o $(BINARY_DIR)/$(BINARY_NAME) \
		.
	@echo "✓ Built: $(BINARY_DIR)/$(BINARY_NAME)"
	@ls -lh $(BINARY_DIR)/$(BINARY_NAME)

## Build and run the app
run: build
	./$(BINARY_DIR)/$(BINARY_NAME)

## Run all tests
test:
	@echo "→ Running tests..."
	GOFLAGS="" $(GO) test ./... -race -count=1
	@echo "✓ All tests passed"

## Run integration tests (requires build tag)
test-integration:
	@echo "→ Running integration tests..."
	GOFLAGS="" $(GO) test -tags integration ./... -race -count=1
	@echo "✓ Integration tests passed"

## Run tests with coverage report
test-coverage:
	@echo "→ Running tests with coverage..."
	GOFLAGS="" $(GO) test ./... -race -count=1 \
		-coverprofile=coverage.out \
		-covermode=atomic
	$(GO) tool cover -func=coverage.out | tail -1
	@echo ""
	@echo "→ Checking coverage threshold (80%)..."
	@COVERAGE=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | tr -d '%'); \
		if [ "$$(echo "$$COVERAGE < 80" | bc)" = "1" ]; then \
			echo "✗ Coverage $$COVERAGE% is below 80% threshold"; \
			exit 1; \
		else \
			echo "✓ Coverage $$COVERAGE% meets threshold"; \
		fi

## Open coverage report in browser
coverage-html: test-coverage
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✓ Report written to coverage.html"

## Run linter
lint:
	@echo "→ Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install: https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...
	@echo "✓ Lint passed"

## Format all Go files
fmt:
	@echo "→ Formatting..."
	$(GO) fmt ./...
	@echo "✓ Formatted"

## Verify formatting (for CI — fails if files would change)
fmt-check:
	@echo "→ Checking formatting..."
	@UNFORMATTED=$$(gofmt -l .); \
		if [ -n "$$UNFORMATTED" ]; then \
			echo "✗ Unformatted files:"; \
			echo "$$UNFORMATTED"; \
			exit 1; \
		fi
	@echo "✓ All files formatted"

## Verify the module is tidy
tidy-check:
	@echo "→ Checking go.mod/go.sum..."
	$(GO) mod tidy
	@git diff --exit-code go.mod go.sum || (echo "✗ go.mod/go.sum not tidy. Run: go mod tidy" && exit 1)
	@echo "✓ Module tidy"

## Download dependencies
deps:
	@echo "→ Downloading dependencies..."
	$(GO) mod download
	@echo "✓ Dependencies ready"

## Install binary to GOPATH/bin
install:
	@echo "→ Installing $(BINARY_NAME)..."
	$(GO) install $(GOFLAGS) \
		-ldflags="$(LDFLAGS) -X main.version=$(VERSION) -X $(MODULE)/cmd.spotifyClientID=$(SPOTIFY_CLIENT_ID)" \
		./...
	@echo "✓ Installed: $$(which $(BINARY_NAME))"

## Build release binaries for all platforms
release:
	@echo "→ Building release binaries (version: $(VERSION))..."
	@mkdir -p $(BINARY_DIR)/release
	@$(foreach platform,$(PLATFORMS), \
		$(eval GOOS=$(word 1,$(subst /, ,$(platform)))) \
		$(eval GOARCH=$(word 2,$(subst /, ,$(platform)))) \
		$(eval EXT=$(if $(filter windows,$(GOOS)),.exe,)) \
		echo "  Building $(GOOS)/$(GOARCH)..."; \
		GOOS=$(GOOS) GOARCH=$(GOARCH) \
		$(GO) build $(GOFLAGS) \
			-ldflags="$(LDFLAGS) -X main.version=$(VERSION) -X $(MODULE)/cmd.spotifyClientID=$(SPOTIFY_CLIENT_ID)" \
			-o $(BINARY_DIR)/release/$(BINARY_NAME)-$(GOOS)-$(GOARCH)$(EXT) \
			.; \
	)
	@ls -lh $(BINARY_DIR)/release/
	@echo "✓ Release binaries built"

## Remove build artifacts
clean:
	@echo "→ Cleaning..."
	@rm -rf $(BINARY_DIR) coverage.out coverage.html
	@echo "✓ Clean"

## Run the full CI check (what CI runs)
ci: fmt-check tidy-check lint test-coverage build
	@echo ""
	@echo "✓ All CI checks passed"

## Show this help
help:
	@echo "spotnik — available make targets:"
	@echo ""
	@grep -E '^## ' Makefile | sed 's/## /  /' | while read line; do \
		target=$$(echo "$$line" | awk '{print $$1}'); \
		desc=$$(echo "$$line" | cut -d' ' -f2-); \
		printf "  %-20s %s\n" "$$target" "$$desc"; \
	done
	@echo ""
	@echo "  Variables:"
	@echo "    VERSION   Release version tag (default: git describe)"
