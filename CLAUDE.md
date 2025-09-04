# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Gradekeeper is a CLI application written in Go that automates development environment setup for Windows users. It creates a "DOMJudge" folder on the Windows Desktop, opens VS Code with that folder, and launches Chrome with multiple tabs (Google, GitHub, Stack Overflow).

## Technology Stack

- Go (golang) - Main programming language
- Windows-specific APIs and file system operations

## Build Commands

### On Windows:
```bash
build.bat
```

### On Linux/macOS (cross-compilation):
```bash
./build.sh
```

### Manual build:
```bash
GOOS=windows GOARCH=amd64 go build -o gradekeeper.exe main.go
```

## Architecture

The application is a single-file CLI tool (`main.go`) with the following key functions:

- `getDesktopPath()` - Detects Windows Desktop path using USERPROFILE environment variable
- `openVSCode()` - Attempts to launch VS Code with multiple fallback paths
- `openChromeWithTabs()` - Opens Chrome with multiple tabs, with fallback to default browser
- `openChrome()` - Wrapper function for single URL (calls openChromeWithTabs)

## Development Notes

- Application is Windows-only (runtime.GOOS check)
- Uses Go's os/exec package for launching external programs
- Handles multiple installation paths for VS Code and Chrome
- Provides user feedback for each operation