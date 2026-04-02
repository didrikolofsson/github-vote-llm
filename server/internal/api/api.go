package api

import (
	"net/http"

	handlers "github.com/didrikolofsson/github-vote-llm/internal/api/handlers"
	"github.com/didrikolofsson/github-vote-llm/internal/api/middleware"
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/gin-gonic/gin"
)

type RestApiRouter interface {
	Create() *gin.Engine
}

type RestApiRouterImpl struct {
	env    *config.Environment
	logger *logger.Logger
	h      *handlers.HandlerCollection
}

func NewRestApiRouter(
	env *config.Environment,
	logger *logger.Logger,
	h *handlers.HandlerCollection,
) RestApiRouter {
	return &RestApiRouterImpl{
		env:    env,
		logger: logger,
		h:      h,
	}
}

func (r *RestApiRouterImpl) Create() *gin.Engine {
	router := gin.New()

	router.SetTrustedProxies(nil)
	router.Use(middleware.AddRequestID)
	router.Use(middleware.LogRequests(r.logger))

	api := router.Group("/v1/")
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// OAuth2 endpoints
	auth := api.Group("/auth")
	auth.POST("/authorize", r.h.Auth.Authorize)
	auth.POST("/token", r.h.Auth.Token)
	auth.POST("/revoke", r.h.Auth.Revoke)

	github := api.Group("/github")
	github.GET("/callback", r.h.Github.Callback)
	github.Use(middleware.RequireAuth(r.env.JWT_SECRET))
	github.GET("/authorize", r.h.Github.Authorize)
	github.GET("/status", r.h.Github.Status)
	github.GET("/repositories", r.h.Github.ListReposByAuthenticatedUser)
	github.DELETE("/connection", r.h.Github.Disconnect)

	users := api.Group("/users")
	users.POST("/signup", r.h.User.SignupUser)
	users.Use(middleware.RequireAuth(r.env.JWT_SECRET))
	users.GET("/me", r.h.User.GetMe)
	users.PATCH("/me/username", r.h.User.UpdateUsername)
	users.DELETE("/:id", r.h.User.DeleteUser)

	// Public portal routes (no auth)
	portal := api.Group("/portal/:orgSlug/:repoName")
	portal.GET("", r.h.Portal.GetPortalPage)
	portal.GET("/events", r.h.Portal.Subscribe)
	portal.POST("/features/:featureId/vote", r.h.Portal.ToggleVote)
	portal.GET("/features/:featureId/comments", r.h.Portal.ListComments)
	portal.POST("/features/:featureId/comments", r.h.Portal.CreateComment)

	// Organization endpoints
	organizations := api.Group("/organizations")
	organizations.Use(middleware.RequireAuth(r.env.JWT_SECRET))
	organizations.GET("", r.h.Organization.ListMyOrganizations)
	organizations.POST("", r.h.Organization.CreateOrganization)
	organizations.GET("/:id", r.h.Organization.GetOrganization)
	organizations.PUT("/:id", r.h.Organization.UpdateOrganization)
	organizations.PATCH("/:id/slug", r.h.Organization.UpdateSlug)
	organizations.DELETE("/:id", r.h.Organization.DeleteOrganization)

	// Organization repositories
	organizations.GET("/:id/repositories", r.h.Repository.List)
	organizations.POST("/:id/repositories", r.h.Repository.Add)
	organizations.DELETE("/:id/repositories/:repoId", r.h.Repository.Remove)

	// Organization members
	organizations.GET("/:id/members", r.h.Members.List)
	organizations.POST("/:id/members", r.h.Members.Invite)
	organizations.DELETE("/:id/members/:user_id", r.h.Members.Remove)
	organizations.PATCH("/:id/members/:user_id", r.h.Members.UpdateRole)

	// Repository features (all private for now)
	repos := api.Group("/repositories/:repoId")
	repos.Use(middleware.RequireAuth(r.env.JWT_SECRET))
	repos.GET("/roadmap", r.h.Feature.GetRoadmap)
	repos.GET("/meta", r.h.Repository.GetRepoMeta)
	repos.GET("/features", r.h.Feature.ListFeatures)
	repos.GET("/features/:featureId", r.h.Feature.GetFeature)
	repos.POST("/features", r.h.Feature.CreateFeature)
	repos.DELETE("/features/:featureId", r.h.Feature.DeleteFeature)
	repos.GET("/features/:featureId/comments", r.h.Feature.ListComments)
	repos.POST("/features/:featureId/comments", r.h.Feature.CreateComment)
	repos.POST("/features/:featureId/vote", r.h.Feature.ToggleVote)
	repos.PATCH("/features/:featureId", r.h.Feature.PatchFeature)
	repos.PATCH("/features/:featureId/position", r.h.Feature.UpdatePosition)
	repos.POST("/features/:featureId/dependencies", r.h.Feature.AddDependency)
	repos.DELETE("/features/:featureId/dependencies/:dependsOn", r.h.Feature.RemoveDependency)
	repos.PATCH("/portal", r.h.Repository.UpdatePortalVisibility)

	return router
}
