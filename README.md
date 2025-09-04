# GradeKeeper

A CLI application written in Go that automates development environment setup for Windows users, with support for centralized management of multiple computers.

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

- Windows operating system
- VS Code (optional, but recommended)
- Google Chrome (optional, will fallback to default browser)

## Installation

### Option 1: Download Release
Download the latest executables from the [Releases](../../releases) page:
- `gradekeeper-master.exe` - Master server with web dashboard
- `gradekeeper-client.exe` - Client that connects to master
- `gradekeeper-standalone.exe` - Original standalone version

### Option 2: Build from Source

#### Build All Versions:

**On Windows:**
```bash
build-all.bat
```

**On Linux/macOS (cross-compilation):**
```bash
./build-all.sh
```

#### Build Individual Components:

**Standalone version:**
```bash
GOOS=windows GOARCH=amd64 go build -o gradekeeper-standalone.exe main.go
```

**Client version:**
```bash
GOOS=windows GOARCH=amd64 go build -o gradekeeper-client.exe client.go
```

**Master server:**
```bash
cd master
GOOS=windows GOARCH=amd64 go build -o gradekeeper-master.exe main.go
```

## Usage

### Standalone Mode
Simply run the standalone executable:
```bash
gradekeeper-standalone.exe
```

The application will:
1. Create a "DOMJudge" folder on your Desktop
2. Open VS Code with that folder
3. Launch Chrome with multiple helpful tabs

### Master-Client Mode

#### 1. Start the Master Server
On the control computer, run:
```bash
gradekeeper-master.exe
```

Then open your browser to: `http://localhost:8080`

#### 2. Connect Clients
On each target computer, run:
```bash
gradekeeper-client.exe -server ws://MASTER_IP:8080/ws
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
# Connect to master server
gradekeeper-client.exe -server ws://192.168.1.100:8080/ws

# Run in standalone mode (same as gradekeeper-standalone.exe)
gradekeeper-client.exe -standalone
```

## Architecture

### Standalone Mode
Single Go file (`main.go`) with core functions:
- `getDesktopPath()` - Detects Windows Desktop path using USERPROFILE environment variable
- `openVSCode()` - Attempts to launch VS Code with multiple fallback paths
- `openChromeWithTabs()` - Opens Chrome with multiple tabs, with fallback to default browser

### Master-Client Mode
**Master Server (`master/main.go`):**
- WebSocket server for client communication
- HTTP server for web dashboard
- Command broadcasting to multiple clients
- Real-time client status monitoring

**Client (`client.go`):**
- WebSocket client connecting to master server
- Command execution engine
- Status reporting to master
- Fallback to standalone mode if no server specified

### Communication Protocol
- **WebSocket Messages**: JSON-formatted commands and status updates
- **Commands**: `setup`, `open-vscode`, `open-chrome`
- **Targeting**: Commands can target `all` clients or specific client IDs
- **Status Updates**: Real-time connection and execution status

## Customization

To modify the URLs that open in Chrome, edit the `urls` slice in `main.go`:

```go
urls := []string{
    "https://google.com",
    "https://github.com",
    "https://stackoverflow.com",
    // Add your custom URLs here
}
```

## Error Handling

The application includes robust error handling:
- Validates Windows environment
- Checks for Desktop directory existence
- Multiple fallback paths for VS Code and Chrome
- Graceful degradation to default browser if Chrome isn't found

## Contributing

Feel free to submit issues and pull requests to improve the application.

## License

MIT License - Feel free to use and modify as needed.

---

*Built with Go • Automated development environment setup for Windows*