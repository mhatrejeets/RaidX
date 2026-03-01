package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/mhatrejeets/RaidX/internal/handlers"
	"github.com/mhatrejeets/RaidX/internal/logger"
	"github.com/mhatrejeets/RaidX/internal/middleware"
	"github.com/mhatrejeets/RaidX/internal/models"
	"github.com/mhatrejeets/RaidX/internal/redisImpl"
)

func main() {
	logger.SetupAppLogging()

	// Initialize services
	db.InitDB()
	redisImpl.InitRedis()
	app := fiber.New(fiber.Config{
		Views: html.New("./views", ".html"),
	})

	// Setup routes
	setupPublicRoutes(app)

	// Protected routes - require JWT auth
	app.Use("/api", middleware.AuthRequired)
	app.Use("/player/", middleware.AuthRequired, middleware.RoleRequired(models.RolePlayer))
	app.Use("/owner/", middleware.AuthRequired, middleware.RoleRequired(models.RoleTeamOwner))
	app.Use("/organizer/", middleware.AuthRequired, middleware.RoleRequired(models.RoleOrganizer))
	setupProtectedRoutes(app)

	// WebSocket setup
	handlers.SetupWebSocket(app)

	// Static assets
	app.Static("/static", "./Static")

	// Cleanup
	defer db.CloseDB()

	// Start server on port 3000
	if err := app.Listen("0.0.0.0:3000"); err != nil {
		panic(err)
	}
}
