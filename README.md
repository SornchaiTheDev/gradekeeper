# GradeKeeper

A CLI application written in Go that automates development environment setup for Windows users.

## Features

- 📁 Creates a "DOMJudge" folder on Windows Desktop
- 💻 Opens VS Code with the created folder
- 🌐 Opens Chrome with multiple useful tabs:
  - Google
  - GitHub
  - Stack Overflow

## Requirements

- Windows operating system
- VS Code (optional, but recommended)
- Google Chrome (optional, will fallback to default browser)

## Installation

### Option 1: Download Release
Download the latest `gradekeeper.exe` from the [Releases](../../releases) page.

### Option 2: Build from Source

#### On Windows:
```bash
build.bat
```

#### On Linux/macOS (cross-compilation):
```bash
./build.sh
```

#### Manual build:
```bash
GOOS=windows GOARCH=amd64 go build -o gradekeeper.exe main.go
```

## Usage

Simply run the executable:
```bash
gradekeeper.exe
```

The application will:
1. Create a "DOMJudge" folder on your Desktop
2. Open VS Code with that folder
3. Launch Chrome with multiple helpful tabs

## Architecture

The application consists of a single Go file with the following key functions:

- `getDesktopPath()` - Detects Windows Desktop path using USERPROFILE environment variable
- `openVSCode()` - Attempts to launch VS Code with multiple fallback paths
- `openChromeWithTabs()` - Opens Chrome with multiple tabs, with fallback to default browser

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