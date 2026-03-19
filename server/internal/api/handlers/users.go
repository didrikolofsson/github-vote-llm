package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type UsersHandlers interface {
	Signup(c *gin.Context)
	Login(c *gin.Context)
	Logout(c *gin.Context)
}

type usersHandlers struct {
}

func NewUsersHandlers() UsersHandlers {
	return &usersHandlers{}
}

func (h *usersHandlers) Signup(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "User signed up"})
}

func (h *usersHandlers) Login(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "User logged in"})
}

func (h *usersHandlers) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "User logged out"})
}
