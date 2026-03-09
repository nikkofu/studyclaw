package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nikkofu/studyclaw/api-server/config"
)

var agentCoreHTTPClient = &http.Client{Timeout: 10 * time.Second}

type ParsedTask struct {
	Subject     string   `json:"subject"`
	GroupTitle  string   `json:"group_title,omitempty"`
	Title       string   `json:"title"`
	Type        string   `json:"type"`
	Confidence  float64  `json:"confidence,omitempty"`
	NeedsReview bool     `json:"needs_review,omitempty"`
	Notes       []string `json:"notes,omitempty"`
}

type ParseParentInputResp struct {
	Status     string                 `json:"status"`
	Data       []ParsedTask           `json:"data"`
	Message    string                 `json:"message,omitempty"`
	ParserMode string                 `json:"parser_mode,omitempty"`
	Analysis   map[string]interface{} `json:"analysis,omitempty"`
}

func getAgentCoreBaseURL() string {
	baseURL := strings.TrimRight(config.GetEnv("AGENT_CORE_URL"), "/")
	if baseURL != "" {
		return baseURL
	}

	port := config.GetEnv("AGENT_PORT")
	if port == "" {
		port = "8000"
	}

	return "http://localhost:" + port
}

func postToAgentCore(path string, payload any, target any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal agent payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, getAgentCoreBaseURL()+path, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("build agent request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := agentCoreHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("call agent core: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read agent response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("agent core returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	if err := json.Unmarshal(respBody, target); err != nil {
		return fmt.Errorf("decode agent response: %w", err)
	}

	return nil
}

func ParseParentInput(rawText string) (ParseParentInputResp, error) {
	var resp ParseParentInputResp
	err := postToAgentCore("/api/v1/internal/parse", map[string]string{
		"raw_text": rawText,
	}, &resp)
	return resp, err
}

func AnalyzeWeeklyStats(daysData []map[string]interface{}) (map[string]interface{}, error) {
	var resp map[string]interface{}
	err := postToAgentCore("/api/v1/internal/analyze/weekly", map[string]interface{}{
		"days_data": daysData,
	}, &resp)
	return resp, err
}
