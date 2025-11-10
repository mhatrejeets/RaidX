package handlers

import (
	"encoding/json"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/mhatrejeets/RaidX/internal/middleware"
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
		// Create a dedicated write channel for this scorer connection
		// This ensures all writes to this connection are serialized
		writeCh := make(chan []byte, 50)
		stopCh := make(chan struct{})

		// Start write pump goroutine - handles all writes for this connection
		go func() {
			defer c.Close()
			for {
				select {
				case msg := <-writeCh:
					if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
						logrus.Error("Error:", "ScorerWritePump:", " Failed to write message: %v", err)
						return
					}
				case <-stopCh:
					return
				}
			}
		}()

		// JWT token must be present in query param
		token := c.Query("token")
		_, err := middleware.AuthWebSocket(token)
		if err != nil {
			logrus.Warn("WebSocket scorer: JWT invalid or missing")
			writeCh <- []byte(`{"error":"Unauthorized: Invalid JWT"}`)
			close(stopCh)
			return
		}
		// Expect first message from client to be a join with matchId
		_, joinMsg, err := c.ReadMessage()
		if err != nil {
			logrus.Error("Error:", "SetupWebSocket:", " Failed to read join message: %v", err)
			close(stopCh)
			return
		}
		var join struct {
			Type    string `json:"type"`
			MatchID string `json:"matchId"`
		}
		if err := json.Unmarshal(joinMsg, &join); err != nil || join.Type != "join" || join.MatchID == "" {
			// ask client to send proper join
			req := map[string]string{"type": "requestJoin"}
			if data, e := json.Marshal(req); e == nil {
				writeCh <- data
			}
			close(stopCh)
			return
		}

		matchID := join.MatchID
		room := GetRoom(matchID)
		room.AddScorer(c)
		defer func() {
			logrus.Info("Info:", "SetupWebSocket:", " Scorer connection closed")
			close(stopCh)
			room.RemoveScorer(c)
		}()

		// ...existing code...

		// send current match state from Redis (per-match key)
		var currentMatch models.EnhancedStatsMessage
		redisKey := "gameStats:" + matchID
		if err := redisImpl.GetRedisKey(redisKey, &currentMatch); err != nil {
			if err == redisImpl.RedisNull {
				// Ask client to send initial state
				req := map[string]string{"type": "requestInit"}
				if data, e := json.Marshal(req); e == nil {
					select {
					case writeCh <- data:
					case <-stopCh:
						return
					}
				}
			} else {
				logrus.Error("Error:", "SetupWebSocket:", " Failed to get gameStats for match %s: %v", matchID, err)
			}
		} else {
			if data, err := json.Marshal(currentMatch); err == nil {
				// send to connecting scorer only
				select {
				case writeCh <- data:
				case <-stopCh:
					return
				}
			}
		}

		// main read loop for this scorer
		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				logrus.Error("Error:", "SetupWebSocket:", " Error reading message from scorer: %v", err)
				break
			}

			// First probe if this is a raid payload (from scorer UI) or a full state update
			var probe struct {
				RaidType string `json:"raidType"`
				Type     string `json:"type"`
			}
			_ = json.Unmarshal(msg, &probe)
			if probe.Type == "initialState" {
				// client sent initial full state for this match - persist
				var received models.EnhancedStatsMessage
				if err := json.Unmarshal(msg, &received); err == nil {
					if err := redisImpl.SetRedisKey(redisKey, received); err == nil {
						// broadcast to room
						if data, e := json.Marshal(received); e == nil {
							room.BroadcastBytes(data)
						}
					}
				}
				continue
			}

			if probe.RaidType != "" {
				// It's a raid action. Unmarshal into a raid struct and process on backend
				var payload RaidPayload
				if err := json.Unmarshal(msg, &payload); err != nil {
					logrus.Error("Error:", "SetupWebSocket:", " Error unmarshalling raid payload: %v", err)
					continue
				}

				// Load current match state (per-match key)
				var currentMatch models.EnhancedStatsMessage
				if err := redisImpl.GetRedisKey(redisKey, &currentMatch); err != nil {
					if err == redisImpl.RedisNull {
						// Ask client to initialize server state
						errMsg := map[string]string{"error": "server: game state not initialized. Please send initial state"}
						if b, e := json.Marshal(errMsg); e == nil {
							select {
							case writeCh <- b:
							case <-stopCh:
								return
							}
						}
						continue
					}
					logrus.Error("Error:", "SetupWebSocket:", " Failed to get gameStats: %v", err)
					continue
				}

				// Validate payload against current match
				if err := validateRaidPayload(payload, &currentMatch); err != nil {
					errMsg := map[string]string{"error": err.Error()}
					if b, e := json.Marshal(errMsg); e == nil {
						select {
						case writeCh <- b:
						case <-stopCh:
							return
						}
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

				// persist updated state and broadcast to this room
				if err := redisImpl.SetRedisKey(redisKey, currentMatch); err != nil {
					logrus.Error("Error:", "SetupWebSocket:", " Failed to set gameStats: %v", err)
					continue
				}
				if data, err := json.Marshal(currentMatch); err == nil {
					room.BroadcastBytes(data)
					// also send updated state back to scorer who initiated the action
					select {
					case writeCh <- data:
					case <-stopCh:
						return
					}
				}
				continue
			}

			// Probe for custom non-raid message types (e.g., lobbyTouch)
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
				if err := redisImpl.GetRedisKey(redisKey, &currentMatch); err != nil {
					if err == redisImpl.RedisNull {
						// Ask client to initialize server state
						errMsg := map[string]string{"error": "server: game state not initialized. Please send initial state"}
						if b, e := json.Marshal(errMsg); e == nil {
							select {
							case writeCh <- b:
							case <-stopCh:
								return
							}
						}
						continue
					}
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

				// Update touched player's status to "out" so viewers/scorer UI stay in sync.
				if p, ok := currentMatch.Data.PlayerStats[lobbyPayload.Data.TouchedPlayerId]; ok {
					p.Status = "out"
					currentMatch.Data.PlayerStats[lobbyPayload.Data.TouchedPlayerId] = p
				}

				// Record raid detail
				currentMatch.Data.RaidDetails = models.RaidDetails{
					Type:         "lobbyTouch",
					Raider:       raiderName,
					PointsGained: 1,
				}

				// Increment raid number
				currentMatch.Data.RaidNumber++

				// Check for all-out and handle revivals/extra points if necessary
				checkAndHandleAllOut(&currentMatch)

				// persist updated state and broadcast
				if err := redisImpl.SetRedisKey(redisKey, currentMatch); err != nil {
					logrus.Error("Error:", "SetupWebSocket:", " Failed to set gameStats for lobbyTouch: %v", err)
					continue
				}
				if data, err := json.Marshal(currentMatch); err == nil {
					room.BroadcastBytes(data)
					select {
					case writeCh <- data:
					case <-stopCh:
						return
					}
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
			err = redisImpl.SetRedisKey(redisKey, receivedMessage)
			if err != nil {
				logrus.Error("Error:", "SetupWebSocket:", " Error storing data in Redis: %v", err)
			}

			// Broadcast updated stats to all viewers in this room
			if data, err := json.Marshal(receivedMessage); err == nil {
				room.BroadcastBytes(data)
			}
		}
	}))

	// Handle viewer WebSocket
	app.Get("/ws/viewer", websocket.New(func(c *websocket.Conn) {
		defer func() {
			logrus.Info("Info:", "SetupWebSocket:", " Viewer connection closed")
			c.Close()
		}()

		// --- JWT Auth for Viewer WebSocket ---
		token := c.Query("token")
		_, err := middleware.AuthWebSocket(token)
		if err != nil {
			resp := map[string]string{"type": "error", "message": "Unauthorized: Invalid or missing JWT token"}
			if data, e := json.Marshal(resp); e == nil {
				_ = c.WriteMessage(websocket.TextMessage, data)
			}
			return
		}

		// Expect a join message with matchId
		_, joinMsg, err := c.ReadMessage()
		if err != nil {
			logrus.Error("Error:", "SetupWebSocket:", " Failed to read join message from viewer: %v", err)
			return
		}
		var join struct {
			Type    string `json:"type"`
			MatchID string `json:"matchId"`
		}
		if err := json.Unmarshal(joinMsg, &join); err != nil || join.Type != "join" || join.MatchID == "" {
			req := map[string]string{"type": "requestJoin"}
			if data, e := json.Marshal(req); e == nil {
				_ = c.WriteMessage(websocket.TextMessage, data)
			}
			return
		}

		matchID := join.MatchID
		room := GetRoom(matchID)
		room.AddViewer(c)

		// send latest game stats from Redis for this match
		var latestStats models.EnhancedStatsMessage
		redisKey := "gameStats:" + matchID
		err = redisImpl.GetRedisKey(redisKey, &latestStats)
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

		room.RemoveViewer(c)
	}))
}
