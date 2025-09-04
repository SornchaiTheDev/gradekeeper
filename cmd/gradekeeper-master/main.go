package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"gradekeeper/internal/templates"
)

const (
	// Heartbeat configuration
	HeartbeatInterval = 30 * time.Second // How often clients should send heartbeat
	HeartbeatTimeout  = 90 * time.Second // How long to wait before marking client as disconnected
)

type Message struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

type Command struct {
	Action string `json:"action"`
	Target string `json:"target,omitempty"` // "all" or specific client ID
}

type ClientInfo struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Status        string    `json:"status"`
	LastSeen      time.Time `json:"lastSeen"`
	FirstSeen     time.Time `json:"firstSeen"`
	LastHeartbeat time.Time `json:"lastHeartbeat"`
	Action        string    `json:"action"`        // Current/last action: "setup", "setupAll", "clear", etc.
	ActionStatus  string    `json:"actionStatus"` // "running", "success", "failed"
	ActionError   string    `json:"actionError"`  // Error message if failed
}

type Master struct {
	clients           map[string]*websocket.Conn
	clientsInfo       map[string]*ClientInfo
	dashboardConns    map[*websocket.Conn]bool
	clientsMu         sync.RWMutex
	dashboardMu       sync.RWMutex
	upgrader          websocket.Upgrader
	dashboardSecret   string
	storageFile       string
	dashboardTemplate *templates.Dashboard
}

func NewMaster() *Master {
	// Initialize dashboard template
	dashboardTemplate, err := templates.NewDashboard()
	if err != nil {
		log.Fatalf("Failed to initialize dashboard template: %v", err)
	}

	m := &Master{
		clients:         make(map[string]*websocket.Conn),
		clientsInfo:     make(map[string]*ClientInfo),
		dashboardConns:  make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
		dashboardSecret:   generateRandomSecret(),
		storageFile:       "gradekeeper-clients.json",
		dashboardTemplate: dashboardTemplate,
	}
	
	// Load existing client data
	m.loadClientData()
	
	// Start heartbeat monitor
	go m.monitorHeartbeats()
	
	return m
}

func (m *Master) loadClientData() {
	data, err := os.ReadFile(m.storageFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error reading client data: %v", err)
		}
		return
	}

	var clients []ClientInfo
	if err := json.Unmarshal(data, &clients); err != nil {
		log.Printf("Error parsing client data: %v", err)
		return
	}

	for _, client := range clients {
		clientInfo := client
		clientInfo.Status = "disconnected" // All clients start as disconnected
		m.clientsInfo[client.ID] = &clientInfo
	}

	log.Printf("Loaded %d client records from storage", len(clients))
}

func (m *Master) saveClientData() {
	m.clientsMu.RLock()
	clients := make([]ClientInfo, 0, len(m.clientsInfo))
	for _, client := range m.clientsInfo {
		clients = append(clients, *client)
	}
	m.clientsMu.RUnlock()

	data, err := json.MarshalIndent(clients, "", "  ")
	if err != nil {
		log.Printf("Error marshaling client data: %v", err)
		return
	}

	if err := os.WriteFile(m.storageFile, data, 0644); err != nil {
		log.Printf("Error saving client data: %v", err)
		return
	}
}

func (m *Master) monitorHeartbeats() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		m.clientsMu.Lock()
		
		var disconnectedClients []string
		for clientID, clientInfo := range m.clientsInfo {
			// Check if client is supposed to be connected but hasn't sent heartbeat recently
			if clientInfo.Status == "connected" {
				if now.Sub(clientInfo.LastHeartbeat) > HeartbeatTimeout {
					// Client missed heartbeat deadline
					clientInfo.Status = "disconnected"
					clientInfo.LastSeen = now
					disconnectedClients = append(disconnectedClients, clientID)
					
					// Also remove from active connections if present
					if conn, exists := m.clients[clientID]; exists {
						conn.Close()
						delete(m.clients, clientID)
					}
				}
			}
		}
		
		m.clientsMu.Unlock()
		
		// Log and notify dashboard of disconnected clients
		for _, clientID := range disconnectedClients {
			log.Printf("Client %s marked as disconnected due to heartbeat timeout", clientID)
			
			// Notify dashboards
			m.broadcastToDashboard(Message{
				Type: "client-disconnected",
				Data: map[string]interface{}{
					"clientId":     clientID,
					"reason":       "heartbeat_timeout",
					"totalClients": len(m.clients),
				},
				Timestamp: now,
			})
		}
		
		if len(disconnectedClients) > 0 {
			m.saveClientData()
		}
	}
}

func (m *Master) cleanup() {
	log.Println("Cleaning up...")
	
	// Clear the clients storage file
	if err := os.Remove(m.storageFile); err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: Could not remove client storage file: %v", err)
	} else if err == nil {
		log.Println("Client storage file cleared successfully")
	}
	
	// Close all client connections
	m.clientsMu.Lock()
	for clientID, conn := range m.clients {
		conn.Close()
		log.Printf("Closed connection to client: %s", clientID)
	}
	m.clientsMu.Unlock()
	
	// Close all dashboard connections
	m.dashboardMu.Lock()
	for conn := range m.dashboardConns {
		conn.Close()
	}
	m.dashboardMu.Unlock()
	
	log.Println("Cleanup completed")
}

