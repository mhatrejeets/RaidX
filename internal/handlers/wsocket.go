package handlers

import (
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/mhatrejeets/RaidX/internal/models"
	"github.com/mhatrejeets/RaidX/internal/redisImpl"
	"github.com/mhatrejeets/RaidX/internal/middleware"
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
			if err := validateRaidPayload(payload, &currentMatch); err != nil {
				errMsg := map[string]string{"error": err.Error()}
				if b, e := json.Marshal(errMsg); e == nil {
					client.send <- b
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
			}
			if err := redisImpl.SetRedisKey(redisKey, currentMatch); err != nil {
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
			if err := json.Unmarshal(msg, &lobbyPayload); err != nil {
				continue
			}
			var currentMatch models.EnhancedStatsMessage
			if err := redisImpl.GetRedisKey(redisKey, &currentMatch); err != nil {
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
		token := c.Query("token")
		_, err := middleware.AuthWebSocket(token)
		if err != nil {
			resp := map[string]string{"type": "error", "message": "Unauthorized: Invalid or missing JWT token"}
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
		var latestStats models.EnhancedStatsMessage
		redisKey := "gameStats:" + matchID
		err = redisImpl.GetRedisKey(redisKey, &latestStats)
		if err == nil {
			data, _ := json.Marshal(latestStats)
			client.send <- data
		}
		for {
			if _, _, err := c.NextReader(); err != nil {
				break
			}
		}
	}))
}
