# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Gradekeeper is a cross-platform CLI application written in Go that automates development environment setup for Windows, Linux, and macOS users. It supports both standalone mode and master-client architecture for managing multiple computers.

**Standalone Mode**: Creates a "DOMJudge" folder on the user's Desktop, opens VS Code with that folder, and launches the browser with multiple tabs in incognito/private mode.

**Master-Client Mode**: Provides centralized control over multiple computers (any supported OS) through WebSocket communication and a web dashboard. Includes setup, VS Code opening, browser launching in incognito mode, and environment clearing commands.

## Technology Stack

- Go (golang) - Main programming language
- Cross-platform file system operations
- WebSocket communication (github.com/gorilla/websocket)
- Platform-specific executable detection and launching

## Build Commands

### Makefile Commands (Recommended):
```bash
# Cross-platform builds
make all                       # All components, all platforms  
make build-master             # Master server only
make build-client             # Client only
make build-standalone         # Standalone only
make build-local              # Current platform only

# Development helpers
make dev-master               # Run master in development mode
make dev-client               # Run client in development mode  
make dev-standalone           # Run standalone in development mode

# Utilities
make clean                    # Remove build artifacts
make test                     # Test builds completed successfully
make show                     # Show built executables
make help                     # Show all available commands
```

### Manual Build (if needed):
```bash
# Build individual components for current platform
go build -o dist/gradekeeper-master ./cmd/gradekeeper-master
go build -o dist/gradekeeper-client ./cmd/gradekeeper-client
go build -o dist/gradekeeper-standalone ./cmd/gradekeeper-standalone
```

## Architecture

### Core Components:
1. **`cmd/gradekeeper-standalone/main.go`** - Cross-platform standalone application  
2. **`cmd/gradekeeper-client/main.go`** - Cross-platform WebSocket client with standalone fallback
3. **`cmd/gradekeeper-master/main.go`** - Master server with web dashboard
4. **`internal/platform/platform.go`** - Shared cross-platform functionality
5. **`internal/config/config.go`** - Centralized configuration for default URLs
6. **`internal/templates/`** - HTML templates and template handling for web dashboard

### Key Functions:
- `getDesktopPath()` - Cross-platform desktop detection (Windows: USERPROFILE, Linux: XDG/~, macOS: ~)
- `openVSCode()` - Platform-specific VS Code launching with multiple fallback paths
- `openBrowserWithTabs()` - Cross-platform browser opening with incognito/private mode and multiple fallbacks per OS
- `openChromeWindows()` - Windows-specific Chrome launching with --incognito flag
- `openBrowserLinux()` - Linux-specific browser launching (Chrome/Chromium with --incognito, Firefox with --private-window)
- `openBrowserMacOS()` - macOS-specific browser launching (Chrome with --incognito flag)
- `ClearEnvironment()` - Removes DOMJudge folder and closes VS Code and browser processes
- `closeVSCode()` - Cross-platform VS Code process termination
- `closeBrowser()` - Cross-platform browser process termination

### Master-Client Architecture:
- **WebSocket Communication**: Real-time command execution
- **Command Types**: `setup`, `open-vscode`, `open-chrome`, `clear`
- **Web Dashboard**: HTML interface at `http://localhost:8080`
- **Client Management**: Connection tracking and status monitoring
- **One-Click Operations**: Setup All (setup + VS Code + Chrome), Clear All with confirmation
- **Individual Client Control**: Per-client one-click setup and clear operations

## Development Notes

- **Standard Go Project Layout**: Uses `cmd/` directory for executables
- **Cross-platform Support**: Works on Windows, Linux, and macOS
- **Shared Library**: Common platform code in `internal/platform/`
- **Runtime Detection**: Uses runtime.GOOS for platform-specific behavior  
- **Executable Launching**: Platform-specific paths and commands for VS Code and browsers
- **Desktop Detection**: XDG compliance on Linux, standard paths on Windows/macOS
- **WebSocket Communication**: Real-time master-client communication
- **Graceful Fallbacks**: Multiple fallback strategies for all external programs
- **Incognito Mode**: All browser launches use incognito/private browsing mode
- **Environment Cleanup**: Clear command removes folders and closes applications
- **Centralized Configuration**: Default browser URLs managed via master dashboard with defaults in `internal/config/config.go`

## Customization

- **Master/Client Mode**: Manage the list of URLs from the *Browser URLs* panel on the master dashboard. Updates are saved by the server and broadcast to every connected client instantly.
- **Standalone Defaults**: Edit `internal/config/config.go` if you need to change the baked-in defaults that ship with the binaries (used only when no remote configuration exists).
