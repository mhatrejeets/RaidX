package handlers

import (
	"context"
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/mhatrejeets/RaidX/internal/middleware"
	"github.com/mhatrejeets/RaidX/internal/models"
	"github.com/mhatrejeets/RaidX/internal/redisImpl"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func SetupWebSocket(app *fiber.App) {
// Handle scorer WebSocket
app.Get("/ws/scorer", websocket.New(func(c *websocket.Conn) {
	token := c.Query("token")
	_, err := middleware.AuthWebSocket(token)
	if err != nil {
		resp := map[string]string{"error": "Unauthorized: Invalid JWT"}
		data, _ := json.Marshal(resp)
		c.WriteMessage(websocket.TextMessage, data)
		c.Close()
		return
	}
	_, joinMsg, err := c.ReadMessage()
	if err != nil {
		return
	}
	var join struct {
		Type    string `json:"type"`
		MatchID string `json:"matchId"`
	}
	if err := json.Unmarshal(joinMsg, &join); err != nil || join.Type != "join" || join.MatchID == "" {
		req := map[string]string{"type": "requestJoin"}
		data, _ := json.Marshal(req)
		c.WriteMessage(websocket.TextMessage, data)
		return
	}
	matchID := join.MatchID
	room := GetRoom(matchID)
	client := &Client{conn: c, send: make(chan []byte, 256), room: room}
	room.AddClient(client)
	client.StartWritePump()
	defer func() {
		room.RemoveClient(client)
		c.Close()
	}()
	// Send current match state
	var currentMatch models.EnhancedStatsMessage
	redisKey := "gameStats:" + matchID
	if err := redisImpl.GetRedisKey(redisKey, &currentMatch); err != nil {
		if err == redisImpl.RedisNull {
			req := map[string]string{"type": "requestInit"}
			data, _ := json.Marshal(req)
			client.send <- data
		}
	} else {
		if data, err := json.Marshal(currentMatch); err == nil {
			client.send <- data
		}
	}
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			break
		}
		// First probe if this is a raid payload (from scorer UI) or a full state update
		var probe struct {
			RaidType string `json:"raidType"`
			Type     string `json:"type"`
		}
		_ = json.Unmarshal(msg, &probe)
		if probe.Type == "initialState" {
			var received models.EnhancedStatsMessage
			if err := json.Unmarshal(msg, &received); err == nil {
				if err := redisImpl.SetRedisKey(redisKey, received); err == nil {
					if data, e := json.Marshal(received); e == nil {
						room.Broadcast(data)
					}
				}
			}
			continue
		}
		if probe.RaidType != "" {
			var payload RaidPayload
			if err := json.Unmarshal(msg, &payload); err != nil {
				continue
			}
			var currentMatch models.EnhancedStatsMessage
			if err := redisImpl.GetRedisKey(redisKey, &currentMatch); err != nil {
				continue
			}
			_ = json.Unmarshal(msg, &probe)
			if probe.Type == "initialState" {
				// client sent initial full state for this match - persist
				var received models.EnhancedStatsMessage
				if err := json.Unmarshal(msg, &received); err == nil {
					if err := redisImpl.SetRedisKey(redisKey, received); err == nil {
						if data, e := json.Marshal(received); e == nil {
							room.BroadcastBytes(data)
						}
					}
				}
				continue
			}

			if probe.RaidType != "" {
				var payload RaidPayload
				if err := json.Unmarshal(msg, &payload); err != nil {
					logrus.Error("Error:", "SetupWebSocket:", " Error unmarshalling raid payload: %v", err)
					continue
				}

				var currentMatch models.EnhancedStatsMessage
				if err := redisImpl.GetRedisKey(redisKey, &currentMatch); err != nil {
					if err == redisImpl.RedisNull {
						errMsg := map[string]string{"error": "server: game state not initialized. Please send initial state"}
						if b, e := json.Marshal(errMsg); e == nil {
							_ = c.WriteMessage(websocket.TextMessage, b)
						}
						continue
					}
					logrus.Error("Error:", "SetupWebSocket:", " Failed to get gameStats: %v", err)
					continue
				}

				if err := validateRaidPayload(payload, &currentMatch); err != nil {
					errMsg := map[string]string{"error": err.Error()}
					if b, e := json.Marshal(errMsg); e == nil {
						_ = c.WriteMessage(websocket.TextMessage, b)
					}
					continue
				}

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

				currentMatch.Data.Awards = computeAwardsFromPlayerStats(currentMatch.Data.PlayerStats)

				if err := redisImpl.SetRedisKey(redisKey, currentMatch); err != nil {
					logrus.Error("Error:", "SetupWebSocket:", " Failed to set gameStats: %v", err)
					continue
				}
				if data, err := json.Marshal(currentMatch); err == nil {
					room.BroadcastBytes(data)
					_ = c.WriteMessage(websocket.TextMessage, data)
				}
				continue
			}
			if data, err := json.Marshal(currentMatch); err == nil {
				room.Broadcast(data)
				client.send <- data
			}
			continue
		}
		var typeProbe struct {
			Type string `json:"type"`
		}
		_ = json.Unmarshal(msg, &typeProbe)
		if typeProbe.Type == "lobbyTouch" {
			var lobbyPayload struct {
				Type string `json:"type"`
				Data struct {
					TouchedPlayerId string `json:"touchedPlayerId"`
					IsRaider        bool   `json:"isRaider"`
					ScoringTeam     string `json:"scoringTeam"`
				} `json:"data"`
			}
			_ = json.Unmarshal(msg, &typeProbe)

			if typeProbe.Type == "lobbyTouch" {
				var lobbyPayload struct {
					Type string `json:"type"`
					Data struct {
						TouchedPlayerId string   `json:"touchedPlayerId"`
						IsRaider        bool     `json:"isRaider"`
						ScoringTeam     string   `json:"scoringTeam"`
						RaiderId        string   `json:"raiderId"`
						DefenderIds     []string `json:"defenderIds"`
						RaidNumber      int      `json:"raidNumber"`
					} `json:"data"`
				}
				if err := json.Unmarshal(msg, &lobbyPayload); err != nil {
					logrus.Error("Error:", "SetupWebSocket:", " Error unmarshalling lobby payload: %v", err)
					continue
				}

				var currentMatch models.EnhancedStatsMessage
				if err := redisImpl.GetRedisKey(redisKey, &currentMatch); err != nil {
					if err == redisImpl.RedisNull {
						errMsg := map[string]string{"error": "server: game state not initialized. Please send initial state"}
						if b, e := json.Marshal(errMsg); e == nil {
							_ = c.WriteMessage(websocket.TextMessage, b)
						}
						continue
					}
					logrus.Error("Error:", "SetupWebSocket:", " Failed to get gameStats for lobbyTouch: %v", err)
					continue
				}

				if lobbyPayload.Data.ScoringTeam == "A" {
					currentMatch.Data.TeamA.Score++
				} else {
					currentMatch.Data.TeamB.Score++
				}

				raiderName := ""
				if p, ok := currentMatch.Data.PlayerStats[lobbyPayload.Data.TouchedPlayerId]; ok {
					raiderName = p.Name
				}

				if p, ok := currentMatch.Data.PlayerStats[lobbyPayload.Data.TouchedPlayerId]; ok {
					p.Status = "out"
					currentMatch.Data.PlayerStats[lobbyPayload.Data.TouchedPlayerId] = p
				}

				if !lobbyPayload.Data.IsRaider && lobbyPayload.Data.RaiderId != "" {
					if r, ok := currentMatch.Data.PlayerStats[lobbyPayload.Data.RaiderId]; ok {
						r.RaidPoints++
						r.TotalPoints++
						currentMatch.Data.PlayerStats[lobbyPayload.Data.RaiderId] = r
					}
				}

				currentMatch.Data.PendingLobby.Events = append(currentMatch.Data.PendingLobby.Events, models.LobbyEvent{
					TouchedPlayerId: lobbyPayload.Data.TouchedPlayerId,
					IsRaider:        lobbyPayload.Data.IsRaider,
					ScoringTeam:     lobbyPayload.Data.ScoringTeam,
					RaidNumber:      lobbyPayload.Data.RaidNumber,
				})

				currentMatch.Data.RaidDetails = models.RaidDetails{
					Type:         "lobbyTouch",
					Raider:       raiderName,
					PointsGained: 1,
				}
				currentMatch.Data.Awards = computeAwardsFromPlayerStats(currentMatch.Data.PlayerStats)

				checkAndHandleAllOut(&currentMatch)

				if err := redisImpl.SetRedisKey(redisKey, currentMatch); err != nil {
					logrus.Error("Error:", "SetupWebSocket:", " Failed to set gameStats for lobbyTouch: %v", err)
					continue
				}
				if data, err := json.Marshal(currentMatch); err == nil {
					room.BroadcastBytes(data)
					_ = c.WriteMessage(websocket.TextMessage, data)
				}
				continue
			}

			// Otherwise treat as a full state update (legacy behavior)
			var receivedMessage models.EnhancedStatsMessage
			if err := json.Unmarshal(msg, &receivedMessage); err != nil {
				logrus.Error("Error:", "SetupWebSocket:", " Error unmarshalling scorer message: %v", err)
				continue
			}

			if err := redisImpl.SetRedisKey(redisKey, receivedMessage); err != nil {
				logrus.Error("Error:", "SetupWebSocket:", " Error storing data in Redis: %v", err)
			}

			if data, err := json.Marshal(receivedMessage); err == nil {
				room.BroadcastBytes(data)
			}
			currentMatch.Data.RaidDetails = models.RaidDetails{
				Type:         "lobbyTouch",
				Raider:       raiderName,
				PointsGained: 1,
			}
			currentMatch.Data.RaidNumber++
			checkAndHandleAllOut(&currentMatch)
			if err := redisImpl.SetRedisKey(redisKey, currentMatch); err != nil {
				continue
			}
			if data, err := json.Marshal(currentMatch); err == nil {
				room.Broadcast(data)
				client.send <- data
			}
			continue
		}
		var receivedMessage models.EnhancedStatsMessage
		err = json.Unmarshal(msg, &receivedMessage)
		if err != nil {
			continue
		}
		err = redisImpl.SetRedisKey(redisKey, receivedMessage)
		if err != nil {
			continue
		}
		if data, err := json.Marshal(receivedMessage); err == nil {
			room.Broadcast(data)
		}
	}
}))

	// Handle viewer WebSocket
	app.Get("/ws/viewer", websocket.New(func(c *websocket.Conn) {
		defer func() {
			logrus.Info("Info:", "SetupWebSocket:", " Viewer connection closed")
			c.Close()
		}()

		// --- Optional JWT Auth for Viewer WebSocket ---
		token := c.Query("token")
		if token != "" {
			if _, err := middleware.AuthWebSocket(token); err != nil {
				resp := map[string]string{"type": "error", "message": "Unauthorized: Invalid JWT token"}
				if data, e := json.Marshal(resp); e == nil {
					_ = c.WriteMessage(websocket.TextMessage, data)
				}
				return
			}
		}
		_, joinMsg, err := c.ReadMessage()
		if err != nil {
			return
		}
		var join struct {
			Type    string `json:"type"`
			MatchID string `json:"matchId"`
		}
		if err := json.Unmarshal(joinMsg, &join); err != nil || join.Type != "join" || join.MatchID == "" {
			req := map[string]string{"type": "requestJoin"}
			data, _ := json.Marshal(req)
			c.WriteMessage(websocket.TextMessage, data)
			return
		}
		matchID := join.MatchID
		room := GetRoom(matchID)
		client := &Client{conn: c, send: make(chan []byte, 256), room: room}
		room.AddClient(client)
		client.StartWritePump()
		defer func() {
			room.RemoveClient(client)
			c.Close()
		}()
		var latestStats models.EnhancedStatsMessage
		redisKey := "gameStats:" + matchID
		err = redisImpl.GetRedisKey(redisKey, &latestStats)
		if err == nil {
			data, _ := json.Marshal(latestStats)
			_ = c.WriteMessage(websocket.TextMessage, data)
		} else if err == redisImpl.RedisNull {
			// Match not found in Redis - check Mongo to distinguish ended vs not initialized
			matchesColl := db.MongoClient.Database("raidx").Collection("matches")
			var matchDoc bson.M
			mErr := matchesColl.FindOne(context.Background(), bson.M{"matchId": matchID}).Decode(&matchDoc)
			if mErr == nil {
				errMsg := map[string]string{"error": "Match ended", "matchId": matchID}
				if data, e := json.Marshal(errMsg); e == nil {
					_ = c.WriteMessage(websocket.TextMessage, data)
				}
			} else if mErr == mongo.ErrNoDocuments {
				errMsg := map[string]string{"error": "Match not initialized", "matchId": matchID}
				if data, e := json.Marshal(errMsg); e == nil {
					_ = c.WriteMessage(websocket.TextMessage, data)
				}
			} else {
				logrus.Error("Error:", "SetupWebSocket:", " Failed to check match in Mongo: %v", mErr)
				errMsg := map[string]string{"error": "Failed to retrieve match data"}
				if data, e := json.Marshal(errMsg); e == nil {
					_ = c.WriteMessage(websocket.TextMessage, data)
				}
			}
		} else {
			// Other Redis error
			logrus.Error("Error:", "SetupWebSocket:", " Failed to get match stats for viewer: %v", err)
			errMsg := map[string]string{"error": "Failed to retrieve match data"}
			if data, e := json.Marshal(errMsg); e == nil {
				_ = c.WriteMessage(websocket.TextMessage, data)
			}
		}
		for {
			if _, _, err := c.NextReader(); err != nil {
				break
			}
		}
	}))
}
