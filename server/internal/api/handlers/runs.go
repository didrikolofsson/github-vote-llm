package handlers

import (
	"net/http"
	"strconv"

	"github.com/didrikolofsson/github-vote-llm/internal/jobs/jobargs"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

type RunsHandlers interface {
	Create(c *gin.Context)
}

type RunsHandlersImpl struct {
	s  services.RunService
	jc *river.Client[pgx.Tx]
}

func NewRunsHandlers(s services.RunService, jc *river.Client[pgx.Tx]) RunsHandlers {
	return &RunsHandlersImpl{s: s, jc: jc}
}

type createRunBody struct {
	Prompt          string `json:"prompt"`
	CreatedByUserID int64  `json:"created_by_user_id"`
	Owner           string `json:"owner"`
	Name            string `json:"name"`
}

func (h *RunsHandlersImpl) Create(c *gin.Context) {
	featureID, err := strconv.ParseInt(c.Param("featureId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feature ID"})
		return
	}

	var body createRunBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	run, err := h.s.CreateRun(c.Request.Context(), services.CreateRunParams{
		Prompt:    body.Prompt,
		FeatureID: featureID,
		UserID:    body.CreatedByUserID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	_, err = h.jc.Insert(c.Request.Context(), jobargs.CloneRepoArgs{
		UserID: body.CreatedByUserID,
		RunID:  run.ID,
		Owner:  body.Owner,
		Name:   body.Name,
	}, nil)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enqueue job"})
		return
	}

	c.JSON(http.StatusCreated, run)

	// tx, err := h.s.BeginTx(c.Request.Context(), pgx.TxOptions{})
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	// 	return
	// }
	// defer tx.Rollback(c.Request.Context())

	// run, err := h.s.CreateRun(c.Request.Context(), tx, services.CreateRunParams{
	// 	Prompt:    body.Prompt,
	// 	FeatureID: featureID,
	// 	UserID:    body.CreatedByUserID,
	// })
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	// 	return
	// }

	// if _, err := h.jc.InsertTx(c.Request.Context(), tx, jobargs.CloneRepoArgs{
	// 	UserID: body.CreatedByUserID,
	// 	RunID:  run.ID,
	// 	Owner:  body.Owner,
	// 	Name:   body.Name,
	// }, nil); err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enqueue job"})
	// 	return
	// }

	// if err := tx.Commit(c.Request.Context()); err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	// 	return
	// }

	// c.JSON(http.StatusCreated, run)
}
