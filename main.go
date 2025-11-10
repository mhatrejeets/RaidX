package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/mhatrejeets/RaidX/internal/handlers"
	"github.com/mhatrejeets/RaidX/internal/middleware"
	"github.com/mhatrejeets/RaidX/internal/redisImpl"
)

func setupPublicRoutes(app *fiber.App) {
	// Public static pages
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/home.html")
	})
	app.Get("/login", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/login.html")
	})
	app.Get("/signup", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/signup.html")
	})

	// Public auth endpoints
	app.Post("/signup", handlers.SignupHandler)
	app.Post("/login", handlers.LoginHandler)

	// Public viewer access
	app.Get("/viewer", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/viewer.html")
	})
}

func setupProtectedRoutes(app *fiber.App) {
	// Protected pages/views
	app.Get("/scorer", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/scorer.html")
	})
	app.Get("/start", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/startscore.html")
	})
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

	// All matches page for a user (protected)
	app.Get("/allmatches/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		return c.Render("allmatches", fiber.Map{
			"ID": id,
		})
	})

	// Convenience redirect: /allmatches -> /allmatches/:id using token (query or Authorization header)
	app.Get("/allmatches", func(c *fiber.Ctx) error {
		// Try query param first
		token := c.Query("token")
		if token == "" {
			// Try Authorization header
			auth := c.Get("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				token = strings.TrimPrefix(auth, "Bearer ")
			}
		}
		if token == "" {
			// No token: redirect to login
			return c.Redirect("/login")
		}

		// Decode JWT payload (no verification) to extract user id
		parts := strings.Split(token, ".")
		if len(parts) < 2 {
			return c.Redirect("/login")
		}
		payloadB64 := parts[1]
		// Adjust padding for base64
		switch len(payloadB64) % 4 {
		case 2:
			payloadB64 += "=="
		case 3:
			payloadB64 += "="
		}
		decoded, err := base64.URLEncoding.DecodeString(payloadB64)
		if err != nil {
			// attempt standard base64
			decoded, err = base64.StdEncoding.DecodeString(payloadB64)
			if err != nil {
				return c.Redirect("/login")
			}
		}
		var payload map[string]interface{}
		if err := json.Unmarshal(decoded, &payload); err != nil {
			return c.Redirect("/login")
		}
		// Check common fields
		var id string
		if v, ok := payload["user_id"]; ok {
			id = fmt.Sprintf("%v", v)
		} else if v, ok := payload["userId"]; ok {
			id = fmt.Sprintf("%v", v)
		} else if v, ok := payload["sub"]; ok {
			id = fmt.Sprintf("%v", v)
		} else if v, ok := payload["session_id"]; ok {
			id = fmt.Sprintf("%v", v)
		} else if v, ok := payload["sessionId"]; ok {
			id = fmt.Sprintf("%v", v)
		}
		if id == "" {
			return c.Redirect("/login")
		}

		// Redirect to the canonical allmatches/:id route preserving token
		return c.Redirect(fmt.Sprintf("/allmatches/%s?token=%s", id, token))
	})

	// Protected match management endpoints
	app.Get("/endgame", handlers.EndGameHandler)
	app.Get("/matches", handlers.GetAllMatches)
	app.Get("/matches/:id", handlers.GetMatchByID)
	app.Post("/api/matches/raid", handlers.ProcessRaidResult)

	// Protected team management endpoints
	app.Get("/api/team/:id", handlers.GetTeamByID)
	app.Get("/api/teams", handlers.GetTeams)
	app.Get("/createteam/:id", handlers.CreateTeamPage)
	app.Post("/createteam/:id", handlers.SubmitTeam)
}

func main() {
	// Initialize services
	db.InitDB()
	redisImpl.InitRedis()
	app := fiber.New(fiber.Config{
		Views: html.New("./views", ".html"),
	})

	// Setup routes
	setupPublicRoutes(app)

	// WebSocket setup BEFORE applying middleware
	handlers.SetupWebSocket(app)

	// Protected routes - require JWT auth
	app.Use("/scorer", middleware.AuthRequired)
	app.Use("/api", middleware.AuthRequired)
	app.Use("/matches", middleware.AuthRequired)
	app.Use("/allmatches", middleware.AuthRequired)
	app.Use("/playerselection", middleware.AuthRequired)
	app.Use("/matchestype", middleware.AuthRequired)
	app.Use("/selectteams", middleware.AuthRequired)
	app.Use("/createteam", middleware.AuthRequired)
	setupProtectedRoutes(app)

	// Static assets
	app.Static("/static", "./Static")

	// Cleanup
	defer db.CloseDB()

	// Start server on port 3000
	if err := app.Listen("0.0.0.0:3000"); err != nil {
		panic(err)
	}
}
