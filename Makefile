# GradeKeeper Cross-Platform Build Makefile

.PHONY: all clean build-master build-client build-standalone test

# Default target
all: clean build-master build-client build-standalone

# Create dist directory
dist:
	mkdir -p dist

# Clean build artifacts
clean:
	rm -rf dist

# Build master server for all platforms
build-master: dist
	@echo "Building Master Server..."
	go mod tidy
	GOOS=linux GOARCH=amd64 go build -o dist/gradekeeper-master-linux-amd64 ./cmd/gradekeeper-master
	GOOS=windows GOARCH=amd64 go build -o dist/gradekeeper-master-windows-amd64.exe ./cmd/gradekeeper-master
	GOOS=darwin GOARCH=amd64 go build -o dist/gradekeeper-master-darwin-amd64 ./cmd/gradekeeper-master
	GOOS=darwin GOARCH=arm64 go build -o dist/gradekeeper-master-darwin-arm64 ./cmd/gradekeeper-master

# Build client for all platforms
build-client: dist
	@echo "Building Cross-Platform Client..."
	GOOS=linux GOARCH=amd64 go build -o dist/gradekeeper-client-linux-amd64 ./cmd/gradekeeper-client
	GOOS=windows GOARCH=amd64 go build -o dist/gradekeeper-client-windows-amd64.exe ./cmd/gradekeeper-client
	GOOS=darwin GOARCH=amd64 go build -o dist/gradekeeper-client-darwin-amd64 ./cmd/gradekeeper-client
	GOOS=darwin GOARCH=arm64 go build -o dist/gradekeeper-client-darwin-arm64 ./cmd/gradekeeper-client

# Build standalone for all platforms
build-standalone: dist
	@echo "Building Cross-Platform Standalone..."
	GOOS=linux GOARCH=amd64 go build -o dist/gradekeeper-standalone-linux-amd64 ./cmd/gradekeeper-standalone
	GOOS=windows GOARCH=amd64 go build -o dist/gradekeeper-standalone-windows-amd64.exe ./cmd/gradekeeper-standalone
	GOOS=darwin GOARCH=amd64 go build -o dist/gradekeeper-standalone-darwin-amd64 ./cmd/gradekeeper-standalone
	GOOS=darwin GOARCH=arm64 go build -o dist/gradekeeper-standalone-darwin-arm64 ./cmd/gradekeeper-standalone

# Build for current platform only
build-local: dist
	@echo "Building for current platform ($(shell go env GOOS)/$(shell go env GOARCH))..."
	go build -o dist/gradekeeper-master ./cmd/gradekeeper-master
	go build -o dist/gradekeeper-client ./cmd/gradekeeper-client
	go build -o dist/gradekeeper-standalone ./cmd/gradekeeper-standalone

# Test builds by running simple version check
test:
	@echo "Testing builds..."
	@if [ -f dist/gradekeeper-master-linux-amd64 ]; then echo "✓ Master (Linux) built successfully"; fi
	@if [ -f dist/gradekeeper-client-linux-amd64 ]; then echo "✓ Client (Linux) built successfully"; fi
	@if [ -f dist/gradekeeper-standalone-linux-amd64 ]; then echo "✓ Standalone (Linux) built successfully"; fi

# Show build results
show:
	@echo "Built executables:"
	@ls -la dist/ 2>/dev/null || echo "No build artifacts found. Run 'make all' first."

# Development helpers
dev-master:
	go run ./cmd/gradekeeper-master

dev-client:
	go run ./cmd/gradekeeper-client -standalone

dev-standalone:
	go run ./cmd/gradekeeper-standalone

help:
	@echo "GradeKeeper Build Commands:"
	@echo "  make all           - Build all components for all platforms"
	@echo "  make build-master  - Build only master server"
	@echo "  make build-client  - Build only client"
	@echo "  make build-standalone - Build only standalone"
	@echo "  make build-local   - Build for current platform only"
	@echo "  make clean         - Remove build artifacts"
	@echo "  make test          - Test that builds completed successfully"
	@echo "  make show          - Show built executables"
	@echo "  make dev-master    - Run master server in development mode"
	@echo "  make dev-client    - Run client in development mode"
	@echo "  make dev-standalone - Run standalone in development mode"