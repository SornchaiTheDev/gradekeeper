package platform

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
)

// GetDesktopPath returns the cross-platform desktop path
func GetDesktopPath() (string, error) {
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

// OpenVSCode opens VS Code with the specified folder path
func OpenVSCode(folderPath string) error {
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

// OpenBrowserWithTabs opens the default browser with multiple tabs
func OpenBrowserWithTabs(urls []string) error {
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
		// Add incognito mode flag
		args := []string{"--incognito"}
		args = append(args, urls...)
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
	// Try Chrome/Chromium browsers first with incognito mode
	chromeBrowsers := []string{
		"google-chrome",
		"google-chrome-stable",
		"chromium-browser",
		"chromium",
	}

	for _, browser := range chromeBrowsers {
		// Add incognito mode flag for Chrome/Chromium
		args := []string{"--incognito"}
		args = append(args, urls...)
		cmd := exec.Command(browser, args...)
		err := cmd.Start()
		if err == nil {
			return nil
		}
	}

	// Try Firefox with private mode
	firefoxBrowsers := []string{
		"firefox",
		"firefox-esr",
	}

	for _, browser := range firefoxBrowsers {
		// Add private browsing flag for Firefox
		args := []string{"--private-window"}
		args = append(args, urls...)
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
	// Try Chrome first on macOS with incognito mode
	chromeCommand := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	args := []string{"--incognito"}
	args = append(args, urls...)
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

// ClearEnvironment removes DOMJudge folder and closes VS Code and browser processes
func ClearEnvironment() error {
	var errors []string

	// Remove DOMJudge folder
	if err := removeDOMJudgeFolder(); err != nil {
		errors = append(errors, fmt.Sprintf("failed to remove DOMJudge folder: %v", err))
	}

	// Close VS Code processes
	if err := closeVSCode(); err != nil {
		errors = append(errors, fmt.Sprintf("failed to close VS Code: %v", err))
	}

	// Close browser processes
	if err := closeBrowser(); err != nil {
		errors = append(errors, fmt.Sprintf("failed to close browser: %v", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("clear environment had errors: %v", errors)
	}

	return nil
}

// removeDOMJudgeFolder removes the DOMJudge folder from desktop
func removeDOMJudgeFolder() error {
	desktopPath, err := GetDesktopPath()
	if err != nil {
		return fmt.Errorf("error getting desktop path: %v", err)
	}

	domjudgePath := filepath.Join(desktopPath, "DOMJudge")
	
	// Check if folder exists
	if _, err := os.Stat(domjudgePath); os.IsNotExist(err) {
		// Folder doesn't exist, nothing to do
		return nil
	}

	// Remove the folder and all its contents
	err = os.RemoveAll(domjudgePath)
	if err != nil {
		return fmt.Errorf("failed to remove DOMJudge folder: %v", err)
	}

	fmt.Printf("DOMJudge folder removed: %s\n", domjudgePath)
	return nil
}

// closeVSCode closes all VS Code processes
func closeVSCode() error {
	switch runtime.GOOS {
	case "windows":
		// Close VS Code on Windows
		cmd := exec.Command("taskkill", "/F", "/IM", "Code.exe")
		err := cmd.Run()
		if err != nil {
			// Don't treat as error if no process found
			return nil
		}
		fmt.Println("VS Code processes closed")
	case "linux":
		// Close VS Code on Linux
		cmd := exec.Command("pkill", "-f", "code")
		err := cmd.Run()
		if err != nil {
			// Don't treat as error if no process found
			return nil
		}
		fmt.Println("VS Code processes closed")
	case "darwin":
		// Close VS Code on macOS
		cmd := exec.Command("pkill", "-f", "Visual Studio Code")
		err := cmd.Run()
		if err != nil {
			// Try alternative approach
			cmd = exec.Command("osascript", "-e", "quit app \"Visual Studio Code\"")
			cmd.Run()
		}
		fmt.Println("VS Code processes closed")
	default:
		return fmt.Errorf("VS Code closing not supported on %s", runtime.GOOS)
	}

	return nil
}

// closeBrowser closes browser processes (Chrome, Chromium, Firefox)
func closeBrowser() error {
	switch runtime.GOOS {
	case "windows":
		// Close browsers on Windows
		browsers := []string{"chrome.exe", "chromium.exe", "firefox.exe", "msedge.exe"}
		for _, browser := range browsers {
			cmd := exec.Command("taskkill", "/F", "/IM", browser)
			cmd.Run() // Ignore errors, process might not be running
		}
		fmt.Println("Browser processes closed")
	case "linux":
		// Close browsers on Linux
		browsers := []string{"google-chrome", "chromium", "firefox", "chrome"}
		for _, browser := range browsers {
			cmd := exec.Command("pkill", "-f", browser)
			cmd.Run() // Ignore errors, process might not be running
		}
		fmt.Println("Browser processes closed")
	case "darwin":
		// Close browsers on macOS
		browsers := []string{
			"quit app \"Google Chrome\"",
			"quit app \"Chromium\"",
			"quit app \"Firefox\"",
			"quit app \"Safari\"",
		}
		for _, browser := range browsers {
			cmd := exec.Command("osascript", "-e", browser)
			cmd.Run() // Ignore errors, app might not be running
		}
		fmt.Println("Browser processes closed")
	default:
		return fmt.Errorf("browser closing not supported on %s", runtime.GOOS)
	}

	return nil
}