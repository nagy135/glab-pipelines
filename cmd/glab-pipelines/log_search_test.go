package main

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/viewport"
)

func TestFindLogSearchMatchesCaseInsensitive(t *testing.T) {
	logs := "starting\nERROR failed\ndone\nretry error"
	matches := findLogSearchMatches(logs, "error")
	want := []logSearchMatch{{Line: 1, Start: 0, End: 5}, {Line: 3, Start: 6, End: 11}}
	if len(matches) != len(want) {
		t.Fatalf("len(matches) = %d, want %d", len(matches), len(want))
	}
	for i := range want {
		if matches[i] != want[i] {
			t.Fatalf("matches[%d] = %+v, want %+v", i, matches[i], want[i])
		}
	}
}

func TestLogSearchMatchNearWraps(t *testing.T) {
	matches := []logSearchMatch{{Line: 2}, {Line: 5}, {Line: 9}}
	if got := logSearchMatchNear(matches, 6, 1); got != 2 {
		t.Fatalf("forward match index = %d, want 2", got)
	}
	if got := logSearchMatchNear(matches, 10, 1); got != 0 {
		t.Fatalf("forward wrapped match index = %d, want 0", got)
	}
	if got := logSearchMatchNear(matches, 4, -1); got != 0 {
		t.Fatalf("backward match index = %d, want 0", got)
	}
	if got := logSearchMatchNear(matches, 1, -1); got != 2 {
		t.Fatalf("backward wrapped match index = %d, want 2", got)
	}
}

func TestFindLogSearchMatchesPreservesUnicodeByteOffsets(t *testing.T) {
	logs := "Échec\nİstanbul"
	matches := findLogSearchMatches(logs, "é")
	if len(matches) != 1 || matches[0] != (logSearchMatch{Line: 0, Start: 0, End: 2}) {
		t.Fatalf("unicode matches = %+v", matches)
	}

	rendered := renderLogContentFor(logs, matches, 0)
	if !strings.Contains(rendered, "É") || !utf8.ValidString(rendered) {
		t.Fatalf("rendered match is invalid: %q", rendered)
	}
}

func TestWrappedSearchMatchUsesVisualLineOffset(t *testing.T) {
	v := viewport.New(6, 1)
	m := model{
		mode:             modeLogs,
		logs:             "1234567890\ntarget",
		logsViewport:     v,
		wrapContent:      true,
		logSearchQuery:   "target",
		logSearchMatches: findLogSearchMatches("1234567890\ntarget", "target"),
		logSearchIndex:   -1,
	}
	m.logsViewport.SetContent(m.renderLogContent())
	m = m.jumpLogSearchMatch(1)

	if m.logsViewport.YOffset != 2 {
		t.Fatalf("wrapped search offset = %d, want visual line 2", m.logsViewport.YOffset)
	}
}
