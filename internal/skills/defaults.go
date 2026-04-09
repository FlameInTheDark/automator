package skills

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed defaults
var bundledDefaults embed.FS

func EnsureBundledDefaults(skillDir string) error {
	trimmed := strings.TrimSpace(skillDir)
	if trimmed == "" {
		return fmt.Errorf("skills directory is required")
	}

	if err := os.MkdirAll(trimmed, 0o755); err != nil {
		return fmt.Errorf("ensure skills directory: %w", err)
	}

	return fs.WalkDir(bundledDefaults, "defaults/skills", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == "defaults/skills" {
			return nil
		}

		relative := strings.TrimPrefix(path, "defaults/skills/")
		destination := filepath.Join(trimmed, filepath.FromSlash(relative))

		if d.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}

		if _, err := os.Stat(destination); err == nil {
			return nil
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("stat bundled skill destination %s: %w", destination, err)
		}

		content, err := bundledDefaults.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read bundled skill %s: %w", path, err)
		}

		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return fmt.Errorf("ensure bundled skill parent %s: %w", destination, err)
		}
		if err := os.WriteFile(destination, content, 0o644); err != nil {
			return fmt.Errorf("write bundled skill %s: %w", destination, err)
		}

		return nil
	})
}
