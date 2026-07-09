package main

import (
	"os"
	"path/filepath"
	"strings"
)

func loadSavedThemeName() (string, bool) {
	path, err := dataStorePath("theme")
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
	path, err := dataStorePath("theme")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(name+"\n"), 0o644)
}

func loadSavedBorderName() (string, bool) {
	path, err := dataStorePath("border")
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

func saveBorderName(name string) error {
	path, err := dataStorePath("border")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(name+"\n"), 0o644)
}

func dataStorePath(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "glab-pipelines", name), nil
}
