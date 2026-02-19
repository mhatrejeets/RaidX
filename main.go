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

func serveReactApp(c *fiber.Ctx) error {
	return c.SendFile("./frontend/apps/web/dist/index.html")
}

func setupPublicRoutes(app *fiber.App) {
	// Public static pages
	app.Get("/", serveReactApp)
	app.Get("/login", serveReactApp)
	app.Get("/signup", serveReactApp)

	// Public auth endpoints
	app.Post("/signup", handlers.SignupHandler)
	app.Post("/login", handlers.LoginHandler)
	app.Post("/logout", handlers.LogoutHandler)
	app.Post("/logout-all", handlers.LogoutAllDevicesHandler) // Logout from all devices
	app.Post("/refresh", handlers.RefreshTokenHandler)

	// Public viewer access
	app.Get("/viewer", serveReactApp)
	app.Get("/viewer/match/:id", serveReactApp)
	app.Get("/viewer/match/:id/overview", serveReactApp)
	app.Get("/rankings/:type/:id", serveReactApp)
	app.Get("/viewer/tournament/:id", serveReactApp)
	app.Get("/viewer/championship/:id", serveReactApp)

	// Public player profile page (client-side auth gate)
	app.Get("/playerprofile/:id", serveReactApp)

	// Public API endpoint to fetch match details by ID (JSON) - no auth required for viewers
	app.Get("/api/match/:id", handlers.GetMatchByIDJSON)
	app.Get("/api/public/rankings/:type/:id", handlers.GetEventRankingsHandler)
	// Public tournament/championship read-only endpoints for viewers
	app.Get("/api/public/tournaments/:id/fixtures", handlers.GetTournamentFixturesHandler)
	app.Get("/api/public/tournaments/:id/standings", handlers.GetTournamentStandingsHandler)
	app.Get("/api/public/championships/:id", handlers.GetChampionshipByIDHandler)
	app.Get("/api/public/championships/:id/fixtures", handlers.GetChampionshipFixturesHandler)
	app.Get("/api/public/championships/:id/stats", handlers.GetChampionshipStatsHandler)

	// Public invite link pages (anyone can visit)
	app.Get("/invite/team/:token", serveReactApp)
	app.Get("/invite/event/:token", serveReactApp)
	app.Get("/api/invite-link/team/:token/details", handlers.GetTeamInviteLinkDetails)
	app.Get("/api/invite-link/event/:token/details", handlers.GetEventInviteLinkDetails)
}

