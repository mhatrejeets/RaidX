package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/mhatrejeets/RaidX/internal/middleware"
	"github.com/mhatrejeets/RaidX/internal/models"
	"github.com/mhatrejeets/RaidX/internal/redisImpl"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var viewerClients = struct {
	clients map[*websocket.Conn]bool
	mu      sync.Mutex
}{
	clients: make(map[*websocket.Conn]bool),
}

var broadcastChan = make(chan []byte)
var snapshotWorkerOnce sync.Once

const scorerLockTTL = 45 * time.Second

func scorerLockKey(matchID string) string {
	return "scorer_lock:" + matchID
}

func acquireScorerLock(matchID, owner string) (bool, error) {
	key := scorerLockKey(matchID)
	ok, err := redisImpl.RedisClient.SetNX(context.Background(), key, owner, scorerLockTTL).Result()
	if err != nil {
		return false, err
	}
	return ok, nil
}

func refreshScorerLock(matchID, owner string) {
	key := scorerLockKey(matchID)
	val, err := redisImpl.RedisClient.Get(context.Background(), key).Result()
	if err != nil || val != owner {
		return
	}
	_, _ = redisImpl.RedisClient.Expire(context.Background(), key, scorerLockTTL).Result()
}

func releaseScorerLock(matchID, owner string) {
	key := scorerLockKey(matchID)
	val, err := redisImpl.RedisClient.Get(context.Background(), key).Result()
	if err != nil || val != owner {
		return
	}
	_ = redisImpl.DeleteRedisKey(key)
}

func persistMatchSnapshot(matchID string, match models.EnhancedStatsMessage) {
	snapshotsColl := db.MongoClient.Database("raidx").Collection("match_snapshots")
	_, err := snapshotsColl.UpdateOne(
		context.Background(),
		bson.M{"matchId": matchID},
		bson.M{"$set": bson.M{
			"matchId":           matchID,
			"type":              "ongoing_snapshot",
			"data":              match.Data,
			"lastScoreChangeAt": match.Data.LastScoreChangeAt,
			"updatedAt":         time.Now(),
		}},
		mongoOptionsUpsert(),
	)
	if err != nil {
		logrus.Warnf("snapshot persist failed for match %s: %v", matchID, err)
	}
}

func loadMatchSnapshot(matchID string, target *models.EnhancedStatsMessage) error {
	snapshotsColl := db.MongoClient.Database("raidx").Collection("match_snapshots")
	var snapshot struct {
		MatchID string `bson:"matchId"`
		Data    bson.M `bson:"data"`
	}
	err := snapshotsColl.FindOne(context.Background(), bson.M{"matchId": matchID}).Decode(&snapshot)
	if err == nil {
		raw, mErr := bson.Marshal(snapshot.Data)
		if mErr != nil {
			return mErr
		}
		if uErr := bson.Unmarshal(raw, &target.Data); uErr != nil {
			return uErr
		}
		target.Type = "enhancedStats"
		return nil
	}

	matchesColl := db.MongoClient.Database("raidx").Collection("matches")
	var matchDoc struct {
		Type string `bson:"type"`
		Data bson.M `bson:"data"`
	}
	err = matchesColl.FindOne(context.Background(), bson.M{"matchId": matchID}).Decode(&matchDoc)
	if err != nil {
		return err
	}
	raw, mErr := bson.Marshal(matchDoc.Data)
	if mErr != nil {
		return mErr
	}
	if uErr := bson.Unmarshal(raw, &target.Data); uErr != nil {
		return uErr
	}
	target.Type = "enhancedStats"
	return nil
}

func mongoOptionsUpsert() *options.UpdateOptions {
	upsert := true
	return &options.UpdateOptions{Upsert: &upsert}
}

