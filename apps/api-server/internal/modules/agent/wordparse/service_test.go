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

func TestDecodeGradeResultNormalizesEnglishFeedbackAndKeepsMarkRegions(t *testing.T) {
	raw := `{
		"status": "success",
		"score": 90,
		"annotated_photo_url": "https://example.com/graded.png",
		"annotated_photo_width": 1200,
		"annotated_photo_height": 900,
		"mark_regions": [
			{
				"index": 2,
				"expected": "library",
				"actual": "libary",
				"is_correct": false,
				"left": 0.12,
				"top": 0.3,
				"width": 0.2,
				"height": 0.08,
				"marker_label": "❌"
			}
		],
		"graded_items": [
			{
				"index": 2,
				"expected": "library",
				"meaning": "图书馆",
				"actual": "libary",
				"is_correct": false,
				"comment": "少了 r",
				"needs_retry": true
			}
		],
		"feedback": "All English words are correct!"
	}`

	result, err := decodeGradeResult(raw)
	if err != nil {
		t.Fatalf("decode grade result with mark regions: %v", err)
	}

	if result.Feedback == "All English words are correct!" {
		t.Fatalf("expected english feedback to be normalized, got %q", result.Feedback)
	}
	if result.AnnotatedPhotoURL != "https://example.com/graded.png" {
		t.Fatalf("expected annotated photo url to survive decode, got %q", result.AnnotatedPhotoURL)
	}
	if len(result.MarkRegions) != 1 {
		t.Fatalf("expected 1 mark region, got %d", len(result.MarkRegions))
	}
	if result.MarkRegions[0].MarkerLabel != "❌" {
		t.Fatalf("expected marker label ❌, got %+v", result.MarkRegions[0])
	}
}

func TestDecodeGradeResultGeneratesChineseFallbackWhenFeedbackMissing(t *testing.T) {
	raw := `{
		"status": "success",
		"score": 100,
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
		"feedback": ""
	}`

	result, err := decodeGradeResult(raw)
	if err != nil {
		t.Fatalf("decode grade result with empty feedback: %v", err)
	}
	if result.Feedback == "" {
		t.Fatal("expected fallback chinese feedback to be generated")
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
