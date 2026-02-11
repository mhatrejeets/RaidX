package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/mhatrejeets/RaidX/internal/db"
	"github.com/mhatrejeets/RaidX/internal/handlers"
	"github.com/mhatrejeets/RaidX/internal/middleware"
	"github.com/mhatrejeets/RaidX/internal/models"
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
	app.Post("/logout", handlers.LogoutHandler)
	app.Post("/logout-all", handlers.LogoutAllDevicesHandler) // Logout from all devices
	app.Post("/refresh", handlers.RefreshTokenHandler)

	// Public viewer access
	app.Get("/viewer", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/viewer.html")
	})

	// Public player profile page (client-side auth gate)
	app.Get("/playerprofile/:id", handlers.PlayerProfileHandler)

	// Public API endpoint to fetch match details by ID (JSON) - no auth required for viewers
	app.Get("/api/match/:id", handlers.GetMatchByIDJSON)

	// Public invite link pages (anyone can visit)
	app.Get("/invite/team/:token", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/invite-team.html")
	})
	app.Get("/invite/event/:token", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/invite-event.html")
	})
	app.Get("/api/invite-link/team/:token/details", handlers.GetTeamInviteLinkDetails)
	app.Get("/api/invite-link/event/:token/details", handlers.GetEventInviteLinkDetails)
}

