package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"chat-backend/models"
	"chat-backend/services"

	"github.com/gorilla/websocket"
)

type Hub struct {
	// roomID -> clients
	rooms map[int]map[*Client]bool

	register   chan *Client
	unregister chan *Client
	broadcast  chan outbound

	mu sync.RWMutex
}

type outbound struct {
	roomID int
	data   []byte
}

type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	roomID   int
	userID   int
	username string
	msgSvc   *services.MessageService
}

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[int]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan outbound, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.addClient(c)
		case c := <-h.unregister:
			h.removeClient(c)
		case out := <-h.broadcast:
			for client := range h.rooms[out.roomID] {
				select {
				case client.send <- out.data:
				default:
					close(client.send)
					delete(h.rooms[out.roomID], client)
				}
			}
		}
	}
}

func (h *Hub) addClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.rooms[client.roomID] == nil {
		h.rooms[client.roomID] = make(map[*Client]bool)
	}
	h.rooms[client.roomID][client] = true

	log.Printf("Client %s (ID: %d) joined room %d. Total clients in room: %d",
		client.username, client.userID, client.roomID, len(h.rooms[client.roomID]))
}

func (h *Hub) removeClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, exists := h.rooms[client.roomID]; exists {
		if _, ok := clients[client]; ok {
			delete(clients, client)
			close(client.send)
			log.Printf("Client %s (ID: %d) left room %d. Remaining clients in room: %d",
				client.username, client.userID, client.roomID, len(clients))

			// Clean up empty rooms
			if len(clients) == 0 {
				delete(h.rooms, client.roomID)
				log.Printf("Room %d is now empty, removing from hub", client.roomID)
			}
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// CORS: allow all for demo
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request, roomID, userID int, username string, msgSvc *services.MessageService) {
	log.Printf("Attempting WebSocket upgrade for user %s (ID: %d) in room %d", username, userID, roomID)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed for user %s: %v", username, err)
		return
	}

	log.Printf("WebSocket upgrade successful for user %s (ID: %d) in room %d", username, userID, roomID)

	client := &Client{
		hub:      h,
		conn:     conn,
		send:     make(chan []byte, 256),
		roomID:   roomID,
		userID:   userID,
		username: username,
		msgSvc:   msgSvc,
	}
	h.register <- client

	go client.writePump()
	go client.readPump()
}

// Client reads from socket, rebroadcasts messages as-is (content is validated in service).
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(1 << 20)
	c.conn.SetReadDeadline(time.Now().Add(300 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(300 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("Client %s read error: %v", c.username, err)
			break
		}

		// Parse the message to check type
		var msgData map[string]interface{}
		if err := json.Unmarshal(message, &msgData); err != nil {
			log.Printf("Client %s unmarshal error: %v", c.username, err)
			continue
		}

		// Handle ping/pong messages
		if msgType, ok := msgData["type"].(string); ok {
			switch msgType {
			case "ping":
				// Respond to ping with pong
				pongMsg := map[string]string{"type": "pong"}
				if pongBytes, err := json.Marshal(pongMsg); err == nil {
					c.conn.WriteMessage(websocket.TextMessage, pongBytes)
				}
				continue
			case "pong":
				// Pong received, connection is healthy
				continue
			}
		}

		// Handle regular chat messages
		var body struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(message, &body); err != nil {
			log.Printf("Client %s content unmarshal error: %v", c.username, err)
			continue
		}

		if body.Content == "" {
			log.Printf("Client %s sent empty message", c.username)
			continue
		}

		if _, err := c.msgSvc.Send(c.roomID, c.userID, body.Content); err != nil {
			log.Printf("Client %s message send error: %v", c.username, err)
			continue
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(240 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
			if !ok {
				log.Printf("Client %s send channel closed", c.username)
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Printf("Client %s next writer error: %v", c.username, err)
				return
			}
			if _, err := w.Write(msg); err != nil {
				log.Printf("Client %s write error: %v", c.username, err)
				return
			}
			if err := w.Close(); err != nil {
				log.Printf("Client %s writer close error: %v", c.username, err)
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte(strconv.Itoa(c.userID))); err != nil {
				log.Printf("Client %s ping error: %v", c.username, err)
				return
			}
		}
	}
}

// BroadcastMessage allows services to fan-out persisted messages to all clients.
func (h *Hub) BroadcastMessage(msg models.Message, username string) {
	out := map[string]any{
		"type":      "message",
		"room_id":   msg.RoomID,
		"sender_id": msg.SenderID,
		"username":  username,
		"content":   msg.Content,
		"ts":        msg.CreatedAt.UnixMilli(),
	}
	b, _ := json.Marshal(out)
	h.broadcast <- outbound{roomID: msg.RoomID, data: b}
}

// GetUserCount returns the number of users in a specific room
func (h *Hub) GetUserCount(roomID int) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, exists := h.rooms[roomID]; exists {
		return len(clients)
	}
	return 0
}
