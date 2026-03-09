package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type PointsRequest struct {
	UserID   uint   `json:"user_id" binding:"required"`
	FamilyID uint   `json:"family_id" binding:"required"`
	Amount   int    `json:"amount" binding:"required"`
	Reason   string `json:"reason" binding:"required"`
}

type PointsHandler struct{}

func NewPointsHandler() *PointsHandler {
	return &PointsHandler{}
}

func (h *PointsHandler) UpdatePoints(c *gin.Context) {
	var req PointsRequest
	if !bindJSONOrAbort(c, &req) {
		return
	}

	log.Printf("[Mock] UpdatePoints bypassed: User %d amount %+d for %s", req.UserID, req.Amount, req.Reason)
	c.JSON(http.StatusOK, gin.H{
		"message": "Points updated successfully (Mock DB)",
		"balance": 100 + req.Amount,
	})
}
