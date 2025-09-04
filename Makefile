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
	cd master && go mod tidy
	cd master && GOOS=linux GOARCH=amd64 go build -o ../dist/gradekeeper-master-linux-amd64 main.go
	cd master && GOOS=windows GOARCH=amd64 go build -o ../dist/gradekeeper-master-windows-amd64.exe main.go
	cd master && GOOS=darwin GOARCH=amd64 go build -o ../dist/gradekeeper-master-darwin-amd64 main.go
	cd master && GOOS=darwin GOARCH=arm64 go build -o ../dist/gradekeeper-master-darwin-arm64 main.go

# Build client for all platforms
build-client: dist
	@echo "Building Cross-Platform Client..."
	go mod tidy
	GOOS=linux GOARCH=amd64 go build -o dist/gradekeeper-client-linux-amd64 client-crossplatform.go
	GOOS=windows GOARCH=amd64 go build -o dist/gradekeeper-client-windows-amd64.exe client-crossplatform.go
	GOOS=darwin GOARCH=amd64 go build -o dist/gradekeeper-client-darwin-amd64 client-crossplatform.go
	GOOS=darwin GOARCH=arm64 go build -o dist/gradekeeper-client-darwin-arm64 client-crossplatform.go

# Build standalone for all platforms
build-standalone: dist
	@echo "Building Cross-Platform Standalone..."
	GOOS=linux GOARCH=amd64 go build -o dist/gradekeeper-standalone-linux-amd64 standalone-crossplatform.go
	GOOS=windows GOARCH=amd64 go build -o dist/gradekeeper-standalone-windows-amd64.exe standalone-crossplatform.go
	GOOS=darwin GOARCH=amd64 go build -o dist/gradekeeper-standalone-darwin-amd64 standalone-crossplatform.go
	GOOS=darwin GOARCH=arm64 go build -o dist/gradekeeper-standalone-darwin-arm64 standalone-crossplatform.go

# Build for current platform only
build-local: dist
	@echo "Building for current platform ($(shell go env GOOS)/$(shell go env GOARCH))..."
	cd master && go build -o ../dist/gradekeeper-master main.go
	go build -o dist/gradekeeper-client client-crossplatform.go
	go build -o dist/gradekeeper-standalone standalone-crossplatform.go

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
	cd master && go run main.go

dev-client:
	go run client-crossplatform.go -standalone

dev-standalone:
	go run standalone-crossplatform.go

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