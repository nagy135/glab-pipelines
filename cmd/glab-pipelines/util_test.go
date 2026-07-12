package main

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

func TestSanitizeTerminalText(t *testing.T) {
	input := "safe\x1b]52;c;clipboard\a\x1b[31mred\x1b[0m\rforged"
	got := sanitizeTerminalText(input)
	if got != "saferedforged" || strings.ContainsRune(got, '\x1b') {
		t.Fatalf("sanitizeTerminalText() = %q", got)
	}
}

func TestTruncateUsesTerminalCellWidth(t *testing.T) {
	got := truncate("界abc", 4)
	if ansi.StringWidth(got) > 4 || !strings.HasSuffix(got, "~") {
		t.Fatalf("truncate() = %q (width %d)", got, ansi.StringWidth(got))
	}
}

func TestShortTimeConvertsGitLabTimestampToLocalTime(t *testing.T) {
	originalLocal := time.Local
	time.Local = time.FixedZone("CEST", 2*60*60)
	t.Cleanup(func() { time.Local = originalLocal })

	got := shortTime("2026-07-11T12:02:52.123Z")
	if got != "2026-07-11 14:02:52" {
		t.Fatalf("shortTime() = %q, want local time", got)
	}
}

func TestTimeAgo(t *testing.T) {
	now := time.Date(2026, 7, 11, 14, 30, 0, 0, time.UTC)
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "not started", want: "--"},
		{name: "seconds", value: "2026-07-11T14:29:18Z", want: "42s ago"},
		{name: "minutes", value: "2026-07-11T14:18:00Z", want: "12m ago"},
		{name: "hours", value: "2026-07-11T12:15:00Z", want: "2h 15m ago"},
		{name: "days", value: "2026-07-09T11:30:00Z", want: "2d 3h ago"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := timeAgo(tt.value, now); got != tt.want {
				t.Fatalf("timeAgo() = %q, want %q", got, tt.want)
			}
		})
	}
}
