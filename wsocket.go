package main

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

type PlayerStat struct {
	Name          string `json:"name"`
	ID            string `json:"id"`
	TotalPoints   int    `json:"totalPoints"`
	RaidPoints    int    `json:"raidPoints"`
	DefencePoints int    `json:"defencePoints"`
	Status        string `json:"status"`
}

type TeamStats struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}

type RaidDetails struct {
	Type         string   `json:"type"`
	Raider       string   `json:"raider"`
	Defenders    []string `json:"defenders,omitempty"`
	PointsGained int      `json:"pointsGained,omitempty"`
	BonusTaken   bool     `json:"bonusTaken,omitempty"`
	SuperTackle  bool     `json:"superTackle,omitempty"`
}

type EnhancedStatsMessage struct {
	Type string `json:"type"`
	Data struct {
		TeamA       TeamStats             `json:"teamA"`
		TeamB       TeamStats             `json:"teamB"`
		PlayerStats map[string]PlayerStat `json:"playerStats"`
		RaidDetails RaidDetails           `json:"raidDetails"`
	} `json:"data"`
}

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
					log.Println("Error sending message to viewer:", err)
					conn.Close()
					delete(viewerClients.clients, conn)
				}
			}
			viewerClients.mu.Unlock()
		}
	}()
}

func BroadcastToViewers(message EnhancedStatsMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling data for viewers:", err)
		return
	}
	broadcastChan <- data
}

func setupWebSocket(app *fiber.App) {
	// Start the broadcast worker
	StartBroadcastWorker()

	// Handle scorer WebSocket
	app.Get("/ws/scorer", websocket.New(func(c *websocket.Conn) {
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
			var receivedMessage EnhancedStatsMessage
			err = json.Unmarshal(msg, &receivedMessage)
			if err != nil {
				log.Println("Error unmarshalling scorer message:", err)
				continue
			}

			// Store data in Redis
			err = SetRedisKey("gameStats", receivedMessage)
			if err != nil {
				log.Println("Error storing data in Redis:", err)
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
		var latestStats EnhancedStatsMessage
		err := GetRedisKey("gameStats", &latestStats)
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
