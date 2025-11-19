package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"

	"gradekeeper/internal/config"
	"gradekeeper/internal/platform"
)

func main() {
	fmt.Printf("GradeKeeper Standalone (%s/%s)\n", runtime.GOOS, runtime.GOARCH)

	// Handle interrupt signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Run operations in a goroutine so we can handle interrupts
	done := make(chan bool, 1)

	go func() {
		// Get Desktop path (cross-platform)
		desktopPath, err := platform.GetDesktopPath()
		if err != nil {
			fmt.Printf("Error getting desktop path: %v\n", err)
			done <- false
			return
		}

		// Create DOMJudge folder
		domjudgePath := filepath.Join(desktopPath, "DOMJudge")
		fmt.Printf("Creating folder: %s\n", domjudgePath)

		err = os.MkdirAll(domjudgePath, os.ModePerm)
		if err != nil {
			fmt.Printf("Error creating folder: %v\n", err)
			done <- false
			return
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
		defaultCfg := config.DefaultAppConfig()
		err = platform.OpenBrowserWithTabs(defaultCfg.URLs)
		if err != nil {
			fmt.Printf("Error opening browser: %v\n", err)
		} else {
			fmt.Printf("Browser opened successfully with multiple tabs in incognito mode on %s!\n", runtime.GOOS)
		}

		fmt.Println("All tasks completed!")
		done <- true
	}()

	// Wait for completion or interrupt
	select {
	case success := <-done:
		if !success {
			os.Exit(1)
		}
	case <-interrupt:
		fmt.Println("\nInterrupt received, exiting...")
		os.Exit(0)
	}
}
