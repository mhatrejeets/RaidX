package handlers

import (
	"encoding/json"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/mhatrejeets/RaidX/internal/models"
	"github.com/mhatrejeets/RaidX/internal/redisImpl"
	"github.com/sirupsen/logrus"
)

var viewerClients = struct {
	clients map[*websocket.Conn]bool
	mu      sync.Mutex
}{
	clients: make(map[*websocket.Conn]bool),
}

var broadcastChan = make(chan []byte)

func StartBroadcastWorker() {
	go func() {
		for msg := range broadcastChan {
			viewerClients.mu.Lock()
			for conn := range viewerClients.clients {
				err := conn.WriteMessage(websocket.TextMessage, msg)
				if err != nil {
					logrus.Error("Error:", "StartBroadcastWorker:", " Error sending message to viewer: %v", err)
					conn.Close()
					delete(viewerClients.clients, conn)
				}
			}
			viewerClients.mu.Unlock()
		}
	}()
}

func BroadcastToViewers(message models.EnhancedStatsMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		logrus.Error("Error:", "BroadcastToViewers:", " Error marshalling data for viewers: %v", err)
		return
	}
	broadcastChan <- data
}

func SetupWebSocket(app *fiber.App) {
	// Start the broadcast worker
	StartBroadcastWorker()

	// Handle scorer WebSocket
	app.Get("/ws/scorer", websocket.New(func(c *websocket.Conn) {
		defer func() {
			logrus.Info("Info:", "SetupWebSocket:", " Scorer connection closed")
			c.Close()
		}()

		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				logrus.Error("Error:", "SetupWebSocket:", " Error reading message from scorer: %v", err)
				break
			}

			// Parse incoming stats
			var receivedMessage models.EnhancedStatsMessage
			err = json.Unmarshal(msg, &receivedMessage)
			if err != nil {
				logrus.Error("Error:", "SetupWebSocket:", " Error unmarshalling scorer message: %v", err)
				continue
			}

			// Store data in Redis
			err = redisImpl.SetRedisKey("gameStats", receivedMessage)
			if err != nil {
				logrus.Error("Error:", "SetupWebSocket:", " Error storing data in Redis: %v", err)
			}

			// Broadcast updated stats to all viewers
			BroadcastToViewers(receivedMessage)
		}
	}))

	// Handle viewer WebSocket
	app.Get("/ws/viewer", websocket.New(func(c *websocket.Conn) {
		viewerClients.mu.Lock()
		viewerClients.clients[c] = true
		viewerClients.mu.Unlock()

		defer func() {
			viewerClients.mu.Lock()
			delete(viewerClients.clients, c)
			viewerClients.mu.Unlock()

			c.Close()
		}()

		// Send latest game stats from Redis on new connection
		var latestStats models.EnhancedStatsMessage
		err := redisImpl.GetRedisKey("gameStats", &latestStats)
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
