package llm

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nikkofu/studyclaw/api-server/config"
)

func TestOpenAICompatibleClientLiveMultimodal(t *testing.T) {
	if os.Getenv("RUN_LIVE_LLM_TEST") != "1" {
		t.Skip("set RUN_LIVE_LLM_TEST=1 to run live multimodal LLM test")
	}

	config.LoadConfig()
	if strings.TrimSpace(config.GetEnv("LLM_API_KEY")) == "" {
		t.Fatal("LLM_API_KEY is not configured after loading runtime env")
	}

	imagePath := findFixturePath(t, "test/listen_image.jpg")
	imageBytes, err := os.ReadFile(imagePath)
	if err != nil {
		t.Fatalf("read image fixture: %v", err)
	}

	contentParts := []ContentPart{
		{
			Type: "image_url",
			ImageURL: &ImageURL{
				URL: "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(imageBytes),
			},
		},
		{
			Type: "text",
			Text: liveGradePrompt(),
		},
	}

	testCases := []struct {
		name   string
		client *OpenAICompatibleClient
	}{
		{
			name:   "default_transport",
			client: NewOpenAICompatibleClient(nil),
		},
		{
			name: "http1_only_transport",
			client: NewOpenAICompatibleClient(&http.Client{
				Timeout: resolveHTTPTimeout(),
				Transport: func() *http.Transport {
					transport := http.DefaultTransport.(*http.Transport).Clone()
					transport.ForceAttemptHTTP2 = false
					return transport
				}(),
			}),
		},
		{
			name: "http1_tls_next_proto_disabled",
			client: NewOpenAICompatibleClient(&http.Client{
				Timeout: resolveHTTPTimeout(),
				Transport: func() *http.Transport {
					transport := http.DefaultTransport.(*http.Transport).Clone()
					transport.ForceAttemptHTTP2 = false
					transport.TLSNextProto = make(map[string]func(string, *tls.Conn) http.RoundTripper)
					return transport
				}(),
			}),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()
			result, err := tc.client.Generate(context.Background(), GenerateRequest{
				ModelEnvKey: "LLM_GRADER_MODEL_NAME",
				Temperature: 0.1,
				Messages: []Message{
					{Role: "system", Content: "You are a professional teacher assistant grading dictation photos. Return valid JSON only."},
					{Role: "user", Content: contentParts},
				},
			})
			duration := time.Since(start)

			if err != nil {
				t.Fatalf("generate after %s: %v", duration, err)
			}

			t.Logf("generate succeeded after %s", duration)
			if strings.TrimSpace(result) == "" {
				t.Fatal("expected non-empty LLM response")
			}
		})
	}
}

func findFixturePath(t *testing.T, relativePath string) string {
	t.Helper()

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	currentDir := workingDir
	for {
		candidate := filepath.Join(currentDir, relativePath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break
		}
		currentDir = parentDir
	}

	t.Fatalf("could not find fixture %s from %s", relativePath, workingDir)
	return ""
}

func liveGradePrompt() string {
	return `
Compare the handwritten content in the image with the ordered answer list below.

Language: english
Mode: word
Ordered answers:
1. touch | 触碰
2. feel | 摸起来
3. soft | 柔软的
4. hard | 坚硬的
5. thick | 厚的
6. thin | 薄的
7. blind | 失明的
8. noise | 响声
9. young | 年轻的

Return JSON only:
{
  "status": "success",
  "score": 0,
  "graded_items": [
    {
      "index": 1,
      "expected": "touch",
      "meaning": "触碰",
      "actual": "touch",
      "is_correct": true,
      "comment": "",
      "needs_retry": false
    }
  ],
  "feedback": "继续保持。"
}

Rules:
- Compare in order.
- Read the handwritten English word when possible.
- Keep comment short.
- If unreadable or mismatched, set needs_retry true.
- Keep feedback concise.
`
}
