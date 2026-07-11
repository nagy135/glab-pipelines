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
