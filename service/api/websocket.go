package api

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"github.com/ozberk-sevinc/wasa-project/service/api/reqcontext"
	"github.com/sirupsen/logrus"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development - adjust for production
		return true
	},
}

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type    string      `json:"type"`    // "new_message", "message_deleted", "reaction_added", etc.
	Payload interface{} `json:"payload"` // The actual data
}

// WebSocketConnection wraps a WebSocket connection with a mutex for thread-safe writes
type WebSocketConnection struct {
	conn     *websocket.Conn
	writeMux sync.Mutex
}

// WriteJSON writes a JSON message to the WebSocket connection (thread-safe)
func (wsc *WebSocketConnection) WriteJSON(v interface{}) error {
	wsc.writeMux.Lock()
	defer wsc.writeMux.Unlock()
	return wsc.conn.WriteJSON(v)
}

// WriteMessage writes a message to the WebSocket connection (thread-safe)
func (wsc *WebSocketConnection) WriteMessage(messageType int, data []byte) error {
	wsc.writeMux.Lock()
	defer wsc.writeMux.Unlock()
	return wsc.conn.WriteMessage(messageType, data)
}

// Close closes the WebSocket connection
func (wsc *WebSocketConnection) Close() error {
	return wsc.conn.Close()
}

// WebSocketHub manages all active WebSocket connections
type WebSocketHub struct {
	// Map of userID -> WebSocket connection wrapper
	connections map[string]*WebSocketConnection
	mu          sync.RWMutex
	logger      *logrus.Logger
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub(logger *logrus.Logger) *WebSocketHub {
	return &WebSocketHub{
		connections: make(map[string]*WebSocketConnection),
		logger:      logger,
	}
}

// Register adds a connection for a user
func (h *WebSocketHub) Register(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Close existing connection if any
	if existingConn, exists := h.connections[userID]; exists {
		existingConn.Close()
	}

	// Wrap connection with mutex for thread-safe writes
	h.connections[userID] = &WebSocketConnection{
		conn: conn,
	}
	h.logger.WithField("user_id", userID).Info("WebSocket connection registered")
}

// Unregister removes a connection for a user
func (h *WebSocketHub) Unregister(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conn, exists := h.connections[userID]; exists {
		conn.Close()
		delete(h.connections, userID)
		h.logger.WithField("user_id", userID).Info("WebSocket connection unregistered")
	}
}

// SendToUser sends a message to a specific user
func (h *WebSocketHub) SendToUser(userID string, message WebSocketMessage) error {
	h.mu.RLock()
	conn, exists := h.connections[userID]
	h.mu.RUnlock()

	if !exists {
		// User not connected, that's okay
		return nil
	}

	data, err := json.Marshal(message)
	if err != nil {
		h.logger.WithError(err).Error("error marshaling WebSocket message")
		return err
	}

	// Use thread-safe WriteMessage method
	err = conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		h.logger.WithError(err).WithField("user_id", userID).Error("error sending WebSocket message")
		// Connection is broken, unregister it
		h.Unregister(userID)
		return err
	}

	return nil
}

// BroadcastToUsers sends a message to multiple users
func (h *WebSocketHub) BroadcastToUsers(userIDs []string, message WebSocketMessage) {
	for _, userID := range userIDs {
		go h.SendToUser(userID, message)
	}
}

// handleWebSocket handles WebSocket upgrade and connection
func (rt *_router) handleWebSocket(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	// Try to get user from context (if authWrap was used)
	user := GetUserFromContext(r.Context())

	// If not in context, try to get token from query parameter
	if user == nil {
		token := r.URL.Query().Get("token")
		if token != "" {
			// Validate token and get user
			var err error
			user, err = rt.db.GetUserByID(token)
			if err != nil || user == nil {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}
		} else {
			sendUnauthorized(w, "User not found in context")
			return
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		ctx.Logger.WithError(err).Error("error upgrading to WebSocket")
		return
	}

	// Register the connection
	rt.wsHub.Register(user.ID, conn)

	// Handle incoming messages (ping/pong for keep-alive)
	go func() {
		defer rt.wsHub.Unregister(user.ID)

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				// Connection closed or error
				break
			}
			// We don't process incoming messages for now, just keep connection alive
		}
	}()
}
