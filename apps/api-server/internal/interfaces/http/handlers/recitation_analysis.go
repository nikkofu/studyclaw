package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nikkofu/studyclaw/api-server/internal/modules/agent/recitationanalysis"
)

type RecitationAnalysisHandler struct {
	service *recitationanalysis.Service
}

type AnalyzeRecitationReq struct {
	Transcript    string            `json:"transcript" binding:"required"`
	Scene         string            `json:"scene,omitempty"`
	Locale        string            `json:"locale,omitempty"`
	ReferenceText string            `json:"reference_text,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

func NewRecitationAnalysisHandler(service *recitationanalysis.Service) *RecitationAnalysisHandler {
	return &RecitationAnalysisHandler{service: service}
}

func (h *RecitationAnalysisHandler) Analyze(c *gin.Context) {
	var req AnalyzeRecitationReq
	if !bindJSONOrAbort(c, &req) {
		return
	}

	log.Printf(
		"[RecitationAnalysisHandler] Analyze: scene=%s transcript_chars=%d reference_chars=%d",
		strings.TrimSpace(req.Scene),
		len([]rune(strings.TrimSpace(req.Transcript))),
		len([]rune(strings.TrimSpace(req.ReferenceText))),
	)

	analysis, err := h.service.Analyze(c.Request.Context(), recitationanalysis.AnalyzeInput{
		Transcript:    req.Transcript,
		Scene:         req.Scene,
		Locale:        req.Locale,
		ReferenceText: req.ReferenceText,
		Metadata:      req.Metadata,
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to analyze recitation", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"analysis": analysis,
	})
}
