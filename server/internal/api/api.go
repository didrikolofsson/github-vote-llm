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
	uh     handlers.UserHandlers
	ah     handlers.AuthHandlers
	oh     handlers.OrganizationHandlers
	gh     handlers.GithubHandlers
	rh     handlers.RepositoryHandlers
	mh     handlers.MembersHandlers
	fh     handlers.FeatureHandlers
}

func NewRestApiRouter(
	env *config.Environment,
	logger *logger.Logger,
	uh handlers.UserHandlers,
	ah handlers.AuthHandlers,
	oh handlers.OrganizationHandlers,
	gh handlers.GithubHandlers,
	rh handlers.RepositoryHandlers,
	mh handlers.MembersHandlers,
	fh handlers.FeatureHandlers,
) RestApiRouter {
	return &RestApiRouterImpl{
		env:    env,
		logger: logger,
		uh:     uh,
		ah:     ah,
		oh:     oh,
		gh:     gh,
		rh:     rh,
		mh:     mh,
		fh:     fh,
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
	auth.POST("/authorize", r.ah.Authorize)
	auth.POST("/token", r.ah.Token)
	auth.POST("/revoke", r.ah.Revoke)

	github := api.Group("/github")
	github.GET("/callback", r.gh.Callback)
	github.Use(middleware.RequireAuth(r.env.JWT_SECRET))
	github.GET("/authorize", r.gh.Authorize)
	github.GET("/status", r.gh.Status)
	github.GET("/repositories", r.gh.ListReposByAuthenticatedUser)
	github.DELETE("/connection", r.gh.Disconnect)

	users := api.Group("/users")
	users.POST("/signup", r.uh.SignupUser)
	users.Use(middleware.RequireAuth(r.env.JWT_SECRET))
	users.GET("/me", r.uh.GetMe)
	users.PATCH("/me/username", r.uh.UpdateUsername)
	users.DELETE("/:id", r.uh.DeleteUser)

	// Organization endpoints
	organizations := api.Group("/organizations")
	organizations.Use(middleware.RequireAuth(r.env.JWT_SECRET))
	organizations.GET("", r.oh.ListMyOrganizations)
	organizations.POST("", r.oh.CreateOrganization)
	organizations.GET("/:id", r.oh.GetOrganization)
	organizations.PUT("/:id", r.oh.UpdateOrganization)
	organizations.DELETE("/:id", r.oh.DeleteOrganization)

	// Organization repositories
	organizations.GET("/:id/repositories", r.rh.List)
	organizations.POST("/:id/repositories", r.rh.Add)
	organizations.DELETE("/:id/repositories/:repoId", r.rh.Remove)

	// Organization members
	organizations.GET("/:id/members", r.mh.List)
	organizations.POST("/:id/members", r.mh.Invite)
	organizations.DELETE("/:id/members/:user_id", r.mh.Remove)
	organizations.PATCH("/:id/members/:user_id", r.mh.UpdateRole)

	// Repository features (all private for now)
	repos := api.Group("/repositories/:repoId")
	repos.Use(middleware.RequireAuth(r.env.JWT_SECRET))
	repos.GET("/roadmap", r.fh.GetRoadmap)
	repos.GET("/features", r.fh.ListFeatures)
	repos.GET("/features/:featureId", r.fh.GetFeature)
	repos.POST("/features", r.fh.CreateFeature)
	repos.DELETE("/features/:featureId", r.fh.DeleteFeature)
	repos.GET("/features/:featureId/comments", r.fh.ListComments)
	repos.POST("/features/:featureId/comments", r.fh.CreateComment)
	repos.POST("/features/:featureId/vote", r.fh.ToggleVote)
	repos.PATCH("/features/:featureId/title", r.fh.UpdateTitle)
	repos.PATCH("/features/:featureId/status", r.fh.UpdateStatus)
	repos.PATCH("/features/:featureId/area", r.fh.UpdateArea)
	repos.PATCH("/features/:featureId/position", r.fh.UpdatePosition)
	repos.POST("/features/:featureId/dependencies", r.fh.AddDependency)
	repos.DELETE("/features/:featureId/dependencies/:dependsOn", r.fh.RemoveDependency)

	return router
}
