# Terminal Wrapped - Build Configuration
# Version
VERSION ?= 1.0.0

# Binary name
BINARY_NAME=terminal-wrapped

# Build directory
BUILD_DIR=dist

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Main package path
MAIN_PACKAGE=./cmd/terminal-wrapped

# Build flags
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

# Platforms
PLATFORMS=darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64

.PHONY: all build clean test deps build-all install help

all: clean deps test build

help: ## Display this help screen
	@echo "Terminal Wrapped - Makefile commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) tidy

build: ## Build for current platform
	@echo "Building $(BINARY_NAME) for current platform..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

build-all: clean ## Build for all platforms
	@echo "Building $(BINARY_NAME) v$(VERSION) for all platforms..."
	@mkdir -p $(BUILD_DIR)
	@$(foreach platform,$(PLATFORMS),\
		$(eval GOOS=$(word 1,$(subst /, ,$(platform))))\
		$(eval GOARCH=$(word 2,$(subst /, ,$(platform))))\
		$(eval OUTPUT=$(BUILD_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH)$(if $(filter windows,$(GOOS)),.exe,))\
		echo "Building for $(GOOS)/$(GOARCH)..." && \
		GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOBUILD) $(LDFLAGS) -o $(OUTPUT) $(MAIN_PACKAGE) && \
		echo "  ✓ $(OUTPUT)" || exit 1;\
	)
	@echo ""
	@echo "All builds complete! Binaries are in $(BUILD_DIR)/"
	@ls -lh $(BUILD_DIR)

install: ## Install to $GOPATH/bin
	$(GOCMD) install $(LDFLAGS) $(MAIN_PACKAGE)

test: ## Run tests
	$(GOTEST) -v ./...

clean: ## Remove build artifacts
	@echo "Cleaning..."
	@$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

run: ## Run the application
	$(GOCMD) run $(MAIN_PACKAGE)

.DEFAULT_GOAL := help
