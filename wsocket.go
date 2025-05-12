package main

import (
	"encoding/json"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

type PlayerStat struct {
	Name          string `json:"name"`
	ID            string `json:"id"`
	TotalPoints   int    `json:"totalPoints"`
	RaidPoints    int    `json:"raidPoints"`
	DefencePoints int    `json:"defencePoints"`
}

type TeamStats struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}

type StatsMessage struct {
	Type string `json:"type"`
	Data struct {
		TeamA       TeamStats             `json:"teamA"`
		TeamB       TeamStats             `json:"teamB"`
		PlayerStats map[string]PlayerStat `json:"playerStats"`
	} `json:"data"`
}

func setupWebSocket(app *fiber.App) {
	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		defer func() {
			log.Println("WebSocket connection closed")
			c.Close()
		}()

		for {
			// Read message from WebSocket client
			_, msg, err := c.ReadMessage()
			if err != nil {
				log.Println("Error reading WebSocket message:", err)
				break
			}

			// Unmarshal the JSON message
			var receivedMessage StatsMessage
			err = json.Unmarshal(msg, &receivedMessage)
			if err != nil {
				log.Println("Error unmarshalling message:", err)
				continue
			}

			// Check if the message type is "gameStats"
			if receivedMessage.Type == "gameStats" {
				log.Println("Game Stats Received:")

				// Log team stats
				log.Printf("Team A: %s, Score: %d\n", receivedMessage.Data.TeamA.Name, receivedMessage.Data.TeamA.Score)
				log.Printf("Team B: %s, Score: %d\n", receivedMessage.Data.TeamB.Name, receivedMessage.Data.TeamB.Score)

				// Log player stats
				log.Println("Player Stats:")
				for playerID, stats := range receivedMessage.Data.PlayerStats {
					log.Printf("Player ID: %s, Name: %s, Total Points: %d, Raid Points: %d, Defence Points: %d\n",
						playerID, stats.Name, stats.TotalPoints, stats.RaidPoints, stats.DefencePoints)
				}
			}
		}
	}))
}
