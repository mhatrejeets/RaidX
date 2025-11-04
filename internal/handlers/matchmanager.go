package handlers

import (
	"sync"
	"time"

	"github.com/gofiber/websocket/v2"
)

// MatchRoom manages all connections and broadcasts for a single match_id
type MatchRoom struct {
	ID           string
	viewers      map[*websocket.Conn]bool
	scorers      map[*websocket.Conn]bool
	broadcastCh  chan []byte
	mu           sync.Mutex
	stopCh       chan struct{}
	lastActivity time.Time
}

// MatchManager keeps track of active match rooms
type MatchManager struct {
	rooms map[string]*MatchRoom
	mu    sync.RWMutex
}

var manager = &MatchManager{rooms: make(map[string]*MatchRoom)}

// GetRoom returns existing room or creates a new one
func GetRoom(matchID string) *MatchRoom {
	manager.mu.RLock()
	r, ok := manager.rooms[matchID]
	manager.mu.RUnlock()
	if ok {
		return r
	}

	manager.mu.Lock()
	defer manager.mu.Unlock()
	// Double-check
	if r, ok = manager.rooms[matchID]; ok {
		return r
	}
	r = &MatchRoom{
		ID:           matchID,
		viewers:      make(map[*websocket.Conn]bool),
		scorers:      make(map[*websocket.Conn]bool),
		broadcastCh:  make(chan []byte, 100),
		stopCh:       make(chan struct{}),
		lastActivity: time.Now(),
	}
	manager.rooms[matchID] = r
	go r.run()
	return r
}

// RemoveRoom deletes a room from manager
func RemoveRoom(matchID string) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if r, ok := manager.rooms[matchID]; ok {
		close(r.stopCh)
		delete(manager.rooms, matchID)
	}
}

// run forwards broadcasts to all viewers and cleans up inactive rooms
func (r *MatchRoom) run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case msg := <-r.broadcastCh:
			r.mu.Lock()
			for conn := range r.viewers {
				_ = conn.WriteMessage(websocket.TextMessage, msg)
			}
			r.mu.Unlock()
			r.lastActivity = time.Now()
		case <-ticker.C:
			// cleanup if no clients for a while
			r.mu.Lock()
			if len(r.viewers) == 0 && len(r.scorers) == 0 && time.Since(r.lastActivity) > time.Minute*5 {
				r.mu.Unlock()
				RemoveRoom(r.ID)
				return
			}
			r.mu.Unlock()
		case <-r.stopCh:
			return
		}
	}
}

func (r *MatchRoom) AddViewer(conn *websocket.Conn) {
	r.mu.Lock()
	r.viewers[conn] = true
	r.lastActivity = time.Now()
	r.mu.Unlock()
}

func (r *MatchRoom) RemoveViewer(conn *websocket.Conn) {
	r.mu.Lock()
	delete(r.viewers, conn)
	r.lastActivity = time.Now()
	r.mu.Unlock()
}

func (r *MatchRoom) AddScorer(conn *websocket.Conn) {
	r.mu.Lock()
	r.scorers[conn] = true
	r.lastActivity = time.Now()
	r.mu.Unlock()
}

func (r *MatchRoom) RemoveScorer(conn *websocket.Conn) {
	r.mu.Lock()
	delete(r.scorers, conn)
	r.lastActivity = time.Now()
	r.mu.Unlock()
}

func (r *MatchRoom) BroadcastBytes(b []byte) {
	select {
	case r.broadcastCh <- b:
	default:
		// drop if full
	}
}
