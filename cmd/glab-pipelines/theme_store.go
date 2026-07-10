package main

import (
	"os"
	"path/filepath"
	"strings"
)

func loadSavedThemeName() (string, bool) {
	return loadSavedName("theme")
}

func saveThemeName(name string) error {
	return saveName("theme", name)
}

func loadSavedBorderName() (string, bool) {
	return loadSavedName("border")
}

func saveBorderName(name string) error {
	return saveName("border", name)
}

func loadSavedName(key string) (string, bool) {
	path, err := dataStorePath(key)
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

func saveName(key, name string) error {
	path, err := dataStorePath(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(name+"\n"), 0o600)
}

func dataStorePath(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "glab-pipelines", name), nil
}
