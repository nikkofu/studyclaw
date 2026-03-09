package services

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestParseParentInput(t *testing.T) {
	originalClient := agentCoreHTTPClient
	t.Cleanup(func() {
		agentCoreHTTPClient = originalClient
	})

	agentCoreHTTPClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != "http://agent-core.test/api/v1/internal/parse" {
				t.Fatalf("unexpected url: %s", r.URL.String())
			}

			if r.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", r.Method)
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("ReadAll returned error: %v", err)
			}

			if !strings.Contains(string(body), `"raw_text":"今晚数学口算30题，英语听写第一单元"`) {
				t.Fatalf("request body missing raw_text: %s", string(body))
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(`{
					"status":"success",
					"parser_mode":"llm_hybrid",
					"analysis":{"task_count":1,"needs_review_count":0},
					"data":[{"subject":"数学","group_title":"口算30题","title":"口算30题","type":"homework","confidence":0.91,"needs_review":false,"notes":[]}]
				}`)),
			}, nil
		}),
	}

	t.Setenv("AGENT_CORE_URL", "http://agent-core.test")

	resp, err := ParseParentInput("今晚数学口算30题，英语听写第一单元")
	if err != nil {
		t.Fatalf("ParseParentInput returned error: %v", err)
	}

	if resp.Status != "success" {
		t.Fatalf("expected status success, got %s", resp.Status)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 parsed task, got %d", len(resp.Data))
	}

	if resp.Data[0].Title != "口算30题" {
		t.Fatalf("expected title 口算30题, got %s", resp.Data[0].Title)
	}

	if resp.Data[0].GroupTitle != "口算30题" {
		t.Fatalf("expected group title 口算30题, got %s", resp.Data[0].GroupTitle)
	}

	if resp.ParserMode != "llm_hybrid" {
		t.Fatalf("expected parser mode llm_hybrid, got %s", resp.ParserMode)
	}

	if resp.Data[0].Confidence <= 0 {
		t.Fatalf("expected confidence to be populated, got %f", resp.Data[0].Confidence)
	}
}
