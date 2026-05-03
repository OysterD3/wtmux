package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Save validates c, marshals it to two-space-indented JSON with a trailing
// newline, and writes it atomically to target via a temp-file + rename.
// The parent directory is created if missing. Matches TS save.ts byte-for-
// byte for the same input.
func Save(target string, c *Config) error {
	if err := c.Validate(); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}
	raw = append(raw, '\n')

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("config: mkdir %s: %w", filepath.Dir(target), err)
	}

	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return fmt.Errorf("config: write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("config: rename %s -> %s: %w", tmp, target, err)
	}
	return nil
}
