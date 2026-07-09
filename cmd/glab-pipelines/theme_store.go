package main

import (
	"os"
	"path/filepath"
	"strings"
)

func loadSavedThemeName() (string, bool) {
	path, err := themeStorePath()
	if err != nil {
		return "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	name := strings.TrimSpace(string(data))
	return name, name != ""
}

func saveThemeName(name string) error {
	path, err := themeStorePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(name+"\n"), 0o644)
}

func themeStorePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "glab-pipelines", "theme"), nil
}
