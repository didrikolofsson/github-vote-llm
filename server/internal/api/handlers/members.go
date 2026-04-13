package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/didrikolofsson/github-vote-llm/internal/api/middleware"
	"github.com/didrikolofsson/github-vote-llm/internal/api/request"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/gin-gonic/gin"
)

type MembersHandlers interface {
	List(c *gin.Context)
	Invite(c *gin.Context)
	Remove(c *gin.Context)
	UpdateRole(c *gin.Context)
}

type MembersHandlersImpl struct {
	s services.MembersService
	l *logger.Logger
}

func NewMembersHandlers(s services.MembersService, l *logger.Logger) MembersHandlers {
	return &MembersHandlersImpl{s: s, l: l}
}

func (h *MembersHandlersImpl) List(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	members, err := h.s.ListMembers(c.Request.Context(), orgID, userID)
	if errors.Is(err, services.ErrNotOrgMember) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this organization"})
		return
	}
	if err != nil {
		h.l.Errorw("Failed to list members", "error", err, "organization_id", orgID, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"members": members})
}

type inviteRequest struct {
	Email string `json:"email" binding:"required,email"`
}

func (h *MembersHandlersImpl) Invite(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	var req inviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	err = h.s.InviteByEmail(c.Request.Context(), orgID, userID, req.Email)
	if errors.Is(err, services.ErrNotOrgMember) {
		c.JSON(http.StatusForbidden, gin.H{"error": "only owners can invite members"})
		return
	}
	if errors.Is(err, services.ErrInviteUserNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found — they need to sign up first"})
		return
	}
	if errors.Is(err, services.ErrUserAlreadyInOrg) {
		c.JSON(http.StatusConflict, gin.H{"error": "user is already in an organization"})
		return
	}
	if err != nil {
		h.l.Errorw("Failed to invite member", "error", err, "organization_id", orgID, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.Status(http.StatusCreated)
}

func (h *MembersHandlersImpl) Remove(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	memberUserID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	err = h.s.RemoveMember(c.Request.Context(), orgID, userID, memberUserID)
	if errors.Is(err, services.ErrNotOrgMember) {
		c.JSON(http.StatusForbidden, gin.H{"error": "only owners can remove members"})
		return
	}
	if errors.Is(err, services.ErrCannotRemoveOwner) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot remove the last owner"})
		return
	}
	if err != nil {
		h.l.Errorw("Failed to remove member", "error", err, "organization_id", orgID, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.Status(http.StatusNoContent)
}

type updateRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=owner member"`
}

func (h *MembersHandlersImpl) UpdateRole(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	memberUserID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req updateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	err = h.s.UpdateRole(c.Request.Context(), orgID, userID, memberUserID, req.Role)
	if errors.Is(err, services.ErrNotOrgMember) {
		c.JSON(http.StatusForbidden, gin.H{"error": "only owners can change roles"})
		return
	}
	if errors.Is(err, services.ErrCannotChangeOwnRole) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "you cannot change your own role"})
		return
	}
	if errors.Is(err, services.ErrCannotRemoveOwner) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot downgrade the last owner"})
		return
	}
	if errors.Is(err, services.ErrMemberNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "member not found"})
		return
	}
	if err != nil {
		h.l.Errorw("Failed to update role", "error", err, "organization_id", orgID, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.Status(http.StatusNoContent)
}
