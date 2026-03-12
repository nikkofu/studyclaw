package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/nikkofu/studyclaw/api-server/config"
)

const defaultArkBaseURL = "https://ark.cn-beijing.volces.com/api/v3"

var ErrLLMUnavailable = errors.New("llm is unavailable")

type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // Can be string or []ContentPart
}

type ContentPart struct {
	Type     string    `json:"type"` // "text" or "image_url"
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
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
		httpClient = &http.Client{Timeout: resolveHTTPTimeout()}
	}

	return &OpenAICompatibleClient{httpClient: httpClient}
}

func resolveHTTPTimeout() time.Duration {
	rawValue := strings.TrimSpace(config.GetEnv("LLM_HTTP_TIMEOUT_SECONDS"))
	if rawValue == "" {
		return 90 * time.Second
	}

	seconds, err := time.ParseDuration(rawValue + "s")
	if err != nil || seconds <= 0 {
		return 90 * time.Second
	}

	return seconds
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
	startedAt := time.Now()
	log.Printf("[LLM.Client] request_started model=%s endpoint=%s timeout=%s payload_bytes=%d messages=%s", modelName, endpoint, c.httpClient.Timeout, len(payload), summarizeMessages(req.Messages))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("build llm request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		log.Printf("[LLM.Client] request_failed model=%s endpoint=%s duration=%s err=%v", modelName, endpoint, time.Since(startedAt), err)
		return "", fmt.Errorf("call llm endpoint: %w", err)
	}
	defer resp.Body.Close()

	var decoded chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		log.Printf("[LLM.Client] response_decode_failed model=%s status=%d duration=%s err=%v", modelName, resp.StatusCode, time.Since(startedAt), err)
		return "", fmt.Errorf("decode llm response: %w", err)
	}

	if resp.StatusCode >= 400 {
		if decoded.Error != nil && strings.TrimSpace(decoded.Error.Message) != "" {
			log.Printf("[LLM.Client] response_error model=%s status=%d duration=%s message=%q", modelName, resp.StatusCode, time.Since(startedAt), strings.TrimSpace(decoded.Error.Message))
			return "", fmt.Errorf("llm returned %d: %s", resp.StatusCode, decoded.Error.Message)
		}
		log.Printf("[LLM.Client] response_error model=%s status=%d duration=%s", modelName, resp.StatusCode, time.Since(startedAt))
		return "", fmt.Errorf("llm returned status %d", resp.StatusCode)
	}

	if len(decoded.Choices) == 0 {
		log.Printf("[LLM.Client] response_invalid model=%s status=%d duration=%s reason=no_choices", modelName, resp.StatusCode, time.Since(startedAt))
		return "", fmt.Errorf("llm returned no choices")
	}

	content, ok := decoded.Choices[0].Message.Content.(string)
	if !ok {
		log.Printf("[LLM.Client] response_invalid model=%s status=%d duration=%s reason=non_string_content", modelName, resp.StatusCode, time.Since(startedAt))
		return "", fmt.Errorf("llm returned non-string content")
	}

	trimmedContent := strings.TrimSpace(content)
	log.Printf("[LLM.Client] request_completed model=%s status=%d duration=%s response_chars=%d", modelName, resp.StatusCode, time.Since(startedAt), len(trimmedContent))
	return trimmedContent, nil
}

func summarizeMessages(messages []Message) string {
	if len(messages) == 0 {
		return "none"
	}

	parts := make([]string, 0, len(messages))
	for index, message := range messages {
		parts = append(parts, fmt.Sprintf("%d:%s(%s)", index, message.Role, summarizeMessageContent(message.Content)))
	}
	return strings.Join(parts, "; ")
}

func summarizeMessageContent(content interface{}) string {
	switch value := content.(type) {
	case string:
		return fmt.Sprintf("text:%d_chars", len(strings.TrimSpace(value)))
	case []ContentPart:
		partSummaries := make([]string, 0, len(value))
		for _, part := range value {
			switch part.Type {
			case "text":
				partSummaries = append(partSummaries, fmt.Sprintf("text:%d_chars", len(strings.TrimSpace(part.Text))))
			case "image_url":
				urlLength := 0
				if part.ImageURL != nil {
					urlLength = len(strings.TrimSpace(part.ImageURL.URL))
				}
				partSummaries = append(partSummaries, fmt.Sprintf("image_url:%d_chars", urlLength))
			default:
				partSummaries = append(partSummaries, part.Type)
			}
		}
		return strings.Join(partSummaries, ",")
	default:
		return fmt.Sprintf("%T", content)
	}
}
