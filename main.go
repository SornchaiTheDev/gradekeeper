package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func main() {
	if runtime.GOOS != "windows" {
		fmt.Println("This application is designed for Windows only")
		os.Exit(1)
	}

	// Get Windows Desktop path
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

	// Open Chrome with multiple tabs
	fmt.Println("Opening Chrome with multiple tabs...")
	urls := []string{
		"https://google.com",
		"https://github.com",
		"https://stackoverflow.com",
	}
	err = openChromeWithTabs(urls)
	if err != nil {
		fmt.Printf("Error opening Chrome: %v\n", err)
	} else {
		fmt.Println("Chrome opened successfully with multiple tabs!")
	}

	fmt.Println("All tasks completed!")
}

func getDesktopPath() (string, error) {
	// Get user profile directory
	userProfile := os.Getenv("USERPROFILE")
	if userProfile == "" {
		return "", fmt.Errorf("USERPROFILE environment variable not found")
	}
	
	// Desktop is typically at %USERPROFILE%\Desktop
	desktopPath := filepath.Join(userProfile, "Desktop")
	
	// Verify the desktop path exists
	if _, err := os.Stat(desktopPath); os.IsNotExist(err) {
		return "", fmt.Errorf("desktop path does not exist: %s", desktopPath)
	}
	
	return desktopPath, nil
}

func openVSCode(folderPath string) error {
	// Try common VS Code executable paths
	vscodeCommands := []string{
		"code",
		"code.cmd",
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Microsoft VS Code", "Code.exe"),
		filepath.Join(os.Getenv("PROGRAMFILES"), "Microsoft VS Code", "Code.exe"),
		filepath.Join(os.Getenv("PROGRAMFILES(X86)"), "Microsoft VS Code", "Code.exe"),
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

func openChromeWithTabs(urls []string) error {
	if len(urls) == 0 {
		return fmt.Errorf("no URLs provided")
	}

	// Try common Chrome executable paths
	chromeCommands := []string{
		"chrome",
		"chrome.exe",
		filepath.Join(os.Getenv("PROGRAMFILES"), "Google", "Chrome", "Application", "chrome.exe"),
		filepath.Join(os.Getenv("PROGRAMFILES(X86)"), "Google", "Chrome", "Application", "chrome.exe"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Google", "Chrome", "Application", "chrome.exe"),
	}
	
	for _, cmdPath := range chromeCommands {
		// Build arguments for Chrome: all URLs will open as separate tabs
		args := append([]string{}, urls...)
		cmd := exec.Command(cmdPath, args...)
		err := cmd.Start()
		if err == nil {
			return nil
		}
	}
	
	// Fallback: open URLs one by one using default browser
	for _, url := range urls {
		cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		cmd.Start() // Don't wait for each command to complete
	}
	
	return nil
}

func openChrome(url string) error {
	return openChromeWithTabs([]string{url})
}