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
	// Public
	// GitHub OAuth: callback is hit by the browser (no JWT); authorize/status need the logged-in user.
	github.GET("/callback", r.gh.Callback)
	// Private
	github.Use(middleware.RequireAuth(r.env.JWT_SECRET))
	github.GET("/authorize", r.gh.Authorize)
	github.GET("/status", r.gh.Status)
	github.GET("/repositories", r.gh.ListReposByAuthenticatedUser)

	users := api.Group("/users")
	// Public user endpoints
	users.POST("/signup", r.uh.SignupUser)
	// Protected user endpoints
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
	organizations.DELETE("/:id/repositories/:owner/:repo", r.rh.Remove)

	// Organization members
	organizations.GET("/:id/members", r.mh.List)
	organizations.POST("/:id/members", r.mh.Invite)
	organizations.DELETE("/:id/members/:user_id", r.mh.Remove)
	organizations.PATCH("/:id/members/:user_id", r.mh.UpdateRole)

	return router
}