func (m *Master) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	clientID := r.Header.Get("X-Client-ID")
	// Check for dashboard authentication via query parameter
	dashboardAuth := r.URL.Query().Get("dashboard")

	// Check if this is a dashboard connection attempt
	if dashboardAuth != "" {
		if dashboardAuth == m.dashboardSecret {
		// This is a dashboard connection
		m.dashboardMu.Lock()
		m.dashboardConns[conn] = true
		m.dashboardMu.Unlock()

		log.Println("Dashboard connected")

		// Send welcome message for dashboard
		welcomeMsg := Message{
			Type:      "dashboard-welcome",
			Data:      map[string]string{"type": "dashboard"},
			Timestamp: time.Now(),
		}
		conn.WriteJSON(welcomeMsg)

		// Handle messages from dashboard (if any)
		for {
			var msg Message
			err := conn.ReadJSON(&msg)
			if err != nil {
				log.Printf("Dashboard disconnected: %v", err)
				break
			}
			// Dashboard messages can be handled here if needed
		}

		// Remove dashboard connection on disconnect
		m.dashboardMu.Lock()
		delete(m.dashboardConns, conn)
		m.dashboardMu.Unlock()
		return
		} else {
			// Invalid dashboard authentication
			log.Printf("Dashboard connection with invalid authentication: %s", dashboardAuth)
			conn.Close()
			return
		}
	}

	// This should be a client connection - require X-Client-ID
	if clientID == "" {
		log.Printf("WebSocket connection rejected: no X-Client-ID header and no dashboard authentication")
		conn.Close()
		return
	}

	m.clientsMu.Lock()
	
	// Check if client is already connected
	if _, exists := m.clients[clientID]; exists {
		log.Printf("Client %s attempted to connect but is already connected, rejecting new connection", clientID)
		m.clientsMu.Unlock()
		
		// Send rejection message before closing
		rejectMsg := Message{
			Type: "error",
			Data: map[string]interface{}{
				"error": "duplicate_connection",
				"message": "A connection with this client ID already exists",
			},
			Timestamp: time.Now(),
		}
		conn.WriteJSON(rejectMsg)
		conn.Close()
		return
	}
	
	m.clients[clientID] = conn
	
	// Update or create client info
	now := time.Now()
	if clientInfo, exists := m.clientsInfo[clientID]; exists {
		// Existing client reconnected
		clientInfo.Status = "connected"
		clientInfo.LastSeen = now
		clientInfo.LastHeartbeat = now
	} else {
		// New client
		m.clientsInfo[clientID] = &ClientInfo{
			ID:            clientID,
			Name:          fmt.Sprintf("Client-%s", clientID[:8]),
			Status:        "connected",
			LastSeen:      now,
			FirstSeen:     now,
			LastHeartbeat: now,
		}
	}
	m.clientsMu.Unlock()
	
	// Save updated client data
	m.saveClientData()

	log.Printf("Client %s connected", clientID)

	// Notify dashboards about new client
	m.broadcastToDashboard(Message{
		Type: "client-connected",
		Data: map[string]interface{}{
			"clientId": clientID,
			"totalClients": len(m.clients),
		},
		Timestamp: time.Now(),
	})

	// Send welcome message
	welcomeMsg := Message{
		Type:      "welcome",
		Data:      map[string]string{"clientId": clientID},
		Timestamp: time.Now(),
	}
	conn.WriteJSON(welcomeMsg)

	// Handle messages from client
	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("Client %s disconnected: %v", clientID, err)
			break
		}

		m.handleClientMessage(clientID, msg)
	}

	// Mark client as disconnected
	m.clientsMu.Lock()
	delete(m.clients, clientID)
	if clientInfo, exists := m.clientsInfo[clientID]; exists {
		clientInfo.Status = "disconnected"
		clientInfo.LastSeen = time.Now()
	}
	clientCount := len(m.clients)
	m.clientsMu.Unlock()
	
	// Save updated client data
	m.saveClientData()

	// Notify dashboards about client disconnect
	m.broadcastToDashboard(Message{
		Type: "client-disconnected",
		Data: map[string]interface{}{
			"clientId": clientID,
			"totalClients": clientCount,
		},
		Timestamp: time.Now(),
	})
}

