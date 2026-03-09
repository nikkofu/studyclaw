package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type userResponse struct {
	ID        uint      `json:"id"`
	FamilyID  uint      `json:"family_id"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	Phone     string    `json:"phone"`
	Points    int       `json:"points"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AuthHandler struct{}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	log.Printf("Mock Login bypassed for phone %s", req.Phone)
	user := userResponse{
		ID:       1,
		FamilyID: 101,
		Name:     "Mock User",
		Role:     "parent",
		Phone:    req.Phone,
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Login successful",
		"token":     "fake-jwt-token-for-" + user.Phone,
		"user":      user,
		"family_id": user.FamilyID,
	})
}
