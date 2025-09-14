package handlers

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mhatrejeets/RaidX/internal/auth"
	"github.com/mhatrejeets/RaidX/internal/models"
	"github.com/mhatrejeets/RaidX/internal/redisImpl"
	"github.com/sirupsen/logrus"
)

// Per-match viewer clients
var matchViewerClients = struct {
	clients map[string]map[*websocket.Conn]bool // matchID -> conn map
	mu      sync.Mutex
}{
	clients: make(map[string]map[*websocket.Conn]bool),
}

// Per-match broadcast channels
var matchBroadcastChans = struct {
	chans map[string]chan []byte // matchID -> chan
	mu    sync.Mutex
}{
	chans: make(map[string]chan []byte),
}

// Per-match commentary lists
var matchCommentaryLists = struct {
	lists map[string][]string // matchID -> comments
	mu    sync.Mutex
}{
	lists: make(map[string][]string),
}

func StartBroadcastWorker(matchID string) {
       matchBroadcastChans.mu.Lock()
       ch, ok := matchBroadcastChans.chans[matchID]
       if !ok {
	       ch = make(chan []byte)
	       matchBroadcastChans.chans[matchID] = ch
       }
       matchBroadcastChans.mu.Unlock()

       go func() {
	       for msg := range ch {
		       matchViewerClients.mu.Lock()
		       clients, ok := matchViewerClients.clients[matchID]
		       if ok {
			       for conn := range clients {
				       err := conn.WriteMessage(websocket.TextMessage, msg)
				       if err != nil {
					       logrus.Error("Error:", "StartBroadcastWorker:", " Error sending message to viewer: %v", err)
					       conn.Close()
					       delete(clients, conn)
				       }
			       }
		       }
		       matchViewerClients.mu.Unlock()
	       }
       }()
}

func BroadcastToViewers(matchID string, message models.EnhancedStatsMessage) {
       // Enhanced kabaddi commentary
       var newComment string
       if message.Type == "gameStats" {
	       raid := message.Data.RaidDetails
	       switch raid.Type {
	       case "raid":
		       if raid.PointsGained > 0 {
			       if len(raid.Defenders) > 0 {
				       newComment = fmt.Sprintf("Raid SUCCESS! %s scored %d points, got out: %s.", raid.Raider, raid.PointsGained, raid.Defenders)
			       } else {
				       newComment = fmt.Sprintf("Raid SUCCESS! %s scored %d points.", raid.Raider, raid.PointsGained)
			       }
		       } else {
			       newComment = fmt.Sprintf("Empty raid by %s. No points scored.", raid.Raider)
		       }
		       if raid.BonusTaken {
			       newComment += " Bonus taken!"
		       }
		       if raid.SuperTackle {
			       newComment += " Super Tackle!"
		       }
	       case "defence":
		       if len(raid.Defenders) > 0 {
			       newComment = fmt.Sprintf("Defence SUCCESS! %s stopped by %s.", raid.Raider, raid.Defenders)
		       } else {
			       newComment = fmt.Sprintf("Defence SUCCESS! %s stopped.", raid.Raider)
		       }
	       case "empty":
		       newComment = fmt.Sprintf("Empty raid by %s. No points scored.", raid.Raider)
	       default:
		       newComment = fmt.Sprintf("Action by %s. Points: %d.", raid.Raider, raid.PointsGained)
	       }
       }

       // Prepend new comment to the commentary list for this match
       matchCommentaryLists.mu.Lock()
       comments := matchCommentaryLists.lists[matchID]
       if newComment != "" {
	       comments = append([]string{newComment}, comments...)
       }
       if len(comments) > 20 {
	       comments = comments[:20]
       }
       matchCommentaryLists.lists[matchID] = comments
       // Prepare message with all comments
       message.Extra = map[string]interface{}{
	       "commentaryList": comments,
       }
       matchCommentaryLists.mu.Unlock()

       // Broadcast to viewers of this match
       matchBroadcastChans.mu.Lock()
       ch, ok := matchBroadcastChans.chans[matchID]
       matchBroadcastChans.mu.Unlock()
       if ok {
	       data, err := json.Marshal(message)
	       if err != nil {
		       logrus.Error("Error:", "BroadcastToViewers:", " Error marshalling data for viewers: %v", err)
		       return
	       }
	       ch <- data
       }
}


func SetupWebSocket(app *fiber.App) {
       // Handle scorer WebSocket (per match)
       app.Get("/ws/scorer/:matchID", func(c *fiber.Ctx) error {
	       matchID := c.Params("matchID")
	       StartBroadcastWorker(matchID)
	       tokenStr := c.Get("Authorization")
	       if tokenStr == "" {
		       tokenStr = c.Query("token")
	       }
	       if tokenStr == "" {
		       return c.Status(fiber.StatusUnauthorized).SendString("Missing token")
	       }
	       token, err := auth.ParseJWT(tokenStr)
	       if err != nil || !token.Valid {
		       return c.Status(fiber.StatusUnauthorized).SendString("Invalid token")
	       }
	       claims, ok := token.Claims.(jwt.MapClaims)
	       if !ok {
		       return c.Status(fiber.StatusUnauthorized).SendString("Invalid claims")
	       }
	       userID, _ := claims["user_id"].(string)
	       c.Locals("user_id", userID)
	       return websocket.New(func(conn *websocket.Conn) {
		       defer func() {
			       logrus.Info("Info:", "SetupWebSocket:", " Scorer connection closed")
			       conn.Close()
		       }()

		       for {
			       _, msg, err := conn.ReadMessage()
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
			       err = redisImpl.SetRedisKey("gameStats:"+matchID, receivedMessage)
			       if err != nil {
				       logrus.Error("Error:", "SetupWebSocket:", " Error storing data in Redis: %v", err)
			       }

			       // Broadcast updated stats to all viewers of this match
			       BroadcastToViewers(matchID, receivedMessage)
		       }
	       })(c)
       })

       // Handle viewer WebSocket (per match)
       app.Get("/ws/viewer/:matchID", func(c *fiber.Ctx) error {
	       matchID := c.Params("matchID")
	       StartBroadcastWorker(matchID)
	       matchViewerClients.mu.Lock()
	       clients, ok := matchViewerClients.clients[matchID]
	       if !ok {
		       clients = make(map[*websocket.Conn]bool)
		       matchViewerClients.clients[matchID] = clients
	       }
	       matchViewerClients.mu.Unlock()
	       return websocket.New(func(conn *websocket.Conn) {
		       matchViewerClients.mu.Lock()
		       matchViewerClients.clients[matchID][conn] = true
		       matchViewerClients.mu.Unlock()

		       defer func() {
			       matchViewerClients.mu.Lock()
			       delete(matchViewerClients.clients[matchID], conn)
			       matchViewerClients.mu.Unlock()
			       conn.Close()
		       }()

		       // Send latest game stats and commentary list from Redis on new connection
		       var latestStats models.EnhancedStatsMessage
		       err := redisImpl.GetRedisKey("gameStats:"+matchID, &latestStats)
		       if err == nil {
			       matchCommentaryLists.mu.Lock()
			       comments := matchCommentaryLists.lists[matchID]
			       latestStats.Extra = map[string]interface{}{
				       "commentaryList": comments,
			       }
			       matchCommentaryLists.mu.Unlock()
			       data, _ := json.Marshal(latestStats)
			       _ = conn.WriteMessage(websocket.TextMessage, data)
		       }

		       // Keep connection open
		       for {
			       if _, _, err := conn.NextReader(); err != nil {
				       break
			       }
		       }
	       })(c)
       })
}
