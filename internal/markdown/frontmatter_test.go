package markdown

import (
	"strings"
	"testing"
)

func TestParse_WithFrontmatter(t *testing.T) {
	raw := "---\nid: abc-123\n---\n# Hello\n\nSome content."
	fm, body := Parse(raw)

	if fm.ID != "abc-123" {
		t.Errorf("expected ID 'abc-123', got %q", fm.ID)
	}
	if !strings.Contains(body, "# Hello") {
		t.Errorf("expected body to contain '# Hello', got %q", body)
	}
	if !strings.Contains(body, "Some content.") {
		t.Errorf("expected body to contain 'Some content.', got %q", body)
	}
}

func TestParse_NoFrontmatter(t *testing.T) {
	raw := "# Just a heading\n\nNo frontmatter here."
	fm, body := Parse(raw)

	if fm.ID != "" {
		t.Errorf("expected empty ID, got %q", fm.ID)
	}
	if body != raw {
		t.Errorf("expected body to be the full raw text, got %q", body)
	}
}

func TestParse_EmptyID(t *testing.T) {
	raw := "---\nid: \n---\n# Page"
	fm, _ := Parse(raw)

	if fm.ID != "" {
		t.Errorf("expected empty ID, got %q", fm.ID)
	}
}

func TestEnsureID(t *testing.T) {
	fm := Frontmatter{ID: ""}
	fm = EnsureID(fm)

	if fm.ID == "" {
		t.Error("expected a generated ID, got empty string")
	}

	// Should not overwrite an existing ID
	fm2 := Frontmatter{ID: "keep-me"}
	fm2 = EnsureID(fm2)
	if fm2.ID != "keep-me" {
		t.Errorf("expected 'keep-me', got %q", fm2.ID)
	}
}

func TestSerialize(t *testing.T) {
	fm := Frontmatter{ID: "test-id"}
	body := "# My Page\n\nContent here.\n"

	result := Serialize(fm, body)

	if !strings.HasPrefix(result, "---\n") {
		t.Error("expected result to start with ---")
	}
	if !strings.Contains(result, "id: test-id") {
		t.Error("expected result to contain 'id: test-id'")
	}
	if !strings.Contains(result, "# My Page") {
		t.Error("expected result to contain body")
	}
}

func TestSerialize_Roundtrip(t *testing.T) {
	original := "---\nid: roundtrip-id\n---\n# Title\n\nBody text.\n"
	fm, body := Parse(original)
	result := Serialize(fm, body)

	fm2, body2 := Parse(result)
	if fm2.ID != "roundtrip-id" {
		t.Errorf("expected ID 'roundtrip-id' after roundtrip, got %q", fm2.ID)
	}
	if !strings.Contains(body2, "# Title") {
		t.Errorf("expected body to contain '# Title' after roundtrip, got %q", body2)
	}
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		title string
	}{
		{"h1 heading", "# My Title\n\nSome content.", "My Title"},
		{"no heading", "Just plain text.", ""},
		{"h2 not matched", "## Subtitle\n\nContent.", ""},
		{"heading after text", "Intro text.\n\n# Actual Title\n\nMore.", "Actual Title"},
		{"heading with spaces", "#   Spaced Title  \n", "Spaced Title"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTitle(tt.body)
			if got != tt.title {
				t.Errorf("expected %q, got %q", tt.title, got)
			}
		})
	}
}
