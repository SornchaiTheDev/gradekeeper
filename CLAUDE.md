# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Gradekeeper is a CLI application written in Go that automates development environment setup for Windows users. It supports both standalone mode and master-client architecture for managing multiple computers.

**Standalone Mode**: Creates a "DOMJudge" folder on the Windows Desktop, opens VS Code with that folder, and launches Chrome with multiple tabs.

**Master-Client Mode**: Provides centralized control over multiple Windows computers through WebSocket communication and a web dashboard.

## Technology Stack

- Go (golang) - Main programming language
- Windows-specific APIs and file system operations

## Build Commands

### Build All Components:
**On Windows:**
```bash
build-all.bat
```

**On Linux/macOS (cross-compilation):**
```bash
./build-all.sh
```

### Build Individual Components:
```bash
# Standalone version
GOOS=windows GOARCH=amd64 go build -o gradekeeper-standalone.exe main.go

# Client version
GOOS=windows GOARCH=amd64 go build -o gradekeeper-client.exe client.go

# Master server
cd master && GOOS=windows GOARCH=amd64 go build -o gradekeeper-master.exe main.go
```

## Architecture

### Core Components:
1. **`main.go`** - Standalone application
2. **`client.go`** - WebSocket client with standalone fallback
3. **`master/main.go`** - Master server with web dashboard

### Key Functions:
- `getDesktopPath()` - Detects Windows Desktop path using USERPROFILE environment variable
- `openVSCode()` - Attempts to launch VS Code with multiple fallback paths  
- `openChromeWithTabs()` - Opens Chrome with multiple tabs, with fallback to default browser
- `openChrome()` - Wrapper function for single URL (calls openChromeWithTabs)

### Master-Client Architecture:
- **WebSocket Communication**: Real-time command execution
- **Command Types**: `setup`, `open-vscode`, `open-chrome`
- **Web Dashboard**: HTML interface at `http://localhost:8080`
- **Client Management**: Connection tracking and status monitoring

## Development Notes

- Application is Windows-only (runtime.GOOS check)
- Uses Go's os/exec package for launching external programs
- Handles multiple installation paths for VS Code and Chrome
- Provides user feedback for each operation