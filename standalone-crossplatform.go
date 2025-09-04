package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
)

func main() {
	fmt.Printf("GradeKeeper Standalone (%s/%s)\n", runtime.GOOS, runtime.GOARCH)

	// Get Desktop path (cross-platform)
	desktopPath, err := getDesktopPath()
	if err != nil {
		fmt.Printf("Error getting desktop path: %v\n", err)
		os.Exit(1)
	}

	// Create DOMJudge folder
	domjudgePath := filepath.Join(desktopPath, "DOMJudge")
	fmt.Printf("Creating folder: %s\n", domjudgePath)

	err = os.MkdirAll(domjudgePath, os.ModePerm)
	if err != nil {
		fmt.Printf("Error creating folder: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("DOMJudge folder created successfully!")

	// Open VS Code with the folder
	fmt.Println("Opening VS Code...")
	err = openVSCode(domjudgePath)
	if err != nil {
		fmt.Printf("Error opening VS Code: %v\n", err)
	} else {
		fmt.Println("VS Code opened successfully!")
	}

	// Open browser with multiple tabs
	fmt.Println("Opening browser with multiple tabs...")
	urls := []string{
		"https://google.com",
		"https://github.com",
		"https://stackoverflow.com",
	}
	err = openBrowserWithTabs(urls)
	if err != nil {
		fmt.Printf("Error opening browser: %v\n", err)
	} else {
		fmt.Printf("Browser opened successfully with multiple tabs on %s!\n", runtime.GOOS)
	}

	fmt.Println("All tasks completed!")
}

// Cross-platform desktop path detection
func getDesktopPath() (string, error) {
	var desktopPath string

	switch runtime.GOOS {
	case "windows":
		// Windows: Use USERPROFILE environment variable
		userProfile := os.Getenv("USERPROFILE")
		if userProfile == "" {
			return "", fmt.Errorf("USERPROFILE environment variable not found")
		}
		desktopPath = filepath.Join(userProfile, "Desktop")

	case "linux":
		// Linux: Use XDG user dirs or fallback to ~/Desktop
		currentUser, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("failed to get current user: %v", err)
		}

		// Try XDG desktop directory first
		xdgDesktop := os.Getenv("XDG_DESKTOP_DIR")
		if xdgDesktop != "" {
			desktopPath = xdgDesktop
		} else {
			// Fallback to ~/Desktop
			desktopPath = filepath.Join(currentUser.HomeDir, "Desktop")
		}

	case "darwin":
		// macOS: ~/Desktop
		currentUser, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("failed to get current user: %v", err)
		}
		desktopPath = filepath.Join(currentUser.HomeDir, "Desktop")

	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	// Verify the desktop path exists, create if it doesn't
	if _, err := os.Stat(desktopPath); os.IsNotExist(err) {
		fmt.Printf("Desktop directory doesn't exist, creating: %s\n", desktopPath)
		if err := os.MkdirAll(desktopPath, os.ModePerm); err != nil {
			return "", fmt.Errorf("failed to create desktop directory: %v", err)
		}
	}

	return desktopPath, nil
}

// Cross-platform VS Code opening
func openVSCode(folderPath string) error {
	var vscodeCommands []string

	switch runtime.GOOS {
	case "windows":
		vscodeCommands = []string{
			"code",
			"code.cmd",
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Microsoft VS Code", "Code.exe"),
			filepath.Join(os.Getenv("PROGRAMFILES"), "Microsoft VS Code", "Code.exe"),
			filepath.Join(os.Getenv("PROGRAMFILES(X86)"), "Microsoft VS Code", "Code.exe"),
		}
	case "linux":
		vscodeCommands = []string{
			"code",
			"code-insiders",
			"/usr/bin/code",
			"/usr/local/bin/code",
			"/snap/bin/code",
			"/var/lib/flatpak/exports/bin/com.visualstudio.code",
		}
	case "darwin":
		vscodeCommands = []string{
			"code",
			"/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code",
			"/usr/local/bin/code",
		}
	default:
		return fmt.Errorf("VS Code opening not supported on %s", runtime.GOOS)
	}

	for _, cmdPath := range vscodeCommands {
		cmd := exec.Command(cmdPath, folderPath)
		err := cmd.Start()
		if err == nil {
			return nil
		}
	}

	return fmt.Errorf("VS Code not found in common locations")
}

// Cross-platform browser opening
func openBrowserWithTabs(urls []string) error {
	if len(urls) == 0 {
		return fmt.Errorf("no URLs provided")
	}

	switch runtime.GOOS {
	case "windows":
		return openChromeWindows(urls)
	case "linux":
		return openBrowserLinux(urls)
	case "darwin":
		return openBrowserMacOS(urls)
	default:
		return fmt.Errorf("browser opening not supported on %s", runtime.GOOS)
	}
}

func openChromeWindows(urls []string) error {
	chromeCommands := []string{
		"chrome",
		"chrome.exe",
		filepath.Join(os.Getenv("PROGRAMFILES"), "Google", "Chrome", "Application", "chrome.exe"),
		filepath.Join(os.Getenv("PROGRAMFILES(X86)"), "Google", "Chrome", "Application", "chrome.exe"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Google", "Chrome", "Application", "chrome.exe"),
	}

	for _, cmdPath := range chromeCommands {
		args := append([]string{}, urls...)
		cmd := exec.Command(cmdPath, args...)
		err := cmd.Start()
		if err == nil {
			return nil
		}
	}

	// Fallback to default browser
	for _, url := range urls {
		cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		cmd.Start()
	}

	return nil
}

func openBrowserLinux(urls []string) error {
	// Try common browsers on Linux
	browsers := []string{
		"google-chrome",
		"google-chrome-stable",
		"chromium-browser",
		"chromium",
		"firefox",
		"firefox-esr",
	}

	for _, browser := range browsers {
		args := append([]string{}, urls...)
		cmd := exec.Command(browser, args...)
		err := cmd.Start()
		if err == nil {
			return nil
		}
	}

	// Fallback to xdg-open for each URL
	for _, url := range urls {
		cmd := exec.Command("xdg-open", url)
		cmd.Start()
	}

	return nil
}

func openBrowserMacOS(urls []string) error {
	// Try Chrome first on macOS
	chromeCommand := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	args := append([]string{}, urls...)
	cmd := exec.Command(chromeCommand, args...)
	err := cmd.Start()
	if err == nil {
		return nil
	}

	// Fallback to default browser
	for _, url := range urls {
		cmd := exec.Command("open", url)
		cmd.Start()
	}

	return nil
}