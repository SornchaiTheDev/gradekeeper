# GradeKeeper

A cross-platform CLI application written in Go that automates development environment setup for Windows, Linux, and macOS users, with support for centralized management of multiple computers.

## Features

### Standalone Mode
- 📁 Creates a "DOMJudge" folder on Windows Desktop
- 💻 Opens VS Code with the created folder
- 🌐 Opens Chrome with multiple useful tabs:
  - Google
  - GitHub
  - Stack Overflow

### Master-Client Mode
- 🎛️ **Master Server**: Web-based dashboard to control multiple client machines
- 🔗 **WebSocket Communication**: Real-time command execution across multiple computers
- 📊 **Client Monitoring**: Track connected clients and their status
- 🚀 **Remote Commands**: Execute setup, VS Code, and Chrome commands on selected clients
- 🌐 **Web Dashboard**: Easy-to-use interface at `http://localhost:8080`

## Requirements

### All Platforms
- VS Code (optional, but recommended)
- A web browser (Chrome, Firefox, etc.)

### Platform-Specific Notes
- **Windows**: Uses `%USERPROFILE%\Desktop` for Desktop path
- **Linux**: Uses XDG Desktop directory or `~/Desktop`, supports common browsers (Chrome, Chromium, Firefox)
- **macOS**: Uses `~/Desktop`, supports Chrome and default browser via `open` command

## Installation

### Option 1: Download Release
Download the latest executables from the [Releases](../../releases) page:

**For your platform:**
- Linux: `gradekeeper-*-linux-amd64`
- Windows: `gradekeeper-*-windows-amd64.exe`
- macOS Intel: `gradekeeper-*-darwin-amd64` 
- macOS Apple Silicon: `gradekeeper-*-darwin-arm64`

**Available versions:**
- Master server with web dashboard
- Client that connects to master
- Standalone version (original functionality)

### Option 2: Build from Source

#### Cross-Platform Build (Recommended):

**Using build scripts:**
```bash
# Linux/macOS
./build-crossplatform.sh

# Windows
build-crossplatform.bat
```

**Using Makefile (Linux/macOS):**
```bash
make all              # Build all components for all platforms
make build-local      # Build for current platform only
make dev-standalone   # Run standalone in development mode
```

#### Build Individual Components:

**Cross-platform versions:**
```bash
# Standalone
go build -o gradekeeper-standalone standalone-crossplatform.go

# Client  
go build -o gradekeeper-client client-crossplatform.go

# Master server
cd master && go build -o gradekeeper-master main.go
```

**Legacy Windows-only versions:**
```bash
# Original Windows-only versions (deprecated)
GOOS=windows GOARCH=amd64 go build -o gradekeeper-standalone.exe main.go
GOOS=windows GOARCH=amd64 go build -o gradekeeper-client.exe client.go
```

## Usage

### Standalone Mode
Simply run the standalone executable:
```bash
# Linux/macOS
./gradekeeper-standalone-linux-amd64
./gradekeeper-standalone-darwin-amd64

# Windows
gradekeeper-standalone-windows-amd64.exe
```

The application will:
1. Create a "DOMJudge" folder on your Desktop
2. Open VS Code with that folder
3. Launch your default browser (or Chrome if available) with multiple helpful tabs

### Master-Client Mode

#### 1. Start the Master Server
On the control computer, run:
```bash
# Linux
./gradekeeper-master-linux-amd64

# Windows  
gradekeeper-master-windows-amd64.exe

# macOS
./gradekeeper-master-darwin-amd64
```

Then open your browser to: `http://localhost:8080`

#### 2. Connect Clients
On each target computer, run:
```bash
# Linux
./gradekeeper-client-linux-amd64 -server ws://MASTER_IP:8080/ws

# Windows
gradekeeper-client-windows-amd64.exe -server ws://MASTER_IP:8080/ws

# macOS
./gradekeeper-client-darwin-amd64 -server ws://MASTER_IP:8080/ws
```

Replace `MASTER_IP` with the IP address of the master server computer.

#### 3. Control from Dashboard
- View connected clients in real-time
- Execute commands on all clients or specific ones:
  - **Setup Environment**: Creates DOMJudge folder
  - **Open VS Code**: Opens VS Code with the DOMJudge folder
  - **Open Chrome**: Opens Chrome with multiple useful tabs
- Monitor command execution results

### Client Mode Options
```bash
# Connect to master server (Linux/macOS)
./gradekeeper-client-linux-amd64 -server ws://192.168.1.100:8080/ws

# Connect to master server (Windows)
gradekeeper-client-windows-amd64.exe -server ws://192.168.1.100:8080/ws

# Run in standalone mode
./gradekeeper-client-linux-amd64 -standalone      # Linux
gradekeeper-client-windows-amd64.exe -standalone  # Windows
```

## Architecture

### Standalone Mode
Cross-platform implementation (`standalone-crossplatform.go`) with core functions:
- `getDesktopPath()` - Cross-platform desktop path detection:
  - Windows: `%USERPROFILE%\Desktop`
  - Linux: XDG Desktop directory or `~/Desktop`  
  - macOS: `~/Desktop`
- `openVSCode()` - Platform-specific VS Code launching with multiple fallback paths
- `openBrowserWithTabs()` - Cross-platform browser opening:
  - Windows: Chrome → default browser
  - Linux: Chrome/Chromium/Firefox → `xdg-open`
  - macOS: Chrome → `open` command

### Master-Client Mode
**Master Server (`master/main.go`):**
- WebSocket server for client communication
- HTTP server for web dashboard
- Command broadcasting to multiple clients
- Real-time client status monitoring

**Client (`client-crossplatform.go`):**
- Cross-platform WebSocket client connecting to master server  
- Platform-aware command execution engine
- Status reporting to master
- Fallback to standalone mode if no server specified
- Works on Windows, Linux, and macOS

### Communication Protocol
- **WebSocket Messages**: JSON-formatted commands and status updates
- **Commands**: `setup`, `open-vscode`, `open-chrome`
- **Targeting**: Commands can target `all` clients or specific client IDs
- **Status Updates**: Real-time connection and execution status

## Customization

To modify the URLs that open in the browser, edit the `urls` slice in the relevant source file:

**For standalone mode** (`standalone-crossplatform.go`):
```go
urls := []string{
    "https://google.com",
    "https://github.com",
    "https://stackoverflow.com",
    // Add your custom URLs here
}
```

**For client mode** (`client-crossplatform.go`):
```go
// In the openChromeAction() function
urls := []string{
    "https://google.com",
    "https://github.com",
    "https://stackoverflow.com",
    // Add your custom URLs here
}
```

## Error Handling

The application includes robust cross-platform error handling:
- **Platform Detection**: Automatically detects and adapts to Windows, Linux, and macOS
- **Desktop Directory**: Creates Desktop directory if it doesn't exist
- **VS Code Fallbacks**: Multiple installation path attempts for each platform
- **Browser Fallbacks**: Graceful degradation to system default browser
- **Network Resilience**: Client reconnection and master server error recovery

## Contributing

Feel free to submit issues and pull requests to improve the application.

## License

MIT License - Feel free to use and modify as needed.

---

*Built with Go • Cross-platform automated development environment setup*