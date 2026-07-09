package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func (p pipeline) UpdatedOrCreated() string {
	if p.UpdatedAt != "" {
		return p.UpdatedAt
	}
	return p.CreatedAt
}

func parseAPITime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err == nil {
		return t
	}
	t, _ = time.Parse(time.RFC3339, value)
	return t
}

func shortTime(value string) string {
	if value == "" {
		return "--"
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

func truncate(value string, limit int) string {
	r := []rune(value)
	if len(r) <= limit {
		return value
	}
	if limit <= 1 {
		return string(r[:limit])
	}
	return string(r[:limit-1]) + "~"
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

func repoLabel(repo string) string {
	if repo == "" {
		return ""
	}
	return "on " + repo + "  "
}

func openURL(url string) {
	if url == "" {
		return
	}
	cmd := exec.Command("open", url)
	if err := cmd.Start(); err == nil {
		return
	}
	_ = exec.Command("xdg-open", url).Start()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
