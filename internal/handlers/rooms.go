package handlers

import (
	"sync"

	"github.com/gofiber/websocket/v2"
)

// Client represents a single WebSocket connection
// Only the writePump goroutine writes to conn
// All outgoing messages are sent to the send channel
// This prevents concurrent writes and disconnects

type Client struct {
	conn *websocket.Conn
	send chan []byte
	room *MatchRoom
}

// MatchRoom manages all clients for a single match
// Broadcasts are sent to all clients via their send channels

type MatchRoom struct {
	ID         string
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	mu         sync.Mutex
}

var rooms = struct {
	mu    sync.Mutex
	rooms map[string]*MatchRoom
}{
	rooms: make(map[string]*MatchRoom),
}

// GetRoom returns the room for a match, creating it if needed
func GetRoom(matchID string) *MatchRoom {
	rooms.mu.Lock()
	r, ok := rooms.rooms[matchID]
	if !ok {
		r = &MatchRoom{
			ID:         matchID,
			clients:    make(map[*Client]bool),
			register:   make(chan *Client),
			unregister: make(chan *Client),
			broadcast:  make(chan []byte, 256),
		}
		go r.run()
		rooms.rooms[matchID] = r
	}
	rooms.mu.Unlock()
	return r
}

// Room event loop
func (r *MatchRoom) run() {
	for {
		select {
		case client := <-r.register:
			r.mu.Lock()
			r.clients[client] = true
			r.mu.Unlock()
		case client := <-r.unregister:
			r.mu.Lock()
			if _, ok := r.clients[client]; ok {
				delete(r.clients, client)
				close(client.send)
			}
			r.mu.Unlock()
		case msg := <-r.broadcast:
			r.mu.Lock()
			for client := range r.clients {
				select {
				case client.send <- msg:
				default:
					// If send buffer is full, disconnect client
					close(client.send)
					delete(r.clients, client)
				}
			}
			r.mu.Unlock()
		}
	}
}

// Add a client to the room
func (r *MatchRoom) AddClient(client *Client) {
	r.register <- client
}

// Remove a client from the room
func (r *MatchRoom) RemoveClient(client *Client) {
	r.unregister <- client
}

// Broadcast a message to all clients in the room
func (r *MatchRoom) Broadcast(msg []byte) {
	r.broadcast <- msg
}

// Start the writePump for a client
func (c *Client) StartWritePump() {
	go func() {
		for msg := range c.send {
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				c.conn.Close()
				break
			}
		}
	}()
}