func startIdleSnapshotWorker() {
	snapshotWorkerOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				keys, err := redisImpl.ListRedisKeys("gameStats:*")
				if err != nil {
					continue
				}
				now := time.Now().Unix()
				for _, key := range keys {
					var current models.EnhancedStatsMessage
					if err := redisImpl.GetRedisKey(key, &current); err != nil {
						continue
					}
					if current.Data.LastScoreChangeAt == 0 {
						current.Data.LastScoreChangeAt = now
						_ = redisImpl.SetRedisKey(key, current)
						continue
					}
					if now-current.Data.LastScoreChangeAt < int64(15*time.Minute/time.Second) {
						continue
					}
					matchID := strings.TrimPrefix(key, "gameStats:")
					persistMatchSnapshot(matchID, current)
				}
			}
		}()
	})
}

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
	startIdleSnapshotWorker()

	// Handle scorer WebSocket
	app.Get("/ws/scorer", websocket.New(func(c *websocket.Conn) {
		// JWT token must be present in query param
		token := c.Query("token")
		claims, err := middleware.AuthWebSocket(token)
		if err != nil {
			logrus.Warn("WebSocket scorer: JWT invalid or missing")
			c.WriteMessage(websocket.TextMessage, []byte(`{"error":"Unauthorized: Invalid JWT"}`))
			c.Close()
			return
		}
		// Expect first message from client to be a join with matchId
		_, joinMsg, err := c.ReadMessage()
		if err != nil {
			logrus.Error("Error:", "SetupWebSocket:", " Failed to read join message: %v", err)
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
				_ = c.WriteMessage(websocket.TextMessage, data)
			}
			return
		}

		matchID := join.MatchID
		scorerOwner := fmt.Sprintf("%v:%v", claims["user_id"], claims["session_id"])
		acquired, lockErr := acquireScorerLock(matchID, scorerOwner)
		if lockErr != nil {
			_ = c.WriteMessage(websocket.TextMessage, []byte(`{"error":"Failed to acquire scorer lock"}`))
			c.Close()
			return
		}
		if !acquired {
			_ = c.WriteMessage(websocket.TextMessage, []byte(`{"error":"This match is already being scored by another active scorer"}`))
			c.Close()
			return
		}

		room := GetRoom(matchID)
		room.AddScorer(c)
		defer func() {
			releaseScorerLock(matchID, scorerOwner)
			room.RemoveScorer(c)
			logrus.Info("Info:", "SetupWebSocket:", " Scorer connection closed")
			c.Close()
		}()

		// ...existing code...

		// send current match state from Redis (per-match key)
		var currentMatch models.EnhancedStatsMessage
		redisKey := "gameStats:" + matchID
		if err := redisImpl.GetRedisKey(redisKey, &currentMatch); err != nil {
			if err == redisImpl.RedisNull {
				if snapErr := loadMatchSnapshot(matchID, &currentMatch); snapErr == nil {
					_ = redisImpl.SetRedisKey(redisKey, currentMatch)
					if data, e := json.Marshal(currentMatch); e == nil {
						_ = c.WriteMessage(websocket.TextMessage, data)
					}
				} else {
					// Ask client to send initial state
					req := map[string]string{"type": "requestInit"}
					if data, e := json.Marshal(req); e == nil {
						_ = c.WriteMessage(websocket.TextMessage, data)
					}
				}
			} else {
				logrus.Error("Error:", "SetupWebSocket:", " Failed to get gameStats for match %s: %v", matchID, err)
			}
		} else {
			if data, err := json.Marshal(currentMatch); err == nil {
				// send to connecting scorer only
				_ = c.WriteMessage(websocket.TextMessage, data)
			}
		}

		// main read loop for this scorer
		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				logrus.Error("Error:", "SetupWebSocket:", " Error reading message from scorer: %v", err)
				break
			}
			refreshScorerLock(matchID, scorerOwner)

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
					if received.Data.LastScoreChangeAt == 0 {
						received.Data.LastScoreChangeAt = time.Now().Unix()
					}
					if err := redisImpl.SetRedisKey(redisKey, received); err == nil {
						persistMatchSnapshot(matchID, received)
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

				prevTeamAScore := currentMatch.Data.TeamA.Score
				prevTeamBScore := currentMatch.Data.TeamB.Score

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
				if currentMatch.Data.TeamA.Score != prevTeamAScore || currentMatch.Data.TeamB.Score != prevTeamBScore {
					currentMatch.Data.LastScoreChangeAt = time.Now().Unix()
				}

				if err := redisImpl.SetRedisKey(redisKey, currentMatch); err != nil {
					logrus.Error("Error:", "SetupWebSocket:", " Failed to set gameStats: %v", err)
					continue
				}
				persistMatchSnapshot(matchID, currentMatch)
				if data, err := json.Marshal(currentMatch); err == nil {
					room.BroadcastBytes(data)
					_ = c.WriteMessage(websocket.TextMessage, data)
				}
				continue
			}

			// Probe for custom non-raid message types (e.g., lobbyTouch)
			var typeProbe struct {
				Type string `json:"type"`
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

				prevTeamAScore := currentMatch.Data.TeamA.Score
				prevTeamBScore := currentMatch.Data.TeamB.Score

				if lobbyPayload.Data.ScoringTeam == "A" {
					currentMatch.Data.TeamA.Score++
				} else {
					currentMatch.Data.TeamB.Score++
				}
				if currentMatch.Data.TeamA.Score != prevTeamAScore || currentMatch.Data.TeamB.Score != prevTeamBScore {
					currentMatch.Data.LastScoreChangeAt = time.Now().Unix()
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
				persistMatchSnapshot(matchID, currentMatch)
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
			if receivedMessage.Data.LastScoreChangeAt == 0 {
				receivedMessage.Data.LastScoreChangeAt = time.Now().Unix()
			}
			persistMatchSnapshot(matchID, receivedMessage)

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

		// Keep connection open
		for {
			if _, _, err := c.NextReader(); err != nil {
				break
			}
		}

		room.RemoveViewer(c)
	}))
}
