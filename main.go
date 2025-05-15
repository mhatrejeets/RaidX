package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
)

func main() {
	InitDB()
	InitRedis()
	app := fiber.New(fiber.Config{
		Views: html.New("./views", ".html"), // Set the directory and extension for templates
	})
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/home.html")
	})

	app.Get("/login", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/login.html")
	})

	app.Get("/home1", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/home1.html")
	})

	// Serve scorer.html at /scorer
	app.Get("/scorer", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/scorer.html")
	})

	app.Get("/selectteam", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/selectteams.html")
	})

	app.Get("/api/team/:id", getTeamByID)

	app.Get("/api/teams", getTeams)

	app.Get("/start", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/startscore.html")
	})

	app.Get("/signup", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/signup.html")
	})

	app.Post("/signup", SignupHandler)

	app.Post("/login", LoginHandler)

	app.Get("/home1/:id", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/home1.html")
	})

	app.Get("/playerprofile/:id", playerprofileHandler)

	app.Get("/playerselection/:id", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/playerselection.html")
	})

	app.Get("/matchestype/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		return c.Render("matches_type", fiber.Map{
			"ID": id,
		})
	})

	app.Get("/selectteams/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		return c.Render("selectteams", fiber.Map{
			"ID": id,
		})
	})

	setupWebSocket(app)

	defer CloseDB()
	// Serve other static assets like CSS, JS if needed
	app.Static("/static", "./Static")

	// Start server on port 3000
	err := app.Listen(":3000")
	if err != nil {
		panic(err)
	}
}
