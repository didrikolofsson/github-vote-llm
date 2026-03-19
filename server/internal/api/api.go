package api

import (
	"net/http"

	"github.com/didrikolofsson/github-vote-llm/internal/api/handlers"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/gin-gonic/gin"
)

type RestApiRouter interface {
	Create() *gin.Engine
}

type restApiRouter struct {
	logger        *logger.Logger
	usersHandlers handlers.UsersHandlers
}

func RestApiRouterFactory(logger *logger.Logger, usersHandlers handlers.UsersHandlers) RestApiRouter {
	return &restApiRouter{
		logger:        logger,
		usersHandlers: usersHandlers,
	}
}

func (r *restApiRouter) Create() *gin.Engine {
	router := gin.New()
	router.SetTrustedProxies(nil)

	api := router.Group("/v1/")

	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Users endpoints
	users := api.Group("/users")
	users.POST("/signup", r.usersHandlers.Signup)
	users.POST("/login", r.usersHandlers.Login)
	users.POST("/logout", r.usersHandlers.Logout)

	return router
}

// func SetupAPIRouter(router *gin.Engine, logger *logger.Logger, handlers *api_handlers.ApiHandlers, env *config.Environment) {
// 	logger.Infow("Setting up API router")

// 	api := router.Group("/v1/api")

// 	api.GET("/health", func(c *gin.Context) {
// 		c.JSON(http.StatusOK, gin.H{"status": "ok"})
// 	})

// 	api.Use(api_middleware.ValidateAPIKey(env.API_KEY))

// 	api.GET("/runs", handlers.Runs.List)
// 	api.POST("/runs", handlers.Runs.Create)
// 	api.GET("/runs/:id", handlers.Runs.Get)
// 	api.POST("/runs/:id/retry", handlers.Runs.Retry)
// 	api.POST("/runs/:id/cancel", handlers.Runs.Cancel)
// 	api.GET("/repos", handlers.Repos.List)
// 	api.GET("/repos/:owner/:repo/config", handlers.Repos.GetConfig)
// 	api.PUT("/repos/:owner/:repo/config", handlers.Repos.UpdateConfig)
// 	api.DELETE("/repos/:owner/:repo/config", handlers.Repos.DeleteConfig)
// 	api.GET("/repos/:owner/:repo/roadmap", handlers.Repos.ListRoadmapItems)
// 	api.PATCH("/repos/:owner/:repo/proposals/:id", handlers.Repos.UpdateProposalStatus)
// }

// func SetupPublicBoardRouter(router *gin.Engine, handlers *api_handlers.ApiHandlers) *gin.Engine {
// 	board := router.Group("/board")
// 	board.GET("/:owner/:repo/proposals", handlers.Board.ListProposals)
// 	board.POST("/:owner/:repo/proposals", handlers.Board.CreateProposal)
// 	board.POST("/:owner/:repo/proposals/:id/vote", handlers.Board.VoteProposal)
// 	board.GET("/:owner/:repo/proposals/:id/comments", handlers.Board.ListComments)
// 	board.POST("/:owner/:repo/proposals/:id/comments", handlers.Board.CreateComment)
// 	return router
// }
