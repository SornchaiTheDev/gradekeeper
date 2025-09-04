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
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
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
}

func NewMaster() *Master {
	m := &Master{
		clients:         make(map[string]*websocket.Conn),
		clientsInfo:     make(map[string]*ClientInfo),
		dashboardConns:  make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
		dashboardSecret: generateRandomSecret(),
		storageFile:     "gradekeeper-clients.json",
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
	return clients
}

func generateRandomSecret() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (m *Master) handleDashboard(w http.ResponseWriter, r *http.Request) {
	// Get the dashboard secret for this session
	dashboardSecret := m.dashboardSecret
	
	// Build HTML with injected dashboard secret
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>GradeKeeper Master Dashboard</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <script src="https://cdn.tailwindcss.com"></script>
    <script src="https://unpkg.com/lucide@latest/dist/umd/lucide.js"></script>
    <script>
        tailwind.config = {
            theme: {
                extend: {
                    colors: {
                        primary: '#2563eb',
                        success: '#16a34a',
                        danger: '#dc2626',
                    }
                }
            }
        }
    </script>
</head>
<body class="bg-gray-50 min-h-screen">
    <div class="bg-primary text-white p-6 shadow-lg">
        <div class="flex items-center gap-3">
            <i data-lucide="graduation-cap" class="w-8 h-8"></i>
            <div>
                <h1 class="text-2xl font-bold">GradeKeeper Master Dashboard</h1>
                <p class="text-blue-100">Manage and control multiple GradeKeeper clients</p>
            </div>
        </div>
    </div>

    <div class="container mx-auto px-6 py-6">
        <div class="bg-white rounded-lg shadow-sm p-6 mb-6">
            <h2 class="text-lg font-semibold text-gray-800 mb-4 flex items-center gap-2">
                <i data-lucide="settings" class="w-5 h-5"></i>
                Global Controls
            </h2>
            <div class="flex flex-wrap gap-3">
                <button onclick="setupAll()" class="bg-success hover:bg-green-700 text-white font-semibold px-6 py-3 rounded-lg transition-colors duration-200 flex items-center gap-2 shadow-sm">
                    <i data-lucide="rocket" class="w-5 h-5"></i>
                    Setup All (Complete)
                </button>
                <button onclick="clearAll()" class="bg-danger hover:bg-red-700 text-white font-semibold px-6 py-3 rounded-lg transition-colors duration-200 flex items-center gap-2 shadow-sm">
                    <i data-lucide="trash-2" class="w-5 h-5"></i>
                    Clear All (Complete)
                </button>
                <button onclick="refreshClients()" class="bg-gray-600 hover:bg-gray-700 text-white px-4 py-3 rounded-lg transition-colors duration-200 flex items-center gap-2 shadow-sm">
                    <i data-lucide="refresh-cw" class="w-5 h-5"></i>
                    Refresh
                </button>
            </div>
        </div>

        <div class="bg-white rounded-lg shadow-sm p-6 mb-6">
            <h2 class="text-lg font-semibold text-gray-800 mb-4 flex items-center gap-2">
                <i data-lucide="monitor" class="w-5 h-5"></i>
                Connected Clients
            </h2>
            <div id="clients" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"></div>
        </div>

        <div class="bg-white rounded-lg shadow-sm p-6">
            <h2 class="text-lg font-semibold text-gray-800 mb-4 flex items-center gap-2">
                <i data-lucide="activity" class="w-5 h-5"></i>
                Activity Log
            </h2>
            <div id="log" class="bg-gray-900 text-green-400 p-4 h-64 overflow-y-auto rounded-lg font-mono text-sm"></div>
        </div>
    </div>

    <script>
        const dashboardSecret = '%s';
        let ws;
        
        // Initialize Lucide icons after DOM is loaded
        document.addEventListener('DOMContentLoaded', function() {
            lucide.createIcons();
        });
        
        function connect() {
            // Use query parameter for dashboard authentication
            ws = new WebSocket('ws://localhost:8080/ws?dashboard=' + dashboardSecret);
            
            ws.onopen = function() {
                log('Dashboard connected to master server');
                refreshClients();
            };
            
            ws.onmessage = function(event) {
                const data = JSON.parse(event.data);
                
                switch(data.type) {
                    case 'dashboard-welcome':
                        log('Dashboard authenticated successfully');
                        break;
                    case 'client-connected':
                        log('Client connected: ' + data.data.clientId + ' (Total: ' + data.data.totalClients + ')');
                        refreshClients();
                        break;
                    case 'client-disconnected':
                        log('Client disconnected: ' + data.data.clientId + ' (Total: ' + data.data.totalClients + ')');
                        refreshClients();
                        break;
                    case 'command-sent':
                        log('Command sent: ' + data.data.action + ' to ' + (data.data.target || 'all') + ' (' + data.data.clientCount + ' clients)');
                        break;
                    default:
                        log('Received: ' + JSON.stringify(data));
                }
            };
            
            ws.onclose = function() {
                log('Dashboard disconnected from server. Reconnecting...');
                setTimeout(connect, 1000);
            };
        }
        
        function sendCommand(action) {
            if (ws && ws.readyState === WebSocket.OPEN) {
                const command = { action: action, target: 'all' };
                fetch('/api/command', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(command)
                });
                log('Sent command: ' + action + ' to all clients');
            }
        }
        
        function refreshClients() {
            fetch('/api/clients')
                .then(response => response.json())
                .then(clients => {
                    const container = document.getElementById('clients');
                    container.innerHTML = clients.map(client => {
                        const isConnected = client.status === 'connected';
                        const statusColor = isConnected ? 'border-l-success' : 'border-l-gray-400';
                        const statusBadgeColor = isConnected ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-600';
                        const lastSeenTime = new Date(client.lastSeen).toLocaleString();
                        const firstSeenTime = new Date(client.firstSeen).toLocaleString();
                        
                        return '<div class="bg-gray-50 border border-gray-200 rounded-lg p-4 border-l-4 ' + statusColor + '">' +
                        '<div class="flex items-center gap-2 mb-2">' +
                        '<i data-lucide="monitor" class="w-4 h-4 text-gray-600"></i>' +
                        '<h3 class="font-semibold text-gray-800">' + client.name + '</h3>' +
                        '</div>' +
                        '<p class="text-sm text-gray-600 mb-1">ID: <code class="bg-gray-200 px-1 rounded text-xs">' + client.id + '</code></p>' +
                        '<p class="text-sm text-gray-600 mb-1">Status: <span class="inline-flex items-center px-2 py-1 rounded-full text-xs ' + statusBadgeColor + '">' + client.status + '</span></p>' +
                        '<p class="text-sm text-gray-600 mb-1">Last Seen: <span class="text-xs">' + lastSeenTime + '</span></p>' +
                        (client.firstSeen ? '<p class="text-sm text-gray-600 mb-4">First Seen: <span class="text-xs">' + firstSeenTime + '</span></p>' : '<div class="mb-4"></div>') +
                        '<div class="space-y-3">' +
                        '<div class="flex gap-2">' +
                        '<button onclick="setupAllForClient(\'' + client.id + '\')" ' + (isConnected ? '' : 'disabled ') + 'class="' + (isConnected ? 'bg-success hover:bg-green-700 text-white' : 'bg-gray-300 text-gray-500 cursor-not-allowed') + ' text-sm px-3 py-2 rounded-md transition-colors duration-200 flex items-center gap-1 flex-1">' +
                        '<i data-lucide="rocket" class="w-4 h-4"></i>Setup All</button>' +
                        '<button onclick="clearAllForClient(\'' + client.id + '\')" ' + (isConnected ? '' : 'disabled ') + 'class="' + (isConnected ? 'bg-danger hover:bg-red-700 text-white' : 'bg-gray-300 text-gray-500 cursor-not-allowed') + ' text-sm px-3 py-2 rounded-md transition-colors duration-200 flex items-center gap-1 flex-1">' +
                        '<i data-lucide="trash-2" class="w-4 h-4"></i>Clear All</button>' +
                        '</div>' +
                        '<div class="grid grid-cols-2 gap-2">' +
                        '<button onclick="sendCommandToClient(\'' + client.id + '\', \'setup\')" ' + (isConnected ? '' : 'disabled ') + 'class="' + (isConnected ? 'bg-blue-600 hover:bg-blue-700 text-white' : 'bg-gray-300 text-gray-500 cursor-not-allowed') + ' text-sm px-3 py-2 rounded-md transition-colors duration-200 flex items-center gap-1 justify-center">' +
                        '<i data-lucide="folder-plus" class="w-4 h-4"></i>Setup</button>' +
                        '<button onclick="sendCommandToClient(\'' + client.id + '\', \'open-vscode\')" ' + (isConnected ? '' : 'disabled ') + 'class="' + (isConnected ? 'bg-blue-600 hover:bg-blue-700 text-white' : 'bg-gray-300 text-gray-500 cursor-not-allowed') + ' text-sm px-3 py-2 rounded-md transition-colors duration-200 flex items-center gap-1 justify-center">' +
                        '<i data-lucide="code" class="w-4 h-4"></i>VS Code</button>' +
                        '<button onclick="sendCommandToClient(\'' + client.id + '\', \'open-chrome\')" ' + (isConnected ? '' : 'disabled ') + 'class="' + (isConnected ? 'bg-blue-600 hover:bg-blue-700 text-white' : 'bg-gray-300 text-gray-500 cursor-not-allowed') + ' text-sm px-3 py-2 rounded-md transition-colors duration-200 flex items-center gap-1 justify-center">' +
                        '<i data-lucide="globe" class="w-4 h-4"></i>Chrome</button>' +
                        '<button onclick="sendCommandToClient(\'' + client.id + '\', \'clear\')" ' + (isConnected ? '' : 'disabled ') + 'class="' + (isConnected ? 'bg-gray-600 hover:bg-gray-700 text-white' : 'bg-gray-300 text-gray-500 cursor-not-allowed') + ' text-sm px-3 py-2 rounded-md transition-colors duration-200 flex items-center gap-1 justify-center">' +
                        '<i data-lucide="x" class="w-4 h-4"></i>Clear</button>' +
                        '</div>' +
                        '</div>' +
                        '</div>';
                    }).join('');
                    // Re-initialize Lucide icons for dynamically added content
                    lucide.createIcons();
                });
        }
        
        function sendCommandToClient(clientId, action) {
            const command = { action: action, target: clientId };
            fetch('/api/command', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(command)
            });
            log('Sent command: ' + action + ' to client ' + clientId);
        }
        
        function setupAll() {
            log('🚀 Starting complete setup for all clients...');
            const commands = ['setup', 'open-vscode', 'open-chrome'];
            let currentCommand = 0;
            
            function executeNext() {
                if (currentCommand < commands.length) {
                    const action = commands[currentCommand];
                    sendCommand(action);
                    currentCommand++;
                    
                    // Wait a bit between commands to avoid overwhelming clients
                    setTimeout(executeNext, 1000);
                } else {
                    log('✅ Complete setup finished for all clients!');
                }
            }
            
            executeNext();
        }
        
        function clearAll() {
            if (confirm('⚠️ This will clear all environments on all clients. Are you sure?')) {
                log('💥 Starting complete clear for all clients...');
                sendCommand('clear');
                log('✅ Clear command sent to all clients!');
            }
        }
        
        function setupAllForClient(clientId) {
            log('🚀 Starting complete setup for client ' + clientId + '...');
            const commands = ['setup', 'open-vscode', 'open-chrome'];
            let currentCommand = 0;
            
            function executeNext() {
                if (currentCommand < commands.length) {
                    const action = commands[currentCommand];
                    sendCommandToClient(clientId, action);
                    currentCommand++;
                    
                    // Wait a bit between commands
                    setTimeout(executeNext, 1000);
                } else {
                    log('✅ Complete setup finished for client ' + clientId + '!');
                }
            }
            
            executeNext();
        }
        
        function clearAllForClient(clientId) {
            if (confirm('⚠️ This will clear the environment on client ' + clientId + '. Are you sure?')) {
                log('💥 Starting complete clear for client ' + clientId + '...');
                sendCommandToClient(clientId, 'clear');
                log('✅ Clear command sent to client ' + clientId + '!');
            }
        }
        
        function log(message) {
            const logEl = document.getElementById('log');
            const timestamp = new Date().toLocaleTimeString();
            logEl.innerHTML += '<div><span class="text-cyan-400">[' + timestamp + ']</span> ' + message + '</div>';
            logEl.scrollTop = logEl.scrollHeight;
        }
        
        connect();
        setInterval(refreshClients, 5000); // Refresh every 5 seconds
    </script>
</body>
</html>`, dashboardSecret)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
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
