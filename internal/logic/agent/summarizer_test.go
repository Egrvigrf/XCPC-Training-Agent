package agent

import "testing"

func TestDefaultToolSummarizer_Object(t *testing.T) {
	s := NewDefaultToolSummarizer()
	result := map[string]any{
		"student_id": "240511307",
		"from":       "2025-01-01",
		"to":         "2025-12-31",
		"cf_total":   376,
		"items": []any{
			map[string]any{"rank": 1, "student_id": "A"},
			map[string]any{"rank": 2, "student_id": "B"},
		},
	}

	summary := s.Summarize("training_summary_range", result)

	if summary["tool"] != "training_summary_range" {
		t.Fatalf("unexpected tool: %v", summary["tool"])
	}

	identity, ok := summary["identity"].(map[string]any)
	if !ok || identity["student_id"] != "240511307" {
		t.Fatalf("missing identity summary: %#v", summary["identity"])
	}

	arrays, ok := summary["arrays"].(map[string]any)
	if !ok {
		t.Fatalf("missing arrays summary")
	}

	items, ok := arrays["items"].(map[string]any)
	if !ok || items["count"] != 2 {
		t.Fatalf("unexpected items summary: %#v", arrays["items"])
	}
}

func TestDefaultToolSummarizer_Array(t *testing.T) {
	s := NewDefaultToolSummarizer()
	result := []any{
		map[string]any{"student_id": "A", "rank": 12},
		map[string]any{"student_id": "B", "rank": 21},
		map[string]any{"student_id": "C", "rank": 35},
	}

	summary := s.Summarize("contest_ranking", result)
	arraySummary, ok := summary["array"].(map[string]any)
	if !ok {
		t.Fatalf("missing array summary")
	}
	if arraySummary["count"] != 3 {
		t.Fatalf("unexpected count: %#v", arraySummary)
	}
}
