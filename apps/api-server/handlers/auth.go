package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nikkofu/studyclaw/api-server/models"
)

// Simplified Authentication for MVP
// In a real app, use bcrypt and JWT.

type LoginRequest struct {
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	var user models.User
	// Mock User for testing without DB
	log.Printf("Mock Login bypassed for phone %s", req.Phone)
	user = models.User{
		ID:       1,
		FamilyID: 101,
		Name:     "Mock User",
		Role:     "parent",
		Phone:    req.Phone,
	}

	// For MVP, return user info as a naive token substitute
	c.JSON(http.StatusOK, gin.H{
		"message":   "Login successful",
		"token":     "fake-jwt-token-for-" + user.Phone,
		"user":      user,
		"family_id": user.FamilyID,
	})
}
