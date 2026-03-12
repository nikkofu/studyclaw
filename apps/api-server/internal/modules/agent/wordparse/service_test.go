package wordparse

import "testing"

func TestDecodeGradeResultDirect(t *testing.T) {
	raw := `{
		"status": "success",
		"score": 78,
		"graded_items": [
			{
				"index": 1,
				"expected": "touch",
				"meaning": "触碰",
				"actual": "touch",
				"is_correct": true,
				"comment": "正确",
				"needs_retry": false
			}
		],
		"feedback": "整体不错。"
	}`

	result, err := decodeGradeResult(raw)
	if err != nil {
		t.Fatalf("decode direct grade result: %v", err)
	}

	if result.Score != 78 {
		t.Fatalf("expected score 78, got %d", result.Score)
	}
	if result.Status != "success" {
		t.Fatalf("expected status success, got %s", result.Status)
	}
	if len(result.GradedItems) != 1 {
		t.Fatalf("expected 1 graded item, got %d", len(result.GradedItems))
	}
}

func TestDecodeGradeResultWrapped(t *testing.T) {
	raw := `{
		"data": {
			"status": "success",
			"score": 92,
			"graded_items": [
				{
					"index": 1,
					"expected": "soft",
					"meaning": "柔软的",
					"actual": "soft",
					"is_correct": true,
					"comment": "正确",
					"needs_retry": false
				}
			],
			"feedback": "写得很棒。"
		}
	}`

	result, err := decodeGradeResult(raw)
	if err != nil {
		t.Fatalf("decode wrapped grade result: %v", err)
	}

	if result.Score != 92 {
		t.Fatalf("expected score 92, got %d", result.Score)
	}
	if result.Feedback != "写得很棒。" {
		t.Fatalf("expected wrapped feedback to survive decode, got %q", result.Feedback)
	}
}

func TestDecodeGradeResultRejectsIncompleteWrapper(t *testing.T) {
	raw := `{"data":{"status":"success"}}`

	if _, err := decodeGradeResult(raw); err == nil {
		t.Fatal("expected incomplete wrapped result to fail")
	}
}

func TestDecodeGradeResultRejectsUnknownObject(t *testing.T) {
	raw := `{"foo":"bar"}`

	if _, err := decodeGradeResult(raw); err == nil {
		t.Fatal("expected unknown grading payload to fail")
	}
}
