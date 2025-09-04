package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

type Command struct {
	Action string `json:"action"`
	Target string `json:"target,omitempty"`
}

type Client struct {
	conn     *websocket.Conn
	serverURL string
	clientID  string
	done     chan struct{}
}

func NewClient(serverURL string) *Client {
	return &Client{
		serverURL: serverURL,
		clientID:  generateClientID(),
		done:      make(chan struct{}),
	}
}

func (c *Client) connect() error {
	u, err := url.Parse(c.serverURL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %v", err)
	}

	header := make(map[string][]string)
	header["X-Client-ID"] = []string{c.clientID}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		return fmt.Errorf("failed to connect to master: %v", err)
	}

	c.conn = conn
	fmt.Printf("Connected to master server as client: %s\n", c.clientID)
	return nil
}

func (c *Client) listen() {
	defer close(c.done)

	for {
		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			return
		}

		c.handleMessage(msg)
	}
}

func (c *Client) handleMessage(msg Message) {
	log.Printf("Received message: %+v", msg)

	switch msg.Type {
	case "welcome":
		fmt.Println("Welcome message received from master")
	case "command":
		if cmdData, ok := msg.Data.(map[string]interface{}); ok {
			action := cmdData["action"].(string)
			target := ""
			if cmdData["target"] != nil {
				target = cmdData["target"].(string)
			}

			// Check if command is for this client
			if target == "all" || target == "" || target == c.clientID {
				c.executeCommand(action)
			}
		}
	}
}

func (c *Client) executeCommand(action string) {
	fmt.Printf("Executing command: %s\n", action)

	var result map[string]interface{}
	var err error

	switch action {
	case "setup":
		err = c.setupEnvironment()
		result = map[string]interface{}{
			"action": action,
			"status": "completed",
			"error":  errorToString(err),
		}
	case "open-vscode":
		err = c.openVSCodeAction()
		result = map[string]interface{}{
			"action": action,
			"status": "completed",
			"error":  errorToString(err),
		}
	case "open-chrome":
		err = c.openChromeAction()
		result = map[string]interface{}{
			"action": action,
			"status": "completed",
			"error":  errorToString(err),
		}
	default:
		result = map[string]interface{}{
			"action": action,
			"status": "error",
			"error":  "unknown command",
		}
	}

	// Send result back to master
	c.sendResult(result)
}

func (c *Client) setupEnvironment() error {
	// Get Desktop path (cross-platform)
	desktopPath, err := getDesktopPath()
	if err != nil {
		return fmt.Errorf("error getting desktop path: %v", err)
	}

	// Create DOMJudge folder
	domjudgePath := filepath.Join(desktopPath, "DOMJudge")
	fmt.Printf("Creating folder: %s\n", domjudgePath)

	err = os.MkdirAll(domjudgePath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error creating folder: %v", err)
	}

	fmt.Println("DOMJudge folder created successfully!")
	return nil
}

func (c *Client) openVSCodeAction() error {
	desktopPath, err := getDesktopPath()
	if err != nil {
		return err
	}

	domjudgePath := filepath.Join(desktopPath, "DOMJudge")
	fmt.Println("Opening VS Code...")

	err = openVSCode(domjudgePath)
	if err != nil {
		return fmt.Errorf("error opening VS Code: %v", err)
	}

	fmt.Println("VS Code opened successfully!")
	return nil
}

func (c *Client) openChromeAction() error {
	fmt.Println("Opening browser with multiple tabs...")
	urls := []string{
		"https://google.com",
		"https://github.com",
		"https://stackoverflow.com",
	}

	err := openBrowserWithTabs(urls)
	if err != nil {
		return fmt.Errorf("error opening browser: %v", err)
	}

	fmt.Println("Browser opened successfully with multiple tabs!")
	return nil
}

func (c *Client) sendResult(result map[string]interface{}) {
	msg := Message{
		Type:      "result",
		Data:      result,
		Timestamp: time.Now(),
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		log.Printf("Error sending result: %v", err)
	}
}

func (c *Client) sendStatus(status string) {
	msg := Message{
		Type: "status",
		Data: map[string]interface{}{
			"clientId": c.clientID,
			"status":   status,
		},
		Timestamp: time.Now(),
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		log.Printf("Error sending status: %v", err)
	}
}

func (c *Client) close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func generateClientID() string {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	return fmt.Sprintf("%s-%s-%d", runtime.GOOS, hostname, time.Now().Unix())
}

func errorToString(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
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

func main() {
	fmt.Printf("GradeKeeper Client (%s/%s)\n", runtime.GOOS, runtime.GOARCH)

	// Command line flags
	var serverURL = flag.String("server", "", "Master server WebSocket URL (e.g., ws://192.168.1.100:8080/ws)")
	var standalone = flag.Bool("standalone", false, "Run in standalone mode")
	flag.Parse()

	// If standalone mode or no server specified, run original functionality
	if *standalone || *serverURL == "" {
		fmt.Println("Running in standalone mode...")
		runStandalone()
		return
	}

	// Client mode - connect to master server
	fmt.Printf("Running in client mode, connecting to: %s\n", *serverURL)

	client := NewClient(*serverURL)
	err := client.connect()
	if err != nil {
		log.Fatalf("Failed to connect to master server: %v", err)
	}
	defer client.close()

	// Send initial status
	client.sendStatus("connected")

	// Handle interrupt signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Start listening for messages
	go client.listen()

	// Keep client running
	for {
		select {
		case <-client.done:
			log.Println("Connection closed")
			return
		case <-interrupt:
			log.Println("Interrupt received, closing connection...")
			client.sendStatus("disconnecting")
			client.close()
			return
		}
	}
}

func runStandalone() {
	// Cross-platform standalone functionality
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