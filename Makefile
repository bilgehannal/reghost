.PHONY: all build install clean test lint run-daemon run-cli help build-all build-all-platforms

# Build variables
BINARY_DAEMON=reghostd
BINARY_CLI=reghostctl
INSTALL_PATH=/usr/local/bin
CONFIG_PATH=/etc/reghost.yml
LOG_PATH=/var/log/reghost.log
BUILD_DIR=build
VERSION?=dev

# Go variables
GOFLAGS=-v
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

# Platform configurations
PLATFORMS=darwin/amd64 darwin/arm64 linux/amd64 linux/arm64

all: build

## Build both daemon and CLI
build: build-daemon build-cli

## Build daemon
build-daemon:
	@echo "Building $(BINARY_DAEMON)..."
	@go build $(GOFLAGS) $(LDFLAGS) -o bin/$(BINARY_DAEMON) ./cmd/reghostd

## Build CLI
build-cli:
	@echo "Building $(BINARY_CLI)..."
	@go build $(GOFLAGS) $(LDFLAGS) -o bin/$(BINARY_CLI) ./cmd/reghostctl

## Install binaries with SUID and SETGID for daemon (requires root)
install: build
	@echo "Installing binaries to $(INSTALL_PATH)..."
	@sudo cp bin/$(BINARY_DAEMON) $(INSTALL_PATH)/
	@sudo cp bin/$(BINARY_CLI) $(INSTALL_PATH)/
	@echo "Setting ownership to root..."
	@sudo chown root $(INSTALL_PATH)/$(BINARY_DAEMON)
	@sudo chown root $(INSTALL_PATH)/$(BINARY_CLI)
	@echo "Setting group..."
	@if [ "$$(uname)" = "Darwin" ]; then \
		echo "  macOS detected - setting group to admin"; \
		sudo chgrp admin $(INSTALL_PATH)/$(BINARY_DAEMON); \
		sudo chgrp admin $(INSTALL_PATH)/$(BINARY_CLI); \
	else \
		echo "  Linux detected - setting group to root"; \
		sudo chgrp root $(INSTALL_PATH)/$(BINARY_DAEMON); \
		sudo chgrp root $(INSTALL_PATH)/$(BINARY_CLI); \
	fi
	@echo "Setting permissions (SUID + SETGID for both binaries)..."
	@sudo chmod 6755 $(INSTALL_PATH)/$(BINARY_DAEMON)
	@sudo chmod 6755 $(INSTALL_PATH)/$(BINARY_CLI)
	@echo ""
	@echo "✓ Installation complete!"
	@echo ""
	@echo "  Daemon: $(INSTALL_PATH)/$(BINARY_DAEMON) (SUID + SETGID enabled)"
	@echo "    - Owner: root"
	@echo "    - Group: $$(if [ "$$(uname)" = "Darwin" ]; then echo 'admin'; else echo 'root'; fi)"
	@echo "    - Can be run by any user with root privileges"
	@echo "    - Usage: $(BINARY_DAEMON)"
	@echo ""
	@echo "  CLI: $(INSTALL_PATH)/$(BINARY_CLI) (SUID + SETGID enabled)"
	@echo "    - Owner: root"
	@echo "    - Group: $$(if [ "$$(uname)" = "Darwin" ]; then echo 'admin'; else echo 'root'; fi)"
	@echo "    - Can be run by any user with root privileges"
	@echo "    - Usage: $(BINARY_CLI) <command>"

## Uninstall binaries (requires root)
uninstall:
	@echo "Uninstalling binaries..."
	@sudo rm -f $(INSTALL_PATH)/$(BINARY_DAEMON)
	@sudo rm -f $(INSTALL_PATH)/$(BINARY_CLI)
	@echo "Uninstallation complete!"

## Run daemon from bin/ (requires root)
run-daemon: build-daemon
	@echo "Running $(BINARY_DAEMON) from bin/..."
	@echo "Note: bin/ binary is not SUID. Use 'make install' for SUID setup."
	@sudo ./bin/$(BINARY_DAEMON)

## Run CLI
run-cli: build-cli
	@./bin/$(BINARY_CLI)

## Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

## Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## Run linter
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install with: brew install golangci-lint" && exit 1)
	@golangci-lint run ./...

## Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

## Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	@go mod tidy

## Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download

## Build for all platforms
build-all: clean-builds
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*}; \
		GOARCH=$${platform#*/}; \
		OUTPUT_DIR=$(BUILD_DIR)/$$GOOS-$$GOARCH; \
		mkdir -p $$OUTPUT_DIR; \
		echo "Building for $$GOOS/$$GOARCH..."; \
		GOOS=$$GOOS GOARCH=$$GOARCH go build $(LDFLAGS) -o $$OUTPUT_DIR/$(BINARY_DAEMON) ./cmd/reghostd; \
		GOOS=$$GOOS GOARCH=$$GOARCH go build $(LDFLAGS) -o $$OUTPUT_DIR/$(BINARY_CLI) ./cmd/reghostctl; \
		echo "✓ Built $$GOOS/$$GOARCH"; \
	done
	@echo ""
	@echo "Build complete! Binaries available in $(BUILD_DIR)/"
	@ls -lh $(BUILD_DIR)

## Build and package all platforms
build-all-platforms: build-all
	@echo "Creating archives..."
	@cd $(BUILD_DIR) && for dir in */; do \
		platform=$${dir%/}; \
		archive_name=reghost-$(VERSION)-$$platform.tar.gz; \
		echo "Creating $$archive_name..."; \
		tar -czf $$archive_name $$platform; \
	done
	@echo ""
	@echo "Archives created:"
	@ls -lh $(BUILD_DIR)/*.tar.gz

## Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "Clean complete!"

## Clean multi-platform builds
clean-builds:
	@echo "Cleaning build directory..."
	@rm -rf $(BUILD_DIR)

## Show help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build:"
	@echo "  build              - Build both daemon and CLI for current platform"
	@echo "  build-daemon       - Build daemon only"
	@echo "  build-cli          - Build CLI only"
	@echo "  build-all          - Build for all platforms (macOS/Linux x64/ARM)"
	@echo "  build-all-platforms - Build and create archives for all platforms"
	@echo ""
	@echo "Install:"
	@echo "  install            - Install binaries with SUID (requires root)"
	@echo "                       Daemon will run with root privileges for any user"
	@echo "  uninstall          - Uninstall binaries (requires root)"
	@echo ""
	@echo "Run:"
	@echo "  run-daemon         - Build and run daemon from bin/ (requires sudo)"
	@echo "  run-cli            - Build and run CLI from bin/"
	@echo ""
	@echo "Test & Quality:"
	@echo "  test               - Run tests"
	@echo "  test-coverage      - Run tests with coverage report"
	@echo "  lint               - Run linter"
	@echo "  fmt                - Format code"
	@echo ""
	@echo "Maintenance:"
	@echo "  tidy               - Tidy dependencies"
	@echo "  deps               - Download dependencies"
	@echo "  clean              - Clean build artifacts"
	@echo "  clean-builds       - Clean multi-platform build directory"
	@echo ""
	@echo "  help               - Show this help message"
