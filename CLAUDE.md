# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Gradekeeper is a cross-platform CLI application written in Go that automates development environment setup for Windows, Linux, and macOS users. It supports both standalone mode and master-client architecture for managing multiple computers.

**Standalone Mode**: Creates a "DOMJudge" folder on the user's Desktop, opens VS Code with that folder, and launches the default browser with multiple tabs.

**Master-Client Mode**: Provides centralized control over multiple computers (any supported OS) through WebSocket communication and a web dashboard.

## Technology Stack

- Go (golang) - Main programming language
- Cross-platform file system operations
- WebSocket communication (github.com/gorilla/websocket)
- Platform-specific executable detection and launching

## Build Commands

### Recommended Cross-Platform Build:
```bash
# Build all platforms
./build-crossplatform.sh       # Linux/macOS
build-crossplatform.bat        # Windows

# Using Makefile (Linux/macOS)
make all                       # All components, all platforms
make build-local              # Current platform only
make dev-standalone           # Development mode
```

### Legacy Windows-Only Build:
```bash
./build-all.sh               # Linux/macOS 
build-all.bat                # Windows
```

### Individual Components:
```bash
# Cross-platform versions (recommended)
go build -o gradekeeper-standalone standalone-crossplatform.go
go build -o gradekeeper-client client-crossplatform.go
cd master && go build -o gradekeeper-master main.go

# Legacy build scripts (now use cross-platform files)
./build-all.sh                # Builds Windows executables using cross-platform source
build-all.bat                 # Builds Windows executables using cross-platform source
```

## Architecture

### Core Components:
1. **`standalone-crossplatform.go`** - Cross-platform standalone application  
2. **`client-crossplatform.go`** - Cross-platform WebSocket client with standalone fallback
3. **`master/main.go`** - Master server with web dashboard

### Key Functions:
- `getDesktopPath()` - Cross-platform desktop detection (Windows: USERPROFILE, Linux: XDG/~, macOS: ~)
- `openVSCode()` - Platform-specific VS Code launching with multiple fallback paths
- `openBrowserWithTabs()` - Cross-platform browser opening with multiple fallbacks per OS
- `openChromeWindows()` - Windows-specific Chrome launching
- `openBrowserLinux()` - Linux-specific browser launching (Chrome/Chromium/Firefox → xdg-open)
- `openBrowserMacOS()` - macOS-specific browser launching (Chrome → open command)

### Master-Client Architecture:
- **WebSocket Communication**: Real-time command execution
- **Command Types**: `setup`, `open-vscode`, `open-chrome`
- **Web Dashboard**: HTML interface at `http://localhost:8080`
- **Client Management**: Connection tracking and status monitoring

## Development Notes

- **Cross-platform Support**: Works on Windows, Linux, and macOS
- **Runtime Detection**: Uses runtime.GOOS for platform-specific behavior  
- **Executable Launching**: Platform-specific paths and commands for VS Code and browsers
- **Desktop Detection**: XDG compliance on Linux, standard paths on Windows/macOS
- **WebSocket Communication**: Real-time master-client communication
- **Graceful Fallbacks**: Multiple fallback strategies for all external programs
- **Clean Architecture**: Removed duplicate legacy Windows-only code