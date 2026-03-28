package agentmemory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoaderLoad(t *testing.T) {
	root := t.TempDir()

	if err := os.WriteFile(filepath.Join(root, "project.md"), []byte("global memory"), 0o644); err != nil {
		t.Fatalf("write project memory: %v", err)
	}

	rulesDir := filepath.Join(root, "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("mkdir rules: %v", err)
	}

	ruleA := "---\npaths:\n  - internal/logic/agent/**\n---\nagent rule"
	if err := os.WriteFile(filepath.Join(rulesDir, "agent.md"), []byte(ruleA), 0o644); err != nil {
		t.Fatalf("write agent rule: %v", err)
	}

	ruleB := "---\npaths:\n  - internal/llm/**\n---\nllm rule"
	if err := os.WriteFile(filepath.Join(rulesDir, "llm.md"), []byte(ruleB), 0o644); err != nil {
		t.Fatalf("write llm rule: %v", err)
	}

	loader := NewLoader(root)
	bundle, err := loader.Load([]string{"internal/logic/agent/controller.go"})
	if err != nil {
		t.Fatalf("load memory: %v", err)
	}

	if bundle.Project != "global memory" {
		t.Fatalf("unexpected project memory: %q", bundle.Project)
	}
	if len(bundle.Rules) != 1 {
		t.Fatalf("expected 1 matched rule, got %d", len(bundle.Rules))
	}
	if bundle.Rules[0].Name != "agent" {
		t.Fatalf("unexpected rule name: %s", bundle.Rules[0].Name)
	}
}

func TestMatchPattern(t *testing.T) {
	cases := []struct {
		pattern string
		value   string
		ok      bool
	}{
		{pattern: "internal/logic/agent/**", value: "internal/logic/agent/controller.go", ok: true},
		{pattern: "internal/*/agent/**", value: "internal/logic/agent/controller.go", ok: true},
		{pattern: "internal/llm/**", value: "internal/logic/agent/controller.go", ok: false},
	}

	for _, tc := range cases {
		if got := matchPattern(tc.pattern, tc.value); got != tc.ok {
			t.Fatalf("matchPattern(%q, %q) = %v, want %v", tc.pattern, tc.value, got, tc.ok)
		}
	}
}
