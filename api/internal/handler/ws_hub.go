package handler

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"

	"github.com/vasi1796/doit/internal/eventstore"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	maxMsgSize = 512
)

// Client wraps a WebSocket connection with a buffered send channel.
type Client struct {
	hub    *Hub
	userID uuid.UUID
	conn   *websocket.Conn
	send   chan []byte
	logger zerolog.Logger
}

// Hub maintains connected WebSocket clients per user and broadcasts events.
type Hub struct {
	mu      sync.RWMutex
	clients map[uuid.UUID]map[*Client]bool
	logger  zerolog.Logger
}

func NewHub(logger zerolog.Logger) *Hub {
	return &Hub{
		clients: make(map[uuid.UUID]map[*Client]bool),
		logger:  logger,
	}
}

func (h *Hub) Register(userID uuid.UUID, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*Client]bool)
	}
	h.clients[userID][client] = true
}

func (h *Hub) Unregister(userID uuid.UUID, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.clients[userID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.clients, userID)
		}
	}
	close(client.send)
}

// Broadcast sends events to all connected clients of a user except the sender.
func (h *Hub) Broadcast(userID uuid.UUID, events []eventstore.Event, sender *Client) {
	if len(events) == 0 {
		return
	}
	data, err := json.Marshal(events)
	if err != nil {
		h.logger.Error().Err(err).Msg("ws: failed to marshal events for broadcast")
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients[userID] {
		if client == sender {
			continue
		}
		select {
		case client.send <- data:
		default:
			// Client buffer full — drop message (client will catch up via sync)
			h.logger.Warn().Str("user_id", userID.String()).Msg("ws: dropping message, client buffer full")
		}
	}
}


func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister(c.userID, c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMsgSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Read loop — we only expect pong frames, discard any messages
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				c.logger.Debug().Err(err).Msg("ws: unexpected close")
			}
			break
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
