package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

// Hub maintains the set of active clients and broadcasts messages to clients
type Hub struct {
	clients         map[*Client]bool
	register        chan *Client
	unregister      chan *Client
	broadcast       chan *Sighting
	broadcastUpdate chan *SightingUpdate
	alertRadius     float64 // in miles
	mu              sync.RWMutex
}

// NewHub creates a new Hub instance
func NewHub(alertRadiusMiles float64) *Hub {
	return &Hub{
		clients:         make(map[*Client]bool),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		broadcast:       make(chan *Sighting),
		broadcastUpdate: make(chan *SightingUpdate),
		alertRadius:     alertRadiusMiles,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Client registered. Total clients: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("Client unregistered. Total clients: %d", len(h.clients))

		case sighting := <-h.broadcast:
			h.broadcastToNearbyClients(sighting)

		case update := <-h.broadcastUpdate:
			h.broadcastUpdateToAllClients(update)
		}
	}
}

// broadcastToNearbyClients sends a sighting to all clients within radius
func (h *Hub) broadcastToNearbyClients(sighting *Sighting) {
	// Exclude image data from broadcast to reduce bandwidth
	broadcastSighting := *sighting
	broadcastSighting.ImageData = ""

	message := WebSocketMessage{
		Type:    "sighting",
		Payload: broadcastSighting,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling sighting: %v", err)
		return
	}

	var toRemove []*Client

	h.mu.RLock()
	for client := range h.clients {
		// Check if client is within radius
		if client.Lat == 0 && client.Lng == 0 {
			continue
		}

		if IsWithinRadius(client.Lat, client.Lng, sighting.Lat, sighting.Lng, h.alertRadius) {
			select {
			case client.Send <- data:
			default:
				toRemove = append(toRemove, client)
			}
		}
	}
	h.mu.RUnlock()

	// Remove slow clients under write lock
	if len(toRemove) > 0 {
		h.mu.Lock()
		for _, client := range toRemove {
			if _, ok := h.clients[client]; ok {
				close(client.Send)
				delete(h.clients, client)
			}
		}
		h.mu.Unlock()
	}
}

// BroadcastSighting queues a sighting for broadcast
func (h *Hub) BroadcastSighting(sighting *Sighting) {
	h.broadcast <- sighting
}

// broadcastUpdateToAllClients sends an update to all connected clients
func (h *Hub) broadcastUpdateToAllClients(update *SightingUpdate) {
	message := WebSocketUpdateMessage{
		Type:    "sighting_update",
		Payload: *update,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling sighting update: %v", err)
		return
	}

	var toRemove []*Client

	h.mu.RLock()
	for client := range h.clients {
		select {
		case client.Send <- data:
		default:
			toRemove = append(toRemove, client)
		}
	}
	h.mu.RUnlock()

	if len(toRemove) > 0 {
		h.mu.Lock()
		for _, client := range toRemove {
			if _, ok := h.clients[client]; ok {
				close(client.Send)
				delete(h.clients, client)
			}
		}
		h.mu.Unlock()
	}
}

// BroadcastSightingUpdate queues a sighting update for broadcast
func (h *Hub) BroadcastSightingUpdate(update *SightingUpdate) {
	h.broadcastUpdate <- update
}

// BroadcastSupportRequest sends a high-priority support request to clients within specified radius
func (h *Hub) BroadcastSupportRequest(sighting *Sighting, radiusMiles float64) {
	message := SupportRequestMessage{
		Type:     "support_request",
		Sighting: *sighting,
		Radius:   radiusMiles,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling support request: %v", err)
		return
	}

	var toRemove []*Client

	h.mu.RLock()
	for client := range h.clients {
		if client.Lat == 0 && client.Lng == 0 {
			continue
		}

		if IsWithinRadius(client.Lat, client.Lng, sighting.Lat, sighting.Lng, radiusMiles) {
			select {
			case client.Send <- data:
			default:
				toRemove = append(toRemove, client)
			}
		}
	}
	h.mu.RUnlock()

	if len(toRemove) > 0 {
		h.mu.Lock()
		for _, client := range toRemove {
			if _, ok := h.clients[client]; ok {
				close(client.Send)
				delete(h.clients, client)
			}
		}
		h.mu.Unlock()
	}
}

// UpdateClientLocation updates a client's location
func (h *Hub) UpdateClientLocation(client *Client, lat, lng float64) {
	h.mu.Lock()
	client.Lat = lat
	client.Lng = lng
	h.mu.Unlock()
}

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Parse location update
		var update LocationUpdate
		if err := json.Unmarshal(message, &update); err != nil {
			log.Printf("Error parsing message: %v", err)
			continue
		}

		if update.Type == "location" {
			c.Hub.UpdateClientLocation(c, update.Lat, update.Lng)
		}
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
