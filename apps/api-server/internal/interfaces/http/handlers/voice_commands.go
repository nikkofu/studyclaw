package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nikkofu/studyclaw/api-server/internal/modules/agent/voicecommand"
)

type VoiceCommandHandler struct {
	service *voicecommand.Service
}

type ResolveVoiceCommandReq struct {
	Transcript string                      `json:"transcript" binding:"required"`
	Context    voicecommand.CommandContext `json:"context" binding:"required"`
}

func NewVoiceCommandHandler(service *voicecommand.Service) *VoiceCommandHandler {
	return &VoiceCommandHandler{service: service}
}

func (h *VoiceCommandHandler) Resolve(c *gin.Context) {
	var req ResolveVoiceCommandReq
	if !bindJSONOrAbort(c, &req) {
		return
	}

	log.Printf(
		"[VoiceCommandHandler] Resolve: transcript=%q surface=%s",
		strings.TrimSpace(req.Transcript),
		strings.TrimSpace(req.Context.Surface),
	)

	resolution, err := h.service.Resolve(c.Request.Context(), voicecommand.ResolveInput{
		Transcript: req.Transcript,
		Context:    req.Context,
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to resolve voice command", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"resolution": resolution,
	})
}
