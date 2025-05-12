package main

import (
	"github.com/gofiber/fiber/v2"
)

func main() {
	InitDB()	
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/home.html")
	})
	// Serve scorer.html at /scorer
	app.Get("/scorer", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/scorer.html")
	})

	app.Get("/start", func(c *fiber.Ctx) error {
    return c.SendFile("./Static/startscore.html")
	})

	app.Get("/signup", func(c *fiber.Ctx) error {
    return c.SendFile("./Static/signup.html")
	})

	app.Post("/signup", SignupHandler)

	app.Post("/login", LoginHandler)


	defer CloseDB()
	// Serve other static assets like CSS, JS if needed
	app.Static("/static", "./Static")

	// Start server on port 3000
	err := app.Listen(":3000")
	if err != nil {
		panic(err)
	}
}
