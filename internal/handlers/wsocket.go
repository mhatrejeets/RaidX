package handlers

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/mhatrejeets/RaidX/internal/models"
	redisC "github.com/mhatrejeets/RaidX/internal/redis"
)

var viewerClients = struct {
	clients map[*websocket.Conn]bool
	mu      sync.Mutex
}{
	clients: make(map[*websocket.Conn]bool),
}

// Multi-match support
var matchBroadcastChans = struct {
	mu    sync.Mutex
	chans map[string]chan []byte // matchID -> channel
}{
	chans: make(map[string]chan []byte),
}

func getMatchChan(matchID string) chan []byte {
	matchBroadcastChans.mu.Lock()
	defer matchBroadcastChans.mu.Unlock()
	if ch, ok := matchBroadcastChans.chans[matchID]; ok {
		return ch
	}
	ch := make(chan []byte)
	matchBroadcastChans.chans[matchID] = ch
	return ch
}

func StartMatchBroadcastWorker(matchID string) {
	ch := getMatchChan(matchID)
	go func() {
		for msg := range ch {
			viewerClients.mu.Lock()
			for conn := range viewerClients.clients {
				// Optionally filter by matchID if you track connections per match
				err := conn.WriteMessage(websocket.TextMessage, msg)
				if err != nil {
					log.Println("Error sending message to viewer:", err)
					conn.Close()
					delete(viewerClients.clients, conn)
				}
			}
			viewerClients.mu.Unlock()
		}
	}()
}

func BroadcastToMatchViewers(matchID string, message models.EnhancedStatsMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling data for viewers:", err)
		return
	}
	ch := getMatchChan(matchID)
	ch <- data
}

func SetupWebSocket(app *fiber.App) {
	// Handle scorer WebSocket
	app.Get("/ws/scorer", websocket.New(func(c *websocket.Conn) {
		matchID := c.Query("matchID")
		StartMatchBroadcastWorker(matchID)
		defer func() {
			log.Println("Scorer connection closed")
			c.Close()
		}()

		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				log.Println("Error reading message from scorer:", err)
				break
			}

			// Parse incoming stats
			var receivedMessage models.EnhancedStatsMessage
			err = json.Unmarshal(msg, &receivedMessage)
			if err != nil {
				log.Println("Error unmarshalling scorer message:", err)
				continue
			}

			// Store data in Redis with matchID as key
			redisKey := "gameStats:" + matchID
			err = redisC.SetRedisKey(redisKey, receivedMessage)
			if err != nil {
				log.Println("Error storing data in Redis:", err)
			}

			// Broadcast updated stats to all viewers
			BroadcastToMatchViewers(matchID, receivedMessage)
		}
	}))

	// Handle viewer WebSocket
	app.Get("/ws/viewer", websocket.New(func(c *websocket.Conn) {
		matchID := c.Query("matchID")
		StartMatchBroadcastWorker(matchID)
		viewerClients.mu.Lock()
		viewerClients.clients[c] = true
		viewerClients.mu.Unlock()

		log.Printf("Viewer connected. Total viewers: %d", len(viewerClients.clients))

		defer func() {
			viewerClients.mu.Lock()
			delete(viewerClients.clients, c)
			viewerClients.mu.Unlock()
			log.Printf("Viewer disconnected. Total viewers: %d", len(viewerClients.clients))
			c.Close()
		}()

		// Send latest game stats from Redis for this match
		var latestStats models.EnhancedStatsMessage
		err := redisC.GetRedisKey("gameStats:"+matchID, &latestStats)
		if err == nil {
			data, _ := json.Marshal(latestStats)
			_ = c.WriteMessage(websocket.TextMessage, data)
		}

		// Keep connection open
		for {
			if _, _, err := c.NextReader(); err != nil {
				break
			}
		}
	}))
}
