package taskparse

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type regressionExpectedTask struct {
	Subject       string   `json:"subject"`
	GroupTitle    string   `json:"group_title"`
	Title         string   `json:"title"`
	NeedsReview   bool     `json:"needs_review"`
	RequiredNotes []string `json:"required_notes"`
}

type regressionCase struct {
	ID                       string                   `json:"id"`
	Category                 string                   `json:"category"`
	Why                      string                   `json:"why"`
	RawText                  string                   `json:"raw_text"`
	ExpectedTaskCount        int                      `json:"expected_task_count"`
	ExpectedNeedsReviewCount int                      `json:"expected_needs_review_count"`
	ExpectedSignals          []string                 `json:"expected_signals"`
	ExpectedTasks            []regressionExpectedTask `json:"expected_tasks"`
}

func loadRegressionCases(t *testing.T) []regressionCase {
	t.Helper()

	path := filepath.Join("testdata", "regression_cases.json")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read regression cases %s: %v", path, err)
	}

	var cases []regressionCase
	if err := json.Unmarshal(content, &cases); err != nil {
		t.Fatalf("decode regression cases %s: %v", path, err)
	}
	if len(cases) == 0 {
		t.Fatalf("expected regression cases in %s", path)
	}

	return cases
}

func TestParseFallbackRegressionFixtures(t *testing.T) {
	for _, tc := range loadRegressionCases(t) {
		t.Run(tc.ID, func(t *testing.T) {
			result := parseFallback(tc.RawText)
			if result.Status != "success" {
				t.Fatalf("%s: expected success, got %+v", tc.Why, result)
			}
			if len(result.Data) != tc.ExpectedTaskCount {
				t.Fatalf("%s: expected %d tasks, got %d", tc.Why, tc.ExpectedTaskCount, len(result.Data))
			}
			if result.Analysis.NeedsReviewCount != tc.ExpectedNeedsReviewCount {
				t.Fatalf("%s: expected needs_review_count=%d, got %d", tc.Why, tc.ExpectedNeedsReviewCount, result.Analysis.NeedsReviewCount)
			}
			for _, signal := range tc.ExpectedSignals {
				if !containsString(result.Analysis.FormatSignals, signal) {
					t.Fatalf("%s: expected format signal %q in %+v", tc.Why, signal, result.Analysis.FormatSignals)
				}
			}

			for _, expectedTask := range tc.ExpectedTasks {
				task := findTaskByTitle(t, result.Data, expectedTask.Title)
				if task.Subject != expectedTask.Subject || task.GroupTitle != expectedTask.GroupTitle {
					t.Fatalf("%s: unexpected task identity for %q: %+v", tc.Why, expectedTask.Title, task)
				}
				if task.NeedsReview != expectedTask.NeedsReview {
					t.Fatalf("%s: expected needs_review=%v for %q, got %+v", tc.Why, expectedTask.NeedsReview, expectedTask.Title, task)
				}
				if !expectedTask.NeedsReview && len(task.Notes) != 0 {
					t.Fatalf("%s: expected no notes for safe task %q, got %+v", tc.Why, expectedTask.Title, task.Notes)
				}
				for _, note := range expectedTask.RequiredNotes {
					if !containsString(task.Notes, note) {
						t.Fatalf("%s: expected note %q for %q, got %+v", tc.Why, note, expectedTask.Title, task.Notes)
					}
				}
			}
		})
	}
}
