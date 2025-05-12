package main

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func setupWebSocket(app *fiber.App) {
	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		defer func() {
			log.Println("WebSocket closed")
			c.Close()
		}()

		for {
			// Read message from client
			msgType, msg, err := c.ReadMessage()
			if err != nil {
				log.Println("read error:", err)
				break
			}

			log.Printf("received message: %s\n", msg)
			

			// Echo or process the message
			err = c.WriteMessage(msgType, []byte(fmt.Sprintf("received: %s", msg)))
			if err != nil {
				log.Println("write error:", err)
				break
			}
		}
	}))
}

type PlayerStat struct {
	Name          string `json:"name"`
	ID            string `json:"id"`
	TotalPoints   int    `json:"totalPoints"`
	RaidPoints    int    `json:"raidPoints"`
	DefencePoints int    `json:"defencePoints"`
}

type StatsMessage struct {
	Type string                `json:"type"`
	Data map[string]PlayerStat `json:"data"`
}


