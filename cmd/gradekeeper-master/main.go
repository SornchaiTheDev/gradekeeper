package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
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
	Target string `json:"target,omitempty"` // "all" or specific client ID
}

type ClientInfo struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Status   string    `json:"status"`
	LastSeen time.Time `json:"lastSeen"`
}

type Master struct {
	clients         map[string]*websocket.Conn
	dashboardConns  map[*websocket.Conn]bool
	clientsMu       sync.RWMutex
	dashboardMu     sync.RWMutex
	upgrader        websocket.Upgrader
	dashboardSecret string
}

func NewMaster() *Master {
	return &Master{
		clients:         make(map[string]*websocket.Conn),
		dashboardConns:  make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
		dashboardSecret: generateRandomSecret(),
	}
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

	// Check if this is a dashboard connection
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
	}

	// This should be a client connection - require X-Client-ID
	if clientID == "" {
		log.Printf("Client attempted to connect without providing X-Client-ID header")
		conn.Close()
		return
	}

	m.clientsMu.Lock()
	m.clients[clientID] = conn
	m.clientsMu.Unlock()

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

	// Remove client on disconnect
	m.clientsMu.Lock()
	delete(m.clients, clientID)
	clientCount := len(m.clients)
	m.clientsMu.Unlock()

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

func (m *Master) getConnectedClients() []ClientInfo {
	m.clientsMu.RLock()
	defer m.clientsMu.RUnlock()

	clients := make([]ClientInfo, 0, len(m.clients))
	for clientID := range m.clients {
		clients = append(clients, ClientInfo{
			ID:       clientID,
			Name:     fmt.Sprintf("Client-%s", clientID[:8]),
			Status:   "connected",
			LastSeen: time.Now(),
		})
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
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #2196F3; color: white; padding: 20px; margin: -20px -20px 20px -20px; }
        .clients { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; }
        .client { border: 1px solid #ddd; padding: 15px; border-radius: 5px; }
        .client.connected { border-left: 5px solid #4CAF50; }
        .controls { margin: 20px 0; }
        .btn { background: #2196F3; color: white; border: none; padding: 10px 20px; margin: 5px; cursor: pointer; border-radius: 3px; }
        .btn:hover { background: #1976D2; }
        .btn.danger { background: #f44336; }
        .btn.danger:hover { background: #d32f2f; }
        .log { background: #f5f5f5; padding: 10px; height: 200px; overflow-y: auto; border-radius: 3px; font-family: monospace; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🎓 GradeKeeper Master Dashboard</h1>
        <p>Manage and control multiple GradeKeeper clients</p>
    </div>

    <div class="controls">
        <button class="btn" onclick="sendCommand('setup')">📁 Setup Environment (All)</button>
        <button class="btn" onclick="sendCommand('open-vscode')">💻 Open VS Code (All)</button>
        <button class="btn" onclick="sendCommand('open-chrome')">🌐 Open Chrome Incognito (All)</button>
        <button class="btn danger" onclick="sendCommand('clear')">🧹 Clear Environment (All)</button>
        <button class="btn" onclick="refreshClients()">🔄 Refresh</button>
    </div>

    <div id="clients-container">
        <h2>Connected Clients</h2>
        <div id="clients" class="clients"></div>
    </div>

    <div>
        <h2>Activity Log</h2>
        <div id="log" class="log"></div>
    </div>

    <script>
        const dashboardSecret = '%s';
        let ws;
        
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
                    container.innerHTML = clients.map(client => 
                        '<div class="client connected">' +
                        '<h3>' + client.name + '</h3>' +
                        '<p>ID: ' + client.id + '</p>' +
                        '<p>Status: ' + client.status + '</p>' +
                        '<button class="btn" onclick="sendCommandToClient(\'' + client.id + '\', \'setup\')">Setup</button>' +
                        '<button class="btn" onclick="sendCommandToClient(\'' + client.id + '\', \'open-vscode\')">VS Code</button>' +
                        '<button class="btn" onclick="sendCommandToClient(\'' + client.id + '\', \'open-chrome\')">Chrome</button>' +
                        '<button class="btn danger" onclick="sendCommandToClient(\'' + client.id + '\', \'clear\')">Clear</button>' +
                        '</div>'
                    ).join('');
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
        
        function log(message) {
            const logEl = document.getElementById('log');
            const timestamp = new Date().toLocaleTimeString();
            logEl.innerHTML += '[' + timestamp + '] ' + message + '\n';
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
	clients := m.getConnectedClients()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clients)
}

func main() {
	master := NewMaster()

	http.HandleFunc("/", master.handleDashboard)
	http.HandleFunc("/ws", master.handleWebSocket)
	http.HandleFunc("/api/command", master.handleAPICommand)
	http.HandleFunc("/api/clients", master.handleAPIClients)

	fmt.Println("🎓 GradeKeeper Master Server starting...")
	fmt.Println("📊 Dashboard: http://localhost:8080")
	fmt.Println("🔌 WebSocket: ws://localhost:8080/ws")
	fmt.Printf("🔐 Dashboard Secret: %s\n", master.dashboardSecret)

	log.Fatal(http.ListenAndServe(":8080", nil))
}