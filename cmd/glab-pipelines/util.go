package main

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/x/ansi"
)

func (p pipeline) UpdatedOrCreated() string {
	if p.UpdatedAt != "" {
		return p.UpdatedAt
	}
	return p.CreatedAt
}

func shortTime(value string) string {
	if value == "" {
		return "--"
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed.In(time.Local).Format("2006-01-02 15:04:05")
	}
	value = strings.TrimSuffix(value, "Z")
	if i := strings.Index(value, "."); i >= 0 {
		value = value[:i]
	}
	return strings.Replace(value, "T", " ", 1)
}

func shortSHA(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	if sha == "" {
		return "--"
	}
	return sha
}

func formatDuration(d *float64) string {
	if d == nil {
		return "--"
	}
	s := int(*d)
	h := s / 3600
	m := (s % 3600) / 60
	sec := s % 60
	if h > 0 {
		return fmt.Sprintf("%dh%02dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%02ds", m, sec)
	}
	return fmt.Sprintf("%ds", sec)
}

func formatPipelineDuration(d *float64) string {
	if d == nil {
		return "--"
	}
	s := int(*d)
	h := s / 3600
	m := (s % 3600) / 60
	sec := s % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, sec)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, sec)
	}
	return fmt.Sprintf("%ds", sec)
}

func truncate(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if ansi.StringWidth(value) <= limit {
		return value
	}
	if limit == 1 {
		return ansi.Truncate(value, limit, "")
	}
	return ansi.Truncate(value, limit, "~")
}

func moveUp(cursor, count int) int {
	if count == 0 {
		return 0
	}
	if cursor <= 0 {
		return count - 1
	}
	return cursor - 1
}

func moveDown(cursor, count int) int {
	if count == 0 {
		return 0
	}
	if cursor >= count-1 {
		return 0
	}
	return cursor + 1
}

func openURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return fmt.Errorf("refusing to open invalid web URL")
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "linux":
		cmd = exec.Command("xdg-open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		return fmt.Errorf("opening URLs is not supported on %s", runtime.GOOS)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open URL: %w", err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

func sanitizeTerminalText(value string) string {
	value = ansi.Strip(value)
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, value)
}
