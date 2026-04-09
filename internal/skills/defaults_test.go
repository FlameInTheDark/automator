package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureBundledDefaultsWritesMissingSkills(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	skillsDir := filepath.Join(dir, ".agents", "skills")

	if err := EnsureBundledDefaults(skillsDir); err != nil {
		t.Fatalf("EnsureBundledDefaults returned error: %v", err)
	}

	luaPath := filepath.Join(skillsDir, "lua-scripting-guide", "SKILL.md")
	content, err := os.ReadFile(luaPath)
	if err != nil {
		t.Fatalf("read seeded lua skill: %v", err)
	}
	if !strings.Contains(string(content), "The current node input is exposed as a global named `input`.") &&
		!strings.Contains(string(content), "The current node input is exposed as a global named input.") {
		t.Fatalf("seeded lua skill missing input guidance:\n%s", string(content))
	}
}

func TestEnsureBundledDefaultsDoesNotOverwriteExistingSkills(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	skillsDir := filepath.Join(dir, ".agents", "skills")
	customPath := filepath.Join(skillsDir, "templating-guide", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(customPath), 0o755); err != nil {
		t.Fatalf("mkdir custom skill dir: %v", err)
	}
	if err := os.WriteFile(customPath, []byte("custom content"), 0o644); err != nil {
		t.Fatalf("write custom skill: %v", err)
	}

	if err := EnsureBundledDefaults(skillsDir); err != nil {
		t.Fatalf("EnsureBundledDefaults returned error: %v", err)
	}

	content, err := os.ReadFile(customPath)
	if err != nil {
		t.Fatalf("read custom skill: %v", err)
	}
	if string(content) != "custom content" {
		t.Fatalf("expected existing skill to be preserved, got %q", string(content))
	}
}