func (m *Master) handleClientMessage(clientID string, msg Message) {
	log.Printf("Received from %s: %+v", clientID, msg)

	switch msg.Type {
	case "status":
		// Client sending status update
		log.Printf("Client %s status: %v", clientID, msg.Data)
	case "result":
		// Client sending command execution result
		log.Printf("Client %s result: %v", clientID, msg.Data)
	case "action_status":
		// Client sending action status update
		m.handleActionStatus(clientID, msg.Data)
	case "heartbeat":
		// Client sending heartbeat - update last heartbeat time
		m.clientsMu.Lock()
		if clientInfo, exists := m.clientsInfo[clientID]; exists {
			clientInfo.LastHeartbeat = time.Now()
			clientInfo.LastSeen = time.Now()
			if clientInfo.Status != "connected" {
				clientInfo.Status = "connected"
				log.Printf("Client %s marked as connected via heartbeat", clientID)
			}
		}
		m.clientsMu.Unlock()
		m.saveClientData()
	}
}

func (m *Master) handleActionStatus(clientID string, data interface{}) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		log.Printf("Invalid action status data from client %s", clientID)
		return
	}

	action, _ := dataMap["action"].(string)
	status, _ := dataMap["status"].(string)
	errorMsg, _ := dataMap["error"].(string)

	log.Printf("Client %s action status: %s -> %s", clientID, action, status)

	m.clientsMu.Lock()
	if clientInfo, exists := m.clientsInfo[clientID]; exists {
		clientInfo.Action = action
		clientInfo.ActionStatus = status
		clientInfo.ActionError = errorMsg
		clientInfo.LastSeen = time.Now()
	}
	m.clientsMu.Unlock()

	// Save updated client data
	m.saveClientData()

	// Broadcast to dashboards for real-time updates
	m.broadcastToDashboard(Message{
		Type: "client_action_update",
		Data: map[string]interface{}{
			"clientId": clientID,
			"action":   action,
			"status":   status,
			"error":    errorMsg,
		},
		Timestamp: time.Now(),
	})
}

func (m *Master) broadcastCommand(cmd Command) {
	message := Message{
		Type:      "command",
		Data:      cmd,
		Timestamp: time.Now(),
	}

	m.clientsMu.RLock()
	defer m.clientsMu.RUnlock()

	if cmd.Target == "all" || cmd.Target == "" {
		// Broadcast to all clients
		for clientID, conn := range m.clients {
			if err := conn.WriteJSON(message); err != nil {
				log.Printf("Error sending to client %s: %v", clientID, err)
			}
		}
	} else {
		// Send to specific client
		if conn, exists := m.clients[cmd.Target]; exists {
			if err := conn.WriteJSON(message); err != nil {
				log.Printf("Error sending to client %s: %v", cmd.Target, err)
			}
		}
	}

	// Also broadcast command info to dashboards
	m.broadcastToDashboard(Message{
		Type: "command-sent",
		Data: map[string]interface{}{
			"action": cmd.Action,
			"target": cmd.Target,
			"clientCount": len(m.clients),
		},
		Timestamp: time.Now(),
	})
}

func (m *Master) broadcastToDashboard(msg Message) {
	m.dashboardMu.RLock()
	defer m.dashboardMu.RUnlock()

	for conn := range m.dashboardConns {
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("Error sending to dashboard: %v", err)
			// Remove failed connection
			delete(m.dashboardConns, conn)
		}
	}
}

func (m *Master) getAllClients() []ClientInfo {
	m.clientsMu.RLock()
	defer m.clientsMu.RUnlock()

	clients := make([]ClientInfo, 0, len(m.clientsInfo))
	for _, clientInfo := range m.clientsInfo {
		clients = append(clients, *clientInfo)
	}
	
	// Sort clients by ID for consistent ordering
	sort.Slice(clients, func(i, j int) bool {
		return clients[i].ID < clients[j].ID
	})
	
	return clients
}

func generateRandomSecret() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (m *Master) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	
	// Prepare template data
	data := templates.DashboardData{
		DashboardSecret: m.dashboardSecret,
	}
	
	// Render the dashboard template
	if err := m.dashboardTemplate.Render(w, data); err != nil {
		log.Printf("Error rendering dashboard template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (m *Master) handleAPICommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var cmd Command
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	m.broadcastCommand(cmd)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
}

func (m *Master) handleAPIClients(w http.ResponseWriter, r *http.Request) {
	clients := m.getAllClients()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clients)
}

func main() {
	master := NewMaster()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	http.HandleFunc("/", master.handleDashboard)
	http.HandleFunc("/ws", master.handleWebSocket)
	http.HandleFunc("/api/command", master.handleAPICommand)
	http.HandleFunc("/api/clients", master.handleAPIClients)

	fmt.Println("🎓 GradeKeeper Master Server starting...")
	fmt.Println("📊 Dashboard: http://localhost:8080")
	fmt.Println("🔌 WebSocket: ws://localhost:8080/ws")
	fmt.Printf("🔐 Dashboard Secret: %s\n", master.dashboardSecret)

	// Start the server in a goroutine
	server := &http.Server{Addr: ":8080"}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\n🛑 Shutdown signal received...")
	
	// Perform cleanup
	master.cleanup()
	
	fmt.Println("👋 GradeKeeper Master Server stopped gracefully")
}