func setupProtectedRoutes(app *fiber.App) {
	// Role-based dashboards (RBAC only)
	app.Get("/player/dashboard", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/player-dashboard.html")
	})
	app.Get("/owner/dashboard", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/owner-dashboard-v2.html")
	})
	app.Get("/organizer/dashboard", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/organizer-dashboard.html")
	})

	// RBAC: Team Owner APIs
	app.Post("/api/teams", middleware.RoleRequired(models.RoleTeamOwner), handlers.CreateTeamHandler)
	app.Post("/api/teams/:id/invite", middleware.RoleRequired(models.RoleTeamOwner), handlers.CreateTeamInviteHandler)
	app.Get("/api/teams/:id/invites", middleware.RoleRequired(models.RoleTeamOwner), handlers.GetTeamInvitesHandler)
	app.Get("/api/owner/event-invitations", middleware.RoleRequired(models.RoleTeamOwner), handlers.GetOwnerEventInvitesHandler)

	// RBAC: Organizer APIs
	app.Post("/api/events", middleware.RoleRequired(models.RoleOrganizer), handlers.CreateEventHandler)
	app.Put("/api/events/:id", middleware.RoleRequired(models.RoleOrganizer), handlers.UpdateEventHandler)
	app.Post("/api/events/:id/complete", middleware.RoleRequired(models.RoleOrganizer), handlers.MarkEventCompletedHandler)
	app.Post("/api/events/:id/invite", middleware.RoleRequired(models.RoleOrganizer), handlers.CreateEventInviteHandler)
	app.Get("/api/events/:id/teams", middleware.RoleRequired(models.RoleOrganizer), handlers.GetEventTeamsHandler)
	app.Get("/api/organizer/events", middleware.RoleRequired(models.RoleOrganizer), handlers.GetOrganizerEventsHandler)
	app.Get("/api/organizer/event-invites", middleware.RoleRequired(models.RoleOrganizer), handlers.GetOrganizerEventInvitesHandler)

	// RBAC: Invitations (players and team owners)
	app.Put("/api/invitations/:id", middleware.RoleRequired(models.RolePlayer, models.RoleTeamOwner), handlers.UpdateInvitationStatusHandler)
	app.Get("/api/invitations", middleware.RoleRequired(models.RolePlayer), handlers.GetPlayerInvitationsHandler)

	// RBAC: Team Owner - Team Management Pages
	app.Get("/owner/teams", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/owner-teams.html")
	})
	app.Get("/owner/team/:id", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/owner-team-detail.html")
	})
	app.Get("/owner/team/:id/edit", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/owner-team-detail.html")
	})

	// RBAC: Organizer - Event & Match Management Pages
	app.Get("/organizer/events", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/organizer-events.html")
	})
	app.Get("/organizer/event/:id", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/organizer-event-detail.html")
	})
	app.Get("/organizer/event/:id/matches", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/organizer-event-matches.html")
	})
	app.Get("/organizer/match/:id/teams", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/organizer-match-teams.html")
	})

	app.Get("/organizer/profile/:id", handlers.OrganizerProfileHandler)
	app.Get("/organizer/match/:id/scorer", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/organizer-scorer.html")
	})

	// RBAC: Shared Match Viewing (both team owner and organizer can view)
	app.Get("/owner/match/:id/view", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/match-viewer.html")
	})
	app.Get("/organizer/match/:id/view", func(c *fiber.Ctx) error {
		return c.SendFile("./Static/match-viewer.html")
	})

	// RBAC: Shared Match APIs
	app.Get("/api/matches", middleware.RoleRequired(models.RoleTeamOwner, models.RoleOrganizer), handlers.GetAllMatches)
	app.Get("/api/matches/:id", middleware.RoleRequired(models.RoleTeamOwner, models.RoleOrganizer), handlers.GetMatchByID)
	app.Post("/api/matches/raid", middleware.RoleRequired(models.RoleTeamOwner, models.RoleOrganizer), handlers.ProcessRaidResult)

	// RBAC: Invite Link APIs (Team Owner & Organizer)
	app.Post("/api/teams/:id/generate-link", middleware.RoleRequired(models.RoleTeamOwner), handlers.GenerateTeamInviteLink)
	app.Post("/api/events/:id/generate-link", middleware.RoleRequired(models.RoleOrganizer), handlers.GenerateEventInviteLink)
	app.Get("/api/teams/:id/pending-approvals", middleware.RoleRequired(models.RoleTeamOwner), handlers.GetPendingApprovalsForTeam)
	app.Get("/api/events/:id/pending-approvals", middleware.RoleRequired(models.RoleOrganizer), handlers.GetPendingApprovalsForEvent)
	app.Put("/api/pending-approvals/:id/approve", middleware.RoleRequired(models.RoleTeamOwner, models.RoleOrganizer), handlers.ApprovePendingApproval)
	app.Put("/api/pending-approvals/:id/reject", middleware.RoleRequired(models.RoleTeamOwner, models.RoleOrganizer), handlers.RejectPendingApproval)

	// RBAC: Invite Link Accept APIs (Public endpoints that check authentication)
	app.Post("/api/invite-link/team/:token/accept", middleware.AuthRequired, handlers.AcceptTeamInviteLink)
	app.Post("/api/invite-link/team/:token/claim", middleware.AuthRequired, handlers.ClaimTeamInviteLink)
	app.Post("/api/invite-link/event/:token/accept", middleware.AuthRequired, handlers.AcceptEventInviteLink)

	// RBAC: Team Owner - Team Management APIs
	app.Get("/api/owner/teams", middleware.RoleRequired(models.RoleTeamOwner), handlers.GetOwnerTeams)
	app.Get("/api/owner/tournament-requests", middleware.RoleRequired(models.RoleTeamOwner), handlers.GetOwnerTournamentRequests)
	app.Get("/api/teams/:id", middleware.AuthRequired, handlers.GetTeamByIDDetail)
	app.Put("/api/teams/:id", middleware.RoleRequired(models.RoleTeamOwner), handlers.UpdateTeam)
	app.Post("/api/teams/:id/add-player", middleware.RoleRequired(models.RoleTeamOwner), handlers.AddPlayerToTeam)
	app.Delete("/api/teams/:id/remove-player/:playerId", middleware.RoleRequired(models.RoleTeamOwner), handlers.RemovePlayerFromTeam)
	app.Delete("/api/teams/:id", middleware.RoleRequired(models.RoleTeamOwner), handlers.DeleteTeam)
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

	// Protected routes - require JWT auth
	app.Use("/api", middleware.AuthRequired)
	app.Use("/player", middleware.AuthRequired, middleware.RoleRequired(models.RolePlayer))
	app.Use("/owner", middleware.AuthRequired, middleware.RoleRequired(models.RoleTeamOwner))
	app.Use("/organizer", middleware.AuthRequired, middleware.RoleRequired(models.RoleOrganizer))
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
