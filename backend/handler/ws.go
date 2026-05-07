package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"loghub/config"
	"loghub/model"
	"loghub/service"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client represents a connected WebSocket client
type Client struct {
	conn      *websocket.Conn
	appID     string
	clientID  string
	isViewer  bool
	viewAppID string
	send      chan []byte
	hub       *WSHub
}

// WSHub manages all WebSocket connections
type WSHub struct {
	mu         sync.RWMutex
	producers  map[string]*Client // appID -> producer client
	viewers    map[*Client]bool   // viewer clients
	logStore   *service.LogStore
	logger     *zap.Logger
	broadcast  chan *model.LogEntry
	register   chan *Client
	unregister chan *Client
}

// NewWSHub creates a new WebSocket hub
func NewWSHub(store *service.LogStore, logger *zap.Logger) *WSHub {
	return &WSHub{
		producers:  make(map[string]*Client),
		viewers:    make(map[*Client]bool),
		logStore:   store,
		logger:     logger,
		broadcast:  make(chan *model.LogEntry, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's main event loop
func (h *WSHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if client.isViewer {
				h.viewers[client] = true
				h.logger.Info("viewer connected",
					zap.String("view_app_id", client.viewAppID),
				)
			} else {
				h.producers[client.appID] = client
				h.logger.Info("producer connected",
					zap.String("app_id", client.appID),
				)
			}
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if client.isViewer {
				delete(h.viewers, client)
				h.logger.Info("viewer disconnected")
			} else {
				if existing, ok := h.producers[client.appID]; ok && existing == client {
					delete(h.producers, client.appID)
					h.logger.Info("producer disconnected",
						zap.String("app_id", client.appID),
					)
				}
			}
			close(client.send)
			h.mu.Unlock()

		case entry := <-h.broadcast:
			h.mu.RLock()
			msg, _ := json.Marshal(&model.WSResponse{
				Type: "log",
				Code: 200,
				Data: entry,
			})
			for viewer := range h.viewers {
				if viewer.viewAppID == "" || viewer.viewAppID == entry.AppID || viewer.viewAppID == "all" {
					select {
					case viewer.send <- msg:
					default:
						// channel full, skip
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// GetOnlineAppCount returns the number of online producer apps
func (h *WSHub) GetOnlineAppCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.producers)
}

// GetOnlineApps returns the list of online app IDs
func (h *WSHub) GetOnlineApps() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make([]string, 0, len(h.producers))
	for appID := range h.producers {
		result = append(result, appID)
	}
	return result
}

// HandleProducer handles WebSocket connections from log producer apps
func (h *WSHub) HandleProducer(w http.ResponseWriter, r *http.Request) {
	appID := r.URL.Query().Get("app_id")
	if appID == "" {
		writeWSError(w, http.StatusBadRequest, "app_id is required")
		return
	}

	cfg := config.Get()
	if !cfg.IsAppAllowed(appID) {
		writeWSError(w, http.StatusForbidden, fmt.Sprintf("app_id %s is not allowed", appID))
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("failed to upgrade websocket connection",
			zap.String("app_id", appID),
			zap.Error(err),
		)
		return
	}

	client := &Client{
		conn:     conn,
		appID:    appID,
		clientID: fmt.Sprintf("%s-%d", appID, time.Now().UnixNano()),
		isViewer: false,
		send:     make(chan []byte, 256),
		hub:      h,
	}

	h.register <- client

	// Send connection acknowledgment
	ack, _ := json.Marshal(&model.WSResponse{
		Type:    "connected",
		Code:    200,
		Message: "connected to LogHub",
		Data: map[string]interface{}{
			"app_id":    appID,
			"client_id": client.clientID,
		},
	})
	conn.WriteMessage(websocket.TextMessage, ack)

	go client.writePump()
	go client.readPumpProducer()
}

// HandleViewer handles WebSocket connections from log viewer clients
func (h *WSHub) HandleViewer(w http.ResponseWriter, r *http.Request) {
	viewAppID := r.URL.Query().Get("app_id")
	if viewAppID == "" {
		viewAppID = "all"
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("failed to upgrade viewer websocket",
			zap.Error(err),
		)
		return
	}

	client := &Client{
		conn:      conn,
		isViewer:  true,
		viewAppID: viewAppID,
		send:      make(chan []byte, 256),
		hub:       h,
	}

	h.register <- client

	ack, _ := json.Marshal(&model.WSResponse{
		Type:    "connected",
		Code:    200,
		Message: "viewer connected",
		Data: map[string]interface{}{
			"watching": viewAppID,
		},
	})
	conn.WriteMessage(websocket.TextMessage, ack)

	go client.writePump()
	go client.readPumpViewer()
}

func (c *Client) readPumpProducer() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(64 * 1024)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				c.hub.logger.Warn("producer connection closed unexpectedly",
					zap.String("app_id", c.appID),
					zap.Error(err),
				)
			}
			return
		}

		var wsMsg model.WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			c.hub.logger.Warn("invalid message format from producer",
				zap.String("app_id", c.appID),
				zap.Error(err),
			)
			sendError(c, "invalid message format")
			continue
		}

		switch wsMsg.Type {
		case "log":
			c.handleLogMessage(wsMsg.Payload)
		case "batch_log":
			c.handleBatchLogMessage(wsMsg.Payload)
		case "ping":
			resp, _ := json.Marshal(&model.WSResponse{
				Type: "pong",
				Code: 200,
			})
			c.send <- resp
		default:
			sendError(c, fmt.Sprintf("unknown message type: %s", wsMsg.Type))
		}
	}
}

