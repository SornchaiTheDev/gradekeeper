package main

import (
	"flag"
	"fmt"
	"log"
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
	fmt.Printf("Connected to master server as client: %s\n", c.clientID)
	return nil
}

func (c *Client) connectWithRetry() {
	backoff := time.Second
	maxBackoff := 30 * time.Second

	for {
		// Check if shutdown was requested before attempting connection
		select {
		case <-c.shutdown:
			log.Println("Shutdown requested, stopping connection attempts...")
			return
		default:
			// Continue with connection attempt
		}

		c.retrying = true
		err := c.connect()
		if err != nil {
			log.Printf("Connection failed: %v. Retrying in %v...", err, backoff)

			// Use select to check for shutdown during sleep
			select {
			case <-time.After(backoff):
				// Continue with retry
			case <-c.shutdown:
				log.Println("Shutdown requested during retry, exiting...")
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
			log.Println("Shutdown requested, stopping message listener...")
			return
		default:
			// Continue with message reading
		}

		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			
			// Check if we're shutting down before attempting reconnect
			select {
			case <-c.shutdown:
				log.Println("Shutdown in progress, not triggering reconnect...")
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
	log.Printf("Received message: %+v", msg)

	switch msg.Type {
	case "welcome":
		fmt.Println("Welcome message received from master")
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
		
		log.Printf("Received error from master: %s - %s", errorType, errorMessage)
		
		if errorType == "duplicate_connection" {
			fmt.Printf("Error: %s\n", errorMessage)
			fmt.Println("Another instance of this client is already connected to the master server.")
			fmt.Println("Please stop the other instance before running this client.")
			
			// Set flag to prevent reconnection and exit
			c.shouldNotReconnect = true
			os.Exit(1)
		}
		
		// Handle other error types here in the future
		log.Printf("Unhandled error type: %s", errorType)
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

	// Send result back to master
	c.sendResult(result)
}

func (c *Client) setupEnvironment() error {
	// Get Desktop path (cross-platform)
	desktopPath, err := platform.GetDesktopPath()
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
	desktopPath, err := platform.GetDesktopPath()
	if err != nil {
		return err
	}

	domjudgePath := filepath.Join(desktopPath, "DOMJudge")
	fmt.Println("Opening VS Code...")

	err = platform.OpenVSCode(domjudgePath)
	if err != nil {
		return fmt.Errorf("error opening VS Code: %v", err)
	}

	fmt.Println("VS Code opened successfully!")
	return nil
}

func (c *Client) openChromeAction() error {
	fmt.Println("Opening browser with multiple tabs...")
	
	err := platform.OpenBrowserWithTabs(config.DefaultURLs)
	if err != nil {
		return fmt.Errorf("error opening browser: %v", err)
	}

	fmt.Println("Browser opened successfully with multiple tabs in incognito mode!")
	return nil
}

func (c *Client) clearEnvironmentAction() error {
	fmt.Println("Clearing environment...")

	err := platform.ClearEnvironment()
	if err != nil {
		return fmt.Errorf("error clearing environment: %v", err)
	}

	fmt.Println("Environment cleared successfully!")
	return nil
}

func (c *Client) sendResult(result map[string]interface{}) {
	// Check if connection exists
	if c.conn == nil {
		log.Printf("Cannot send result: no connection")
		return
	}

	msg := Message{
		Type:      "result",
		Data:      result,
		Timestamp: time.Now(),
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		log.Printf("Error sending result: %v", err)
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
		log.Printf("Cannot send status '%s': no connection", status)
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
		log.Printf("Error sending status: %v", err)
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
					log.Printf("Error sending heartbeat: %v", err)
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
			log.Println("Connection lost, attempting to reconnect...")
			go client.connectWithRetry()
		case <-interrupt:
			log.Println("Interrupt received, closing connection...")
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
				log.Println("Timeout sending disconnect status, forcing shutdown...")
			}

			client.close()
			log.Println("Client shutdown complete.")
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
		err = platform.OpenBrowserWithTabs(config.DefaultURLs)
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
		fmt.Println("\nInterrupt received, exiting standalone mode...")
		os.Exit(0)
	}
}
