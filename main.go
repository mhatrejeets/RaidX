package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/mhatrejeets/RaidX/internal/handlers"
	"github.com/mhatrejeets/RaidX/internal/redisImpl"
)

func main() {
	db.InitDB()
	redisImpl.InitRedis()
	app := fiber.New(fiber.Config{
		Views: html.New("./views", ".html"), // Set the directory and extension for templates
	})
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/home.html")
	})

	app.Get("/login", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/login.html")
	})

	// Serve scorer.html at /scorer
	app.Get("/scorer", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/scorer.html")
	})

	app.Get("/api/team/:id", handlers.GetTeamByID)

	app.Get("/api/teams", handlers.GetTeams)

	app.Get("/start", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/startscore.html")
	})

	app.Get("/signup", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/signup.html")
	})

	app.Get("/viewer", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/viewer.html")
	})

	app.Post("/signup", handlers.SignupHandler)

	app.Post("/login", handlers.LoginHandler)

	app.Get("/home1/:id", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/home1.html")
	})

	app.Get("/playerprofile/:id", handlers.PlayerProfileHandler)

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

	app.Get("/endgame", handlers.EndGameHandler)

	app.Get("/matches", handlers.GetAllMatches)
	app.Get("/matches/:id", handlers.GetMatchByID)
	app.Get("/createteam/:id", handlers.CreateTeamPage)
	app.Post("/createteam/:id", handlers.SubmitTeam)

	handlers.SetupWebSocket(app)
	app.Get("/api/matches/ongoing", handlers.GetOngoingMatches)

	defer db.CloseDB()
	// Serve other static assets like CSS, JS if needed
	app.Static("/static", "./Static")

	// Start server on port 3000
	err := app.Listen(":3000")
	if err != nil {
		panic(err)
	}
}
