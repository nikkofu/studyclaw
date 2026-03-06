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
	result := models.DB.Where("phone = ? AND password = ?", req.Phone, req.Password).First(&user)
	if result.Error != nil {
		log.Printf("Login failed for phone %s: %v", req.Phone, result.Error)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// For MVP, return user info as a naive token substitute
	c.JSON(http.StatusOK, gin.H{
		"message":   "Login successful",
		"token":     "fake-jwt-token-for-" + user.Phone,
		"user":      user,
		"family_id": user.FamilyID,
	})
}
