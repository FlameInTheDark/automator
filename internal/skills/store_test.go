package skills

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreLoadsSkillsFromDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	skillDir := filepath.Join(dir, "frontend-design")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}

	content := `---
name: frontend-design
description: Create polished frontend experiences.
---

# Frontend Design
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	store := NewStore(dir, 25*time.Millisecond)
	if err := store.Start(context.Background()); err != nil {
		t.Fatalf("start store: %v", err)
	}
	defer store.Stop()

	summaries := store.List()
	if len(summaries) != 1 {
		t.Fatalf("expected 1 skill summary, got %d", len(summaries))
	}
	if summaries[0].Name != "frontend-design" {
		t.Fatalf("unexpected skill name %q", summaries[0].Name)
	}
	if summaries[0].Description != "Create polished frontend experiences." {
		t.Fatalf("unexpected description %q", summaries[0].Description)
	}

	skill, ok := store.GetByName("frontend-design")
	if !ok {
		t.Fatal("expected to get skill by name")
	}
	if skill.Content == "" {
		t.Fatal("expected skill content to be loaded")
	}
}

func TestParseSkillFileFallsBackToFolderName(t *testing.T) {
	t.Parallel()

	path := filepath.Join("workspace", "skills", "demo", "SKILL.md")
	skill := parseSkillFile(path, []byte("# No front matter"))
	if skill.Name != "demo" {
		t.Fatalf("expected fallback name demo, got %q", skill.Name)
	}
}
