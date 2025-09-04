package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gorilla/websocket"
	"gradekeeper/internal/config"
	"gradekeeper/internal/platform"
)

const (
	// Heartbeat configuration - should match server settings
	HeartbeatInterval = 30 * time.Second
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorBold   = "\033[1m"
	ColorDim    = "\033[2m"
)

// Beautiful logging functions
func logInfo(format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("%s[%s]%s %s%s%s %s\n", 
		ColorDim, timestamp, ColorReset,
		ColorBlue, "ℹ", ColorReset,
		fmt.Sprintf(format, args...))
}

func logSuccess(format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("%s[%s]%s %s%s%s %s\n", 
		ColorDim, timestamp, ColorReset,
		ColorGreen, "✓", ColorReset,
		fmt.Sprintf(format, args...))
}

func logWarning(format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("%s[%s]%s %s%s%s %s\n", 
		ColorDim, timestamp, ColorReset,
		ColorYellow, "⚠", ColorReset,
		fmt.Sprintf(format, args...))
}

func logError(format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("%s[%s]%s %s%s%s %s\n", 
		ColorDim, timestamp, ColorReset,
		ColorRed, "✗", ColorReset,
		fmt.Sprintf(format, args...))
}

func logDebug(format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("%s[%s]%s %s%s%s %s\n", 
		ColorDim, timestamp, ColorReset,
		ColorPurple, "🔧", ColorReset,
		fmt.Sprintf(format, args...))
}

func logHeartbeat() {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("%s[%s]%s %s%s%s %sHeartbeat sent%s\n", 
		ColorDim, timestamp, ColorReset,
		ColorCyan, "💓", ColorReset,
		ColorDim, ColorReset)
}

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
	conn          *websocket.Conn
	serverURL     string
	clientID      string
	done          chan struct{}
	reconnect     chan struct{}
	shutdown      chan struct{}
	retrying      bool
	shouldNotReconnect bool
}

