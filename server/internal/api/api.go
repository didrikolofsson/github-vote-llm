package api

import (
	"net/http"

	handlers "github.com/didrikolofsson/github-vote-llm/internal/api/handlers"
	"github.com/didrikolofsson/github-vote-llm/internal/api/middleware"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/gin-gonic/gin"
)

type ApiDeps struct {
	Handlers  handlers.Handlers
	Logger    *logger.Logger
	Queries   *store.Queries
	JwtSecret string
}

func New(
	deps ApiDeps,
) *gin.Engine {
	router := gin.New()

	_ = router.SetTrustedProxies(nil)
	router.Use(middleware.AddRequestID)
	router.Use(middleware.LogRequests(deps.Logger))

	api := router.Group("/v1/")
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Authentication endpoints
	auth := api.Group("/auth")
	auth.POST("/authorize", deps.Handlers.Auth.Authorize)
	auth.POST("/token", deps.Handlers.Auth.Token)
	auth.POST("/revoke", deps.Handlers.Auth.Revoke)

	// GitHub endpoints
	github := api.Group("/github")
	github.GET("/app/callback", deps.Handlers.Github.AppInstallCallback)
	github.POST("/webhooks", deps.Handlers.Github.HandleWebhook)
	github.Use(middleware.RequireAuth(deps.JwtSecret))

	// User endpoints
	users := api.Group("/users")
	users.POST("/signup", deps.Handlers.User.SignupUser)
	users.Use(middleware.RequireAuth(deps.JwtSecret))
	users.GET("/me", deps.Handlers.User.GetMe)
	users.PATCH("/me/username", deps.Handlers.User.UpdateUsername)
	users.DELETE("/:id", deps.Handlers.User.DeleteUser)

	// Public portal routes (no auth)
	portal := api.Group("/portal/:orgSlug/:repoName")
	portal.GET("", deps.Handlers.Portal.GetPortalPage)
	portal.GET("/events", deps.Handlers.Portal.Subscribe)
	portal.POST("/features/:featureId/vote", deps.Handlers.Portal.ToggleVote)
	portal.GET("/features/:featureId/comments", deps.Handlers.Portal.ListComments)
	portal.POST("/features/:featureId/comments", deps.Handlers.Portal.CreateComment)

	// Organization endpoints
	organizations := api.Group("/organizations")
	organizations.Use(middleware.RequireAuth(deps.JwtSecret))
	organizations.GET("", deps.Handlers.Organization.ListMyOrganizations)
	organizations.POST("", deps.Handlers.Organization.CreateOrganization)
	organizations.Use(middleware.RequireOrgMember(deps.Queries))
	organizations.GET("/:id", deps.Handlers.Organization.GetOrganization)
	organizations.PUT("/:id", deps.Handlers.Organization.UpdateOrganization)
	organizations.PATCH("/:id/slug", deps.Handlers.Organization.UpdateSlug)
	organizations.DELETE("/:id", deps.Handlers.Organization.DeleteOrganization)

	// GitHub App installation (per org)
	organizations.GET("/:id/github-app/install-url", deps.Handlers.Github.GetAppInstallURL)
	organizations.GET("/:id/github-app/status", deps.Handlers.Github.GetAppInstallationStatus)

	// Organization repositories
	organizations.GET("/:id/repositories", deps.Handlers.Repository.List)
	organizations.POST("/:id/repositories", deps.Handlers.Repository.Add)
	organizations.DELETE("/:id/repositories/:repoId", deps.Handlers.Repository.Remove)

	// Org-scoped GitHub repositories
	organizations.GET("/:id/github/repositories", deps.Handlers.Github.ListInstallationRepositories)

	// Organization members
	organizations.GET("/:id/members", deps.Handlers.Members.List)
	organizations.POST("/:id/members", deps.Handlers.Members.Invite)
	organizations.DELETE("/:id/members/:user_id", deps.Handlers.Members.Remove)
	organizations.PATCH("/:id/members/:user_id", deps.Handlers.Members.UpdateRole)

	// Org-scoped SSE — auth via ?access_token= since EventSource cannot send headers
	api.GET("/organizations/:id/events", middleware.RequireAuthFromQueryOrHeader(deps.JwtSecret), deps.Handlers.Github.Events)

	// Repository features (all private for now)
	repos := api.Group("/repositories/:repoId")
	repos.Use(middleware.RequireAuth(deps.JwtSecret))
	repos.GET("/roadmap", deps.Handlers.Feature.GetRoadmap)
	repos.GET("/meta", deps.Handlers.Repository.GetRepoMeta)
	repos.GET("/runs", deps.Handlers.Runs.ListByRepository)
	repos.GET("/features", deps.Handlers.Feature.ListFeatures)
	repos.GET("/features/:featureId", deps.Handlers.Feature.GetFeature)
	repos.POST("/features", deps.Handlers.Feature.CreateFeature)
	repos.DELETE("/features/:featureId", deps.Handlers.Feature.DeleteFeature)
	repos.GET("/features/:featureId/comments", deps.Handlers.Feature.ListComments)
	repos.POST("/features/:featureId/comments", deps.Handlers.Feature.CreateComment)
	repos.POST("/features/:featureId/vote", deps.Handlers.Feature.ToggleVote)
	repos.PATCH("/features/:featureId", deps.Handlers.Feature.PatchFeature)
	repos.PATCH("/features/:featureId/position", deps.Handlers.Feature.UpdatePosition)
	repos.POST("/features/:featureId/dependencies", deps.Handlers.Feature.AddDependency)
	repos.DELETE("/features/:featureId/dependencies/:dependsOn", deps.Handlers.Feature.RemoveDependency)
	repos.PATCH("/portal", deps.Handlers.Repository.UpdatePortalVisibility)

	// Feature runs
	featureRuns := api.Group("/features/:featureId/runs")
	featureRuns.POST("", deps.Handlers.Runs.Create)

	return router
}