func (c *Client) handleLogMessage(payload interface{}) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		sendError(c, "invalid log payload")
		return
	}

	var entry model.LogEntry
	if err := json.Unmarshal(payloadBytes, &entry); err != nil {
		sendError(c, "invalid log entry format")
		return
	}

	entry.AppID = c.appID
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().Format(time.RFC3339)
	}

	if err := c.hub.logStore.WriteLog(&entry); err != nil {
		c.hub.logger.Error("failed to write log",
			zap.String("app_id", c.appID),
			zap.Error(err),
		)
		sendError(c, "failed to write log")
		return
	}

	c.hub.broadcast <- &entry
}

func (c *Client) handleBatchLogMessage(payload interface{}) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		sendError(c, "invalid batch log payload")
		return
	}

	var entries []model.LogEntry
	if err := json.Unmarshal(payloadBytes, &entries); err != nil {
		sendError(c, "invalid batch log format")
		return
	}

	for i := range entries {
		entries[i].AppID = c.appID
		if entries[i].Timestamp == "" {
			entries[i].Timestamp = time.Now().Format(time.RFC3339)
		}

		if err := c.hub.logStore.WriteLog(&entries[i]); err != nil {
			c.hub.logger.Error("failed to write batch log entry",
				zap.String("app_id", c.appID),
				zap.Error(err),
			)
			continue
		}
		c.hub.broadcast <- &entries[i]
	}

	resp, _ := json.Marshal(&model.WSResponse{
		Type:    "batch_ack",
		Code:    200,
		Message: fmt.Sprintf("received %d log entries", len(entries)),
	})
	c.send <- resp
}

func (c *Client) readPumpViewer() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(4096)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				c.hub.logger.Warn("viewer connection closed unexpectedly",
					zap.Error(err),
				)
			}
			return
		}

		var wsMsg model.WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			continue
		}

		switch wsMsg.Type {
		case "subscribe":
			if payload, ok := wsMsg.Payload.(map[string]interface{}); ok {
				if appID, ok := payload["app_id"].(string); ok {
					c.viewAppID = appID
					resp, _ := json.Marshal(&model.WSResponse{
						Type:    "subscribed",
						Code:    200,
						Message: fmt.Sprintf("subscribed to %s", appID),
					})
					c.send <- resp
				}
			}
		case "ping":
			resp, _ := json.Marshal(&model.WSResponse{
				Type: "pong",
				Code: 200,
			})
			c.send <- resp
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func sendError(c *Client, msg string) {
	resp, _ := json.Marshal(&model.WSResponse{
		Type:    "error",
		Code:    400,
		Message: msg,
	})
	select {
	case c.send <- resp:
	default:
	}
}

func writeWSError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(model.APIResponse{
		Code:    code,
		Message: msg,
	})
}
