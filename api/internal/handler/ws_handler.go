package handler

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

// Use default Gorilla upgrader which checks Origin against Host header.
var upgrader = websocket.Upgrader{}

// WSHandler upgrades HTTP connections to WebSocket for real-time event push.
type WSHandler struct {
	hub    *Hub
	logger zerolog.Logger
}

func NewWSHandler(hub *Hub, logger zerolog.Logger) *WSHandler {
	return &WSHandler{hub: hub, logger: logger}
}

// HandleWS upgrades the connection and registers the client with the hub.
// Auth is handled by the existing JWT middleware (cookie sent on upgrade).
func (h *WSHandler) HandleWS(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error().Err(err).Msg("ws: upgrade failed")
		return
	}

	client := &Client{
		hub:    h.hub,
		userID: userID,
		conn:   conn,
		send:   make(chan []byte, 256),
		logger: h.logger,
	}

	h.hub.Register(userID, client)
	go client.writePump()
	go client.readPump()
}
