package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/nikkofu/studyclaw/api-server/config"
)

const defaultArkBaseURL = "https://ark.cn-beijing.volces.com/api/v3"

var ErrLLMUnavailable = errors.New("llm is unavailable")

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GenerateRequest struct {
	ModelEnvKey  string
	DefaultModel string
	Temperature  float64
	Messages     []Message
}

type Client interface {
	Generate(ctx context.Context, req GenerateRequest) (string, error)
}

type OpenAICompatibleClient struct {
	httpClient *http.Client
}

type chatCompletionRequest struct {
	Model       string    `json:"model"`
	Temperature float64   `json:"temperature,omitempty"`
	Messages    []Message `json:"messages"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func NewOpenAICompatibleClient(httpClient *http.Client) *OpenAICompatibleClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	return &OpenAICompatibleClient{httpClient: httpClient}
}

func getBaseURL() string {
	baseURL := strings.TrimSpace(config.GetEnv("LLM_BASE_URL"))
	if baseURL == "" {
		baseURL = defaultArkBaseURL
	}
	return strings.TrimRight(baseURL, "/")
}

func getModelName(modelEnvKey, defaultModel string) string {
	if modelEnvKey != "" {
		if directValue := strings.TrimSpace(config.GetEnv(modelEnvKey)); directValue != "" {
			return directValue
		}
	}

	if sharedValue := strings.TrimSpace(config.GetEnv("LLM_MODEL_NAME")); sharedValue != "" {
		return sharedValue
	}

	return defaultModel
}

func (c *OpenAICompatibleClient) Generate(ctx context.Context, req GenerateRequest) (string, error) {
	apiKey := strings.TrimSpace(config.GetEnv("LLM_API_KEY"))
	modelName := getModelName(req.ModelEnvKey, req.DefaultModel)
	if apiKey == "" || modelName == "" {
		return "", ErrLLMUnavailable
	}

	payload, err := json.Marshal(chatCompletionRequest{
		Model:       modelName,
		Temperature: req.Temperature,
		Messages:    req.Messages,
	})
	if err != nil {
		return "", fmt.Errorf("marshal llm request: %w", err)
	}

	endpoint := getBaseURL() + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("build llm request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("call llm endpoint: %w", err)
	}
	defer resp.Body.Close()

	var decoded chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", fmt.Errorf("decode llm response: %w", err)
	}

	if resp.StatusCode >= 400 {
		if decoded.Error != nil && strings.TrimSpace(decoded.Error.Message) != "" {
			return "", fmt.Errorf("llm returned %d: %s", resp.StatusCode, decoded.Error.Message)
		}
		return "", fmt.Errorf("llm returned status %d", resp.StatusCode)
	}

	if len(decoded.Choices) == 0 {
		return "", fmt.Errorf("llm returned no choices")
	}

	return strings.TrimSpace(decoded.Choices[0].Message.Content), nil
}