func NewClient(serverURL string) *Client {
	return &Client{
		serverURL: serverURL,
		clientID:  generateClientID(),
		done:      make(chan struct{}),
		reconnect: make(chan struct{}),
		shutdown:  make(chan struct{}),
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
	logSuccess("Connected to master server as client: %s", c.clientID)
	return nil
}

func (c *Client) connectWithRetry() {
	backoff := time.Second
	maxBackoff := 30 * time.Second

	for {
		// Check if shutdown was requested before attempting connection
		select {
		case <-c.shutdown:
			logInfo("Shutdown requested, stopping connection attempts...")
			return
		default:
			// Continue with connection attempt
		}

		c.retrying = true
		err := c.connect()
		if err != nil {
			logWarning("Connection failed: %v. Retrying in %v...", err, backoff)

			// Use select to check for shutdown during sleep
			select {
			case <-time.After(backoff):
				// Continue with retry
			case <-c.shutdown:
				logInfo("Shutdown requested during retry, exiting...")
				return
			}

			// Exponential backoff with max limit
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		// Successfully connected
		c.retrying = false
		c.sendStatus("connected")

		// Start listening for messages
		go c.listen()
		
		// Start heartbeat
		go c.startHeartbeat()
		break
	}
}

func (c *Client) listen() {
	for {
		// Check if shutdown was requested
		select {
		case <-c.shutdown:
			logInfo("Shutdown requested, stopping message listener...")
			return
		default:
			// Continue with message reading
		}

		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			logError("WebSocket connection lost: %v", err)
			
			// Check if we're shutting down before attempting reconnect
			select {
			case <-c.shutdown:
				logInfo("Shutdown in progress, not triggering reconnect...")
				return
			default:
				if !c.retrying && !c.shouldNotReconnect {
					select {
					case c.reconnect <- struct{}{}:
						// Successfully sent reconnect signal
					case <-c.shutdown:
						// Shutdown requested while trying to signal reconnect
						return
					}
				}
				return
			}
		}

		c.handleMessage(msg)
	}
}

func (c *Client) handleMessage(msg Message) {
	logDebug("Received message: %s", msg.Type)

	switch msg.Type {
	case "welcome":
		logSuccess("Welcome message received from master")
	case "error":
		c.handleError(msg)
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

func (c *Client) handleError(msg Message) {
	if errorData, ok := msg.Data.(map[string]interface{}); ok {
		errorType := errorData["error"].(string)
		errorMessage := errorData["message"].(string)
		
		logError("Error from master: %s - %s", errorType, errorMessage)
		
		if errorType == "duplicate_connection" {
			fmt.Printf("\n%s%s━━━ DUPLICATE CONNECTION ERROR ━━━%s\n", ColorRed, ColorBold, ColorReset)
			fmt.Printf("%s%s%s %s\n", ColorRed, "✗", ColorReset, errorMessage)
			fmt.Printf("%s%s%s Another instance of this client is already connected to the master server.\n", ColorYellow, "⚠", ColorReset)
			fmt.Printf("%s%s%s Please stop the other instance before running this client.\n", ColorBlue, "ℹ", ColorReset)
			
			// Set flag to prevent reconnection and exit
			c.shouldNotReconnect = true
			os.Exit(1)
		}
		
		// Handle other error types here in the future
		logWarning("Unhandled error type: %s", errorType)
	}
}

func (c *Client) executeCommand(action string) {
	logInfo("Executing command: %s", action)

	// Send "started" status
	c.sendActionStatus(action, "running", "")

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
	case "setupAll":
		err = c.setupAllAction()
		result = map[string]interface{}{
			"action": action,
			"status": "completed",
			"error":  errorToString(err),
		}
	case "clear":
		err = c.clearEnvironmentAction()
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

	// Send completion status back to master
	if result["status"] == "error" {
		c.sendActionStatus(action, "failed", result["error"].(string))
	} else {
		errorStr := ""
		if result["error"] != nil && result["error"].(string) != "" {
			errorStr = result["error"].(string)
		}
		if errorStr != "" {
			c.sendActionStatus(action, "failed", errorStr)
		} else {
			c.sendActionStatus(action, "success", "")
		}
	}
}

func (c *Client) setupEnvironment() error {
	// Get Desktop path (cross-platform)
	desktopPath, err := platform.GetDesktopPath()
	if err != nil {
		return fmt.Errorf("error getting desktop path: %v", err)
	}

	// Create DOMJudge folder
	domjudgePath := filepath.Join(desktopPath, "DOMJudge")
	logInfo("Creating folder: %s", domjudgePath)

	err = os.MkdirAll(domjudgePath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error creating folder: %v", err)
	}

	logSuccess("DOMJudge folder created successfully!")
	return nil
}

func (c *Client) openVSCodeAction() error {
	desktopPath, err := platform.GetDesktopPath()
	if err != nil {
		return err
	}

	domjudgePath := filepath.Join(desktopPath, "DOMJudge")
	logInfo("Opening VS Code...")

	err = platform.OpenVSCode(domjudgePath)
	if err != nil {
		return fmt.Errorf("error opening VS Code: %v", err)
	}

	logSuccess("VS Code opened successfully!")
	return nil
}

func (c *Client) openChromeAction() error {
	logInfo("Opening browser with multiple tabs...")
	
	err := platform.OpenBrowserWithTabs(config.DefaultURLs)
	if err != nil {
		return fmt.Errorf("error opening browser: %v", err)
	}

	logSuccess("Browser opened successfully with multiple tabs in incognito mode!")
	return nil
}

func (c *Client) setupAllAction() error {
	logInfo("Starting complete environment setup...")

	// Step 1: Setup environment (create DOMJudge folder)
	logInfo("Creating DOMJudge folder...")
	err := c.setupEnvironment()
	if err != nil {
		return fmt.Errorf("setup failed: %v", err)
	}

	// Step 2: Open VS Code
	logInfo("Opening VS Code...")
	err = c.openVSCodeAction()
	if err != nil {
		logWarning("VS Code opening failed: %v", err)
		// Don't return error, continue with browser
	}

	// Step 3: Open browser
	logInfo("Opening browser with multiple tabs...")
	err = c.openChromeAction()
	if err != nil {
		logWarning("Browser opening failed: %v", err)
		// Don't return error, setup is mostly complete
	}

	logSuccess("Complete environment setup finished!")
	return nil
}

func (c *Client) clearEnvironmentAction() error {
	logInfo("Clearing environment...")

	err := platform.ClearEnvironment()
	if err != nil {
		return fmt.Errorf("error clearing environment: %v", err)
	}

	logSuccess("Environment cleared successfully!")
	return nil
}

func (c *Client) sendResult(result map[string]interface{}) {
	// Check if connection exists
	if c.conn == nil {
		logWarning("Cannot send result: no connection")
		return
	}

	msg := Message{
		Type:      "result",
		Data:      result,
		Timestamp: time.Now(),
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		logError("Error sending result: %v", err)
		if !c.retrying && !c.shouldNotReconnect {
			select {
			case c.reconnect <- struct{}{}:
				// Successfully sent reconnect signal
			case <-c.shutdown:
				// Shutdown requested while trying to signal reconnect
				return
			}
		}
	}
}

func (c *Client) sendActionStatus(action, status, errorMsg string) {
	// Check if connection exists
	if c.conn == nil {
		logWarning("Cannot send action status: no connection")
		return
	}

	msg := Message{
		Type: "action_status",
		Data: map[string]interface{}{
			"clientId": c.clientID,
			"action":   action,
			"status":   status,
			"error":    errorMsg,
		},
		Timestamp: time.Now(),
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		logError("Error sending action status: %v", err)
		if !c.retrying && !c.shouldNotReconnect {
			select {
			case c.reconnect <- struct{}{}:
				// Successfully sent reconnect signal
			case <-c.shutdown:
				// Shutdown requested while trying to signal reconnect
				return
			}
		}
	}
}

func (c *Client) sendStatus(status string) {
	// Check if connection exists
	if c.conn == nil {
		logWarning("Cannot send status '%s': no connection", status)
		return
	}

	msg := Message{
		Type: "status",
		Data: map[string]interface{}{
			"clientId": c.clientID,
			"status":   status,
		},
		Timestamp: time.Now(),
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		logError("Error sending status: %v", err)
		if !c.retrying && !c.shouldNotReconnect {
			select {
			case c.reconnect <- struct{}{}:
				// Successfully sent reconnect signal
			case <-c.shutdown:
				// Shutdown requested while trying to signal reconnect
				return
			}
		}
	}
}

func (c *Client) startHeartbeat() {
	ticker := time.NewTicker(HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Send heartbeat message
			if c.conn != nil {
				msg := Message{
					Type: "heartbeat",
					Data: map[string]interface{}{
						"clientId": c.clientID,
						"timestamp": time.Now(),
					},
					Timestamp: time.Now(),
				}

				if err := c.conn.WriteJSON(msg); err != nil {
					logError("Error sending heartbeat: %v", err)
					return
				}
			}
		case <-c.shutdown:
			return
		case <-c.done:
			return
		}
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
	return fmt.Sprintf("%s-%s", runtime.GOOS, hostname)
}

func errorToString(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

func main() {
	fmt.Printf("GradeKeeper Client (%s/%s)\n", runtime.GOOS, runtime.GOARCH)

	// Command line flags
	var serverURL = flag.String("server", "", "Master server WebSocket URL (e.g., ws://192.168.1.100:8080/ws)")
	var standalone = flag.Bool("standalone", false, "Run in standalone mode")
	var clear = flag.Bool("clear", false, "Clear environment (remove DOMJudge folder and close applications)")
	flag.Parse()

	// If clear flag is set, run clear environment and exit
	if *clear {
		logInfo("Running in clear mode...")
		runClear()
		return
	}

	// If standalone mode or no server specified, run original functionality
	if *standalone || *serverURL == "" {
		fmt.Println("Running in standalone mode...")
		runStandalone()
		return
	}

	// Client mode - connect to master server
	fmt.Printf("Running in client mode, connecting to: %s\n", *serverURL)

	client := NewClient(*serverURL)
	defer client.close()

	// Handle interrupt signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Initial connection
	go client.connectWithRetry()

	// Keep client running with auto-reconnect
	for {
		select {
		case <-client.reconnect:
			logWarning("Connection lost, attempting to reconnect...")
			go client.connectWithRetry()
		case <-interrupt:
			logInfo("Interrupt received, closing connection...")
			client.retrying = true
			
			// Signal all goroutines to shutdown
			close(client.shutdown)

			// Try to send disconnecting status with timeout
			done := make(chan struct{})
			go func() {
				client.sendStatus("disconnecting")
				close(done)
			}()

			select {
			case <-done:
				// Status sent successfully
			case <-time.After(2 * time.Second):
				// Timeout, proceed with shutdown
				logWarning("Timeout sending disconnect status, forcing shutdown...")
			}

			client.close()
			logSuccess("Client shutdown complete.")
			return
		}
	}
}

func runStandalone() {
	// Handle interrupt signal for standalone mode
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Run standalone operations in a goroutine so we can handle interrupts
	done := make(chan bool, 1)

	go func() {
		// Cross-platform standalone functionality
		desktopPath, err := platform.GetDesktopPath()
		if err != nil {
			logError("Error getting desktop path: %v", err)
			done <- false
			return
		}

		// Create DOMJudge folder
		domjudgePath := filepath.Join(desktopPath, "DOMJudge")
		logInfo("Creating folder: %s", domjudgePath)

		err = os.MkdirAll(domjudgePath, os.ModePerm)
		if err != nil {
			logError("Error creating folder: %v", err)
			done <- false
			return
		}
		logSuccess("DOMJudge folder created successfully!")

		// Open VS Code with the folder
		logInfo("Opening VS Code...")
		err = platform.OpenVSCode(domjudgePath)
		if err != nil {
			logError("Error opening VS Code: %v", err)
		} else {
			logSuccess("VS Code opened successfully!")
		}

		// Open browser with multiple tabs
		logInfo("Opening browser with multiple tabs...")
		err = platform.OpenBrowserWithTabs(config.DefaultURLs)
		if err != nil {
			logError("Error opening browser: %v", err)
		} else {
			logSuccess("Browser opened successfully with multiple tabs in incognito mode on %s!", runtime.GOOS)
		}

		logSuccess("All tasks completed!")
		done <- true
	}()

	// Wait for completion or interrupt
	select {
	case success := <-done:
		if !success {
			os.Exit(1)
		}
	case <-interrupt:
		logInfo("\nInterrupt received, exiting standalone mode...")
		os.Exit(0)
	}
}

func runClear() {
	// Handle interrupt signal for clear mode
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Run clear operation in a goroutine so we can handle interrupts
	done := make(chan bool, 1)

	go func() {
		// Use the clearEnvironmentAction method
		client := &Client{} // Create empty client just to use the method
		err := client.clearEnvironmentAction()
		if err != nil {
			logError("Clear operation failed: %v", err)
			done <- false
			return
		}

		logSuccess("Clear operation completed successfully!")
		done <- true
	}()

	// Wait for completion or interrupt
	select {
	case success := <-done:
		if !success {
			os.Exit(1)
		}
	case <-interrupt:
		logInfo("\nInterrupt received, exiting clear mode...")
		os.Exit(0)
	}
}
