package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/didrikolofsson/github-vote-llm/internal/api/request"
	"github.com/didrikolofsson/github-vote-llm/internal/api/services"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/gin-gonic/gin"
)

type OrganizationHandlers interface {
	CreateOrganization(c *gin.Context)
	GetOrganization(c *gin.Context)
	ListMyOrganizations(c *gin.Context)
	UpdateOrganization(c *gin.Context)
	UpdateSlug(c *gin.Context)
	DeleteOrganization(c *gin.Context)
}

type OrganizationHandlersImpl struct {
	s services.OrganizationService
	l *logger.Logger
}

func NewOrganizationHandlers(s services.OrganizationService, l *logger.Logger) OrganizationHandlers {
	return &OrganizationHandlersImpl{s: s, l: l}
}

type createOrganizationRequest struct {
	Name string `json:"name" binding:"required"`
	Slug string `json:"slug"` // optional; auto-generated from Name when empty
}

type updateOrganizationSlugRequest struct {
	Slug string `json:"slug" binding:"required"`
}

type updateOrganizationRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *OrganizationHandlersImpl) CreateOrganization(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	uid, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	var req createOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Errorw("Invalid request body", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	org, err := h.s.CreateOrganization(c.Request.Context(), services.CreateOrganizationParams{
		Name:    req.Name,
		Slug:    req.Slug,
		OwnerID: uid,
	})
	if errors.Is(err, services.ErrUserAlreadyInOrganization) {
		h.l.Warnw("User already in organization", "user_id", uid, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusBadRequest, gin.H{"error": "you already belong to an organization"})
		return
	}
	if errors.Is(err, services.ErrOrganizationNameExists) {
		h.l.Warnw("Organization name exists", "name", req.Name, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusBadRequest, gin.H{"error": "organization name already exists"})
		return
	}
	if err != nil {
		h.l.Errorw("Failed to create organization", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, org)
}

func (h *OrganizationHandlersImpl) ListMyOrganizations(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	uid, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	orgs, err := h.s.ListOrganizationsForUser(c.Request.Context(), uid)
	if err != nil {
		h.l.Errorw("Failed to list organizations", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"organizations": orgs})
}

func (h *OrganizationHandlersImpl) GetOrganization(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	org, err := h.s.GetOrganizationByID(c.Request.Context(), id)
	if errors.Is(err, services.ErrOrganizationNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		return
	}
	if err != nil {
		h.l.Errorw("Failed to get organization", "error", err, "organization_id", id, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, org)
}

func (h *OrganizationHandlersImpl) UpdateOrganization(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	var req updateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Errorw("Invalid request body", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	org, err := h.s.UpdateOrganizationByID(c.Request.Context(), id, &store.UpdateOrganizationByIDParams{
		ID:   id,
		Name: req.Name,
	})
	if errors.Is(err, services.ErrOrganizationNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		return
	}
	if errors.Is(err, services.ErrOrganizationNameExists) {
		h.l.Warnw("Organization name exists", "name", req.Name, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusBadRequest, gin.H{"error": "organization name already exists"})
		return
	}
	if err != nil {
		h.l.Errorw("Failed to update organization", "error", err, "organization_id", id, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, org)
}

func (h *OrganizationHandlersImpl) UpdateSlug(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	var req updateOrganizationSlugRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	org, err := h.s.UpdateOrganizationSlug(c.Request.Context(), id, req.Slug)
	if errors.Is(err, services.ErrOrganizationNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		return
	}
	if errors.Is(err, services.ErrOrganizationSlugExists) {
		c.JSON(http.StatusConflict, gin.H{"error": "organization slug already exists"})
		return
	}
	if err != nil {
		h.l.Errorw("Failed to update organization slug", "error", err, "organization_id", id, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, org)
}

func (h *OrganizationHandlersImpl) DeleteOrganization(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	if err := h.s.DeleteOrganization(c.Request.Context(), id); err != nil {
		h.l.Errorw("Failed to delete organization", "error", err, "organization_id", id, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusNoContent)
}
