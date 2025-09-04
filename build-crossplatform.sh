#!/bin/bash
echo "Building GradeKeeper Cross-Platform..."

# Create dist directory
mkdir -p dist

echo ""
echo "Building Master Server..."
go mod tidy

# Build master for different platforms
echo "  - Linux (amd64)"
GOOS=linux GOARCH=amd64 go build -o dist/gradekeeper-master-linux-amd64 ./cmd/gradekeeper-master

echo "  - Windows (amd64)" 
GOOS=windows GOARCH=amd64 go build -o dist/gradekeeper-master-windows-amd64.exe ./cmd/gradekeeper-master

echo "  - macOS (amd64)"
GOOS=darwin GOARCH=amd64 go build -o dist/gradekeeper-master-darwin-amd64 ./cmd/gradekeeper-master

echo "  - macOS (arm64)"
GOOS=darwin GOARCH=arm64 go build -o dist/gradekeeper-master-darwin-arm64 ./cmd/gradekeeper-master

echo ""
echo "Building Cross-Platform Client..."

# Build client for different platforms
echo "  - Linux (amd64)"
GOOS=linux GOARCH=amd64 go build -o dist/gradekeeper-client-linux-amd64 ./cmd/gradekeeper-client

echo "  - Windows (amd64)"
GOOS=windows GOARCH=amd64 go build -o dist/gradekeeper-client-windows-amd64.exe ./cmd/gradekeeper-client

echo "  - macOS (amd64)"
GOOS=darwin GOARCH=amd64 go build -o dist/gradekeeper-client-darwin-amd64 ./cmd/gradekeeper-client

echo "  - macOS (arm64)"
GOOS=darwin GOARCH=arm64 go build -o dist/gradekeeper-client-darwin-arm64 ./cmd/gradekeeper-client

echo ""
echo "Building Cross-Platform Standalone..."

# Build standalone for different platforms
echo "  - Linux (amd64)"
GOOS=linux GOARCH=amd64 go build -o dist/gradekeeper-standalone-linux-amd64 ./cmd/gradekeeper-standalone

echo "  - Windows (amd64)"
GOOS=windows GOARCH=amd64 go build -o dist/gradekeeper-standalone-windows-amd64.exe ./cmd/gradekeeper-standalone

echo "  - macOS (amd64)"
GOOS=darwin GOARCH=amd64 go build -o dist/gradekeeper-standalone-darwin-amd64 ./cmd/gradekeeper-standalone

echo "  - macOS (arm64)"
GOOS=darwin GOARCH=arm64 go build -o dist/gradekeeper-standalone-darwin-arm64 ./cmd/gradekeeper-standalone

echo ""
echo "Build complete! Generated files in dist/:"
ls -la dist/

echo ""
echo "Platform-specific executables:"
echo "Linux:   ./dist/gradekeeper-*-linux-amd64"
echo "Windows: ./dist/gradekeeper-*-windows-amd64.exe" 
echo "macOS:   ./dist/gradekeeper-*-darwin-amd64 (Intel)"
echo "         ./dist/gradekeeper-*-darwin-arm64 (Apple Silicon)"