func setupProtectedRoutes(app *fiber.App) {
	// Authenticated profile API
	app.Get("/api/me/profile", handlers.GetMyProfileHandler)

	// Role-based dashboards (RBAC only)
	app.Get("/player/dashboard", serveReactApp)
	app.Get("/owner/dashboard", serveReactApp)
	app.Get("/organizer/dashboard", serveReactApp)

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
	app.Get("/api/organizer/events/:id", middleware.RoleRequired(models.RoleOrganizer), handlers.GetOrganizerEventDetailHandler)
	app.Get("/api/organizer/events/:id/match", middleware.RoleRequired(models.RoleOrganizer), handlers.GetOrganizerEventMatchStatsHandler)
	app.Get("/api/organizer/event-invites", middleware.RoleRequired(models.RoleOrganizer), handlers.GetOrganizerEventInvitesHandler)
	app.Post("/api/organizer/events/:id/start", middleware.RoleRequired(models.RoleOrganizer), handlers.StartOrganizerEventHandler)

	// RBAC: Tournament APIs
	app.Post("/api/tournaments/initialize/:id", middleware.RoleRequired(models.RoleOrganizer), handlers.InitializeTournamentHandler)
	app.Get("/api/tournaments/:id/fixtures", middleware.RoleRequired(models.RoleOrganizer), handlers.GetTournamentFixturesHandler)
	app.Get("/api/tournaments/:id/standings", middleware.RoleRequired(models.RoleOrganizer), handlers.GetTournamentStandingsHandler)
	app.Post("/api/tournaments/:id/start-match/:fixtureId", middleware.RoleRequired(models.RoleOrganizer), handlers.StartTournamentMatchHandler)

	// RBAC: Championship APIs
	app.Post("/api/championships/initialize/:id", middleware.RoleRequired(models.RoleOrganizer), handlers.InitializeChampionshipHandler)
	app.Get("/api/championships/:id", middleware.RoleRequired(models.RoleOrganizer), handlers.GetChampionshipByIDHandler)
	app.Get("/api/championships/:id/fixtures", middleware.RoleRequired(models.RoleOrganizer), handlers.GetChampionshipFixturesHandler)
	app.Get("/api/championships/:id/stats", middleware.RoleRequired(models.RoleOrganizer), handlers.GetChampionshipStatsHandler)
	app.Post("/api/championships/:id/start-match/:fixtureId", middleware.RoleRequired(models.RoleOrganizer), handlers.StartChampionshipMatchHandler)

	// RBAC: Invitations (players and team owners)
	app.Put("/api/invitations/:id", middleware.RoleRequired(models.RolePlayer, models.RoleTeamOwner), handlers.UpdateInvitationStatusHandler)
	app.Get("/api/invitations", middleware.RoleRequired(models.RolePlayer), handlers.GetPlayerInvitationsHandler)
	app.Get("/api/player/teams", middleware.RoleRequired(models.RolePlayer), handlers.GetPlayerTeamsHandler)
	app.Get("/api/player/events", middleware.RoleRequired(models.RolePlayer), handlers.GetPlayerEventsHandler)

	// RBAC: Team Owner - Team Management Pages
	app.Get("/owner/teams", serveReactApp)
	app.Get("/owner/team/:id", serveReactApp)
	app.Get("/owner/team/:id/edit", serveReactApp)

	// RBAC: Organizer - Event & Match Management Pages
	app.Get("/organizer/events", serveReactApp)
	app.Get("/organizer/event/:id", serveReactApp)
	app.Get("/organizer/tournament", serveReactApp)
	app.Get("/organizer/championship", serveReactApp)
	app.Get("/organizer/event/:id/matches", serveReactApp)
	app.Get("/organizer/match/:id/teams", serveReactApp)
	app.Get("/organizer/match/:id/stats", serveReactApp)

	app.Get("/organizer/profile/:id", serveReactApp)
	app.Get("/owner/profile/:id", serveReactApp)
	app.Get("/organizer/match/:id/scorer", serveReactApp)

	// RBAC: Player Selection & Scorer (Organizer only)
	app.Get("/organizer/playerselection/:id", middleware.AuthRequired, middleware.RoleRequired(models.RoleOrganizer), serveReactApp)
	app.Get("/scorer", middleware.AuthRequired, middleware.RoleRequired(models.RoleOrganizer), serveReactApp)

	// RBAC: Organizer - Event-specific Pending Approvals & Invite Links
	app.Get("/organizer/events/:id/pending-approvals", serveReactApp)
	app.Get("/organizer/event/:id/pending-approvals", serveReactApp)
	app.Get("/organizer/invite-links", serveReactApp)

	// RBAC: Team Owner - Team-specific Pending Approvals & Invite Links
	app.Get("/owner/teams/:id/pending-approvals", serveReactApp)
	app.Get("/owner/invite-links", serveReactApp)

	// RBAC: Shared Match Viewing (both team owner and organizer can view)
	app.Get("/owner/match/:id/view", serveReactApp)
	app.Get("/organizer/match/:id/view", serveReactApp)

	// RBAC: Shared Match APIs
	app.Get("/api/matches", middleware.RoleRequired(models.RoleTeamOwner, models.RoleOrganizer), handlers.GetAllMatches)
	app.Get("/api/matches/:id", middleware.RoleRequired(models.RoleTeamOwner, models.RoleOrganizer), handlers.GetMatchByID)
	app.Post("/api/matches/raid", middleware.RoleRequired(models.RoleTeamOwner, models.RoleOrganizer), handlers.ProcessRaidResult)
	app.Get("/endgame", middleware.AuthRequired, serveReactApp)
	app.Get("/api/endgame", middleware.AuthRequired, handlers.EndGameHandler)

	// RBAC: Invite Link APIs (Team Owner & Organizer)
	app.Post("/api/teams/:id/generate-link", middleware.RoleRequired(models.RoleTeamOwner), handlers.GenerateTeamInviteLink)
	app.Post("/api/events/:id/generate-link", middleware.RoleRequired(models.RoleOrganizer), handlers.GenerateEventInviteLink)
	app.Get("/api/teams/:id/pending-approvals", middleware.RoleRequired(models.RoleTeamOwner), handlers.GetPendingApprovalsForTeam)
	app.Get("/api/events/:id/pending-approvals", middleware.RoleRequired(models.RoleOrganizer), handlers.GetPendingApprovalsForEvent)
	app.Put("/api/pending-approvals/:id/approve", middleware.RoleRequired(models.RoleTeamOwner, models.RoleOrganizer), handlers.ApprovePendingApproval)
	app.Put("/api/pending-approvals/:id/reject", middleware.RoleRequired(models.RoleTeamOwner, models.RoleOrganizer), handlers.RejectPendingApproval)

	// RBAC: Organizer Invite Link Management APIs
	app.Post("/api/organizer/invite-links", middleware.RoleRequired(models.RoleOrganizer), handlers.CreateOrganizerInviteLink)
	app.Get("/api/organizer/invite-links", middleware.RoleRequired(models.RoleOrganizer), handlers.GetOrganizerInviteLinks)
	app.Delete("/api/organizer/invite-links/:id", middleware.RoleRequired(models.RoleOrganizer), handlers.DeleteOrganizerInviteLink)

	// RBAC: Team Owner Invite Link Management APIs
	app.Post("/api/owner/invite-links", middleware.RoleRequired(models.RoleTeamOwner), handlers.CreateOwnerInviteLink)
	app.Get("/api/owner/invite-links", middleware.RoleRequired(models.RoleTeamOwner), handlers.GetOwnerInviteLinks)
	app.Delete("/api/owner/invite-links/:id", middleware.RoleRequired(models.RoleTeamOwner), handlers.DeleteOwnerInviteLink)

	// RBAC: Invite Link Accept APIs (Public endpoints that check authentication)
	app.Post("/api/invite-link/team/:token/accept", middleware.AuthRequired, handlers.AcceptTeamInviteLink)
	app.Post("/api/invite-link/team/:token/claim", middleware.AuthRequired, handlers.ClaimTeamInviteLink)
	app.Post("/api/invite-link/event/:token/accept", middleware.AuthRequired, handlers.AcceptEventInviteLink)
	app.Post("/api/invite-link/event/:token/claim", middleware.AuthRequired, handlers.ClaimEventInviteLink)

	// RBAC: Team Owner - Team Management APIs
	app.Get("/api/owner/teams", middleware.RoleRequired(models.RoleTeamOwner), handlers.GetOwnerTeams)
	app.Get("/api/owner/tournament-requests", middleware.RoleRequired(models.RoleTeamOwner), handlers.GetOwnerTournamentRequests)
	app.Get("/api/teams/:id", middleware.AuthRequired, handlers.GetTeamByIDDetail)
	app.Get("/api/team/:id", middleware.AuthRequired, handlers.GetTeamByIDDetail)
	app.Put("/api/teams/:id", middleware.RoleRequired(models.RoleTeamOwner), handlers.UpdateTeam)
	app.Post("/api/teams/:id/add-player", middleware.RoleRequired(models.RoleTeamOwner), handlers.AddPlayerToTeam)
	app.Delete("/api/teams/:id/remove-player/:playerId", middleware.RoleRequired(models.RoleTeamOwner), handlers.RemovePlayerFromTeam)
	app.Delete("/api/teams/:id", middleware.RoleRequired(models.RoleTeamOwner), handlers.DeleteTeam)
}

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

	// Static assets (legacy + React build)
	app.Static("/static", "./Static")
	app.Static("/assets", "./frontend/apps/web/dist/assets")

	// Cleanup
	defer db.CloseDB()

	// Start server on port 3000
	if err := app.Listen("0.0.0.0:3000"); err != nil {
		panic(err)
	}
}
