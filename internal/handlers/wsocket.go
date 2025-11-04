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

			// First probe if this is a raid payload (from scorer UI) or a full state update
			var probe struct {
				RaidType string `json:"raidType"`
			}
			err = json.Unmarshal(msg, &probe)
			if err == nil && probe.RaidType != "" {
				// It's a raid action. Unmarshal into a raid struct and process on backend
				var payload RaidPayload
				if err := json.Unmarshal(msg, &payload); err != nil {
					logrus.Error("Error:", "SetupWebSocket:", " Error unmarshalling raid payload: %v", err)
					continue
				}

				// Load current match state
				var currentMatch models.EnhancedStatsMessage
				if err := redisImpl.GetRedisKey("gameStats", &currentMatch); err != nil {
					logrus.Error("Error:", "SetupWebSocket:", " Failed to get gameStats: %v", err)
					continue
				}

				// Validate payload against current match
				if err := validateRaidPayload(payload, &currentMatch); err != nil {
					errMsg := map[string]string{"error": err.Error()}
					if b, e := json.Marshal(errMsg); e == nil {
						_ = c.WriteMessage(websocket.TextMessage, b)
					}
					continue
				}

				// Process raid using backend logic in matches.go
				switch payload.RaidType {
				case "successful":
					processSuccessfulRaid(&currentMatch, payload)
				case "defense":
					processDefenseSuccess(&currentMatch, payload)
				case "empty":
					processEmptyRaid(&currentMatch, payload)
				default:
					logrus.Warn("Warning:", "SetupWebSocket:", " Unknown raid type from scorer: %v", payload.RaidType)
				}

				// persist updated state and broadcast
				if err := redisImpl.SetRedisKey("gameStats", currentMatch); err != nil {
					logrus.Error("Error:", "SetupWebSocket:", " Failed to set gameStats: %v", err)
					continue
				}
				BroadcastToViewers(currentMatch)
				// also send updated state back to scorer who initiated the action
				if data, err := json.Marshal(currentMatch); err == nil {
					_ = c.WriteMessage(websocket.TextMessage, data)
				}
				continue
			}

			// Probe for custom non-raid message types (e.g., lobby touch)
			var typeProbe struct {
				Type string `json:"type"`
			}
			_ = json.Unmarshal(msg, &typeProbe)

			if typeProbe.Type == "lobbyTouch" {
				// handle lobby touch events (scorer UI -> backend)
				var lobbyPayload struct {
					Type string `json:"type"`
					Data struct {
						TouchedPlayerId string `json:"touchedPlayerId"`
						IsRaider        bool   `json:"isRaider"`
						ScoringTeam     string `json:"scoringTeam"`
					} `json:"data"`
				}
				if err := json.Unmarshal(msg, &lobbyPayload); err != nil {
					logrus.Error("Error:", "SetupWebSocket:", " Error unmarshalling lobby payload: %v", err)
					continue
				}

				// Load current match state
				var currentMatch models.EnhancedStatsMessage
				if err := redisImpl.GetRedisKey("gameStats", &currentMatch); err != nil {
					logrus.Error("Error:", "SetupWebSocket:", " Failed to get gameStats for lobbyTouch: %v", err)
					continue
				}

				// Award a single point to the scoring team (frontend determines which team)
				if lobbyPayload.Data.ScoringTeam == "A" {
					currentMatch.Data.TeamA.Score++
				} else {
					currentMatch.Data.TeamB.Score++
				}

				// Try to resolve player name from playerStats map for clarity
				raiderName := ""
				if p, ok := currentMatch.Data.PlayerStats[lobbyPayload.Data.TouchedPlayerId]; ok {
					raiderName = p.Name
				}

				// Record raid detail
				currentMatch.Data.RaidDetails = models.RaidDetails{
					Type:         "lobbyTouch",
					Raider:       raiderName,
					PointsGained: 1,
				}

				// Increment raid number
				currentMatch.Data.RaidNumber++

				// persist updated state and broadcast
				if err := redisImpl.SetRedisKey("gameStats", currentMatch); err != nil {
					logrus.Error("Error:", "SetupWebSocket:", " Failed to set gameStats for lobbyTouch: %v", err)
					continue
				}
				BroadcastToViewers(currentMatch)

				// also send updated state back to scorer who initiated the action
				if data, err := json.Marshal(currentMatch); err == nil {
					_ = c.WriteMessage(websocket.TextMessage, data)
				}
				continue
			}

			// Otherwise treat as a full state update (legacy behavior)
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
