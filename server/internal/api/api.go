package api

import (
	"net/http"

	handlers "github.com/didrikolofsson/github-vote-llm/internal/api/handlers"
	"github.com/didrikolofsson/github-vote-llm/internal/api/middleware"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/gin-gonic/gin"
)

func New(
	h handlers.Handlers,
	logger *logger.Logger,
	jwtSecret string,
) *gin.Engine {
	router := gin.New()

	router.SetTrustedProxies(nil)
	router.Use(middleware.AddRequestID)
	router.Use(middleware.LogRequests(logger))

	api := router.Group("/v1/")
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Authentication endpoints
	auth := api.Group("/auth")
	auth.POST("/authorize", h.Auth.Authorize)
	auth.POST("/token", h.Auth.Token)
	auth.POST("/revoke", h.Auth.Revoke)

	github := api.Group("/github")
	// Public: GitHub redirects here after install — no Bearer token, state nonce carries identity.
	github.GET("/auth/callback", h.Github.Callback)
	github.GET("/app/callback", h.Github.AppCallback)
	github.Use(middleware.RequireAuth(jwtSecret))
	github.GET("/authorize", h.Github.Authorize)
	github.GET("/install", h.Github.Install)
	// github.GET("/status", h.Github.Status)
	// github.GET("/repositories", h.Github.ListRepositories)
	// github.DELETE("/installation", h.Github.Disconnect)

	users := api.Group("/users")
	users.POST("/signup", h.User.SignupUser)
	users.Use(middleware.RequireAuth(jwtSecret))
	users.GET("/me", h.User.GetMe)
	users.PATCH("/me/username", h.User.UpdateUsername)
	users.DELETE("/:id", h.User.DeleteUser)

	// Public portal routes (no auth)
	portal := api.Group("/portal/:orgSlug/:repoName")
	portal.GET("", h.Portal.GetPortalPage)
	portal.GET("/events", h.Portal.Subscribe)
	portal.POST("/features/:featureId/vote", h.Portal.ToggleVote)
	portal.GET("/features/:featureId/comments", h.Portal.ListComments)
	portal.POST("/features/:featureId/comments", h.Portal.CreateComment)

	// Organization endpoints
	organizations := api.Group("/organizations")
	organizations.Use(middleware.RequireAuth(jwtSecret))
	organizations.GET("", h.Organization.ListMyOrganizations)
	organizations.POST("", h.Organization.CreateOrganization)
	organizations.GET("/:id", h.Organization.GetOrganization)
	organizations.PUT("/:id", h.Organization.UpdateOrganization)
	organizations.PATCH("/:id/slug", h.Organization.UpdateSlug)
	organizations.DELETE("/:id", h.Organization.DeleteOrganization)

	// Organization repositories
	organizations.GET("/:id/repositories", h.Repository.List)
	organizations.POST("/:id/repositories", h.Repository.Add)
	organizations.DELETE("/:id/repositories/:repoId", h.Repository.Remove)

	// Organization members
	organizations.GET("/:id/members", h.Members.List)
	organizations.POST("/:id/members", h.Members.Invite)
	organizations.DELETE("/:id/members/:user_id", h.Members.Remove)
	organizations.PATCH("/:id/members/:user_id", h.Members.UpdateRole)

	// Repository features (all private for now)
	repos := api.Group("/repositories/:repoId")
	repos.Use(middleware.RequireAuth(jwtSecret))
	repos.GET("/roadmap", h.Feature.GetRoadmap)
	repos.GET("/meta", h.Repository.GetRepoMeta)
	repos.GET("/features", h.Feature.ListFeatures)
	repos.GET("/features/:featureId", h.Feature.GetFeature)
	repos.POST("/features", h.Feature.CreateFeature)
	repos.DELETE("/features/:featureId", h.Feature.DeleteFeature)
	repos.GET("/features/:featureId/comments", h.Feature.ListComments)
	repos.POST("/features/:featureId/comments", h.Feature.CreateComment)
	repos.POST("/features/:featureId/vote", h.Feature.ToggleVote)
	repos.PATCH("/features/:featureId", h.Feature.PatchFeature)
	repos.PATCH("/features/:featureId/position", h.Feature.UpdatePosition)
	repos.POST("/features/:featureId/dependencies", h.Feature.AddDependency)
	repos.DELETE("/features/:featureId/dependencies/:dependsOn", h.Feature.RemoveDependency)
	repos.PATCH("/portal", h.Repository.UpdatePortalVisibility)

	// Feature runs
	featureRuns := api.Group("/features/:featureId/runs")
	featureRuns.POST("", h.Runs.Create)

	return router
}
