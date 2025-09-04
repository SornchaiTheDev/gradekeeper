package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gradekeeper/internal/platform"
)

func main() {
	fmt.Printf("GradeKeeper Standalone (%s/%s)\n", runtime.GOOS, runtime.GOARCH)

	// Get Desktop path (cross-platform)
	desktopPath, err := platform.GetDesktopPath()
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
	err = platform.OpenVSCode(domjudgePath)
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
	err = platform.OpenBrowserWithTabs(urls)
	if err != nil {
		fmt.Printf("Error opening browser: %v\n", err)
	} else {
		fmt.Printf("Browser opened successfully with multiple tabs on %s!\n", runtime.GOOS)
	}

	fmt.Println("All tasks completed!")
}