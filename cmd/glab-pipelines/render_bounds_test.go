package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestSinglePaneFitsAtScrollBoundary(t *testing.T) {
	for _, size := range []struct{ width, height int }{{100, 24}, {180, 56}, {40, 10}} {
		for delta := -1; delta <= 1; delta++ {
			t.Run(fmt.Sprintf("%dx%d/rows%+d", size.width, size.height, delta), func(t *testing.T) {
				m := model{width: size.width, height: size.height, mode: modeDetail}
				// Two header rows and two border rows leave height-4 body rows.
				body := strings.Repeat("job row\n", size.height-4+delta)
				view := m.renderSinglePane(body)
				if got := strings.Count(view, "\n") + 1; got != size.height-2 {
					t.Fatalf("pane has %d rows, want %d", got, size.height-2)
				}
			})
		}
	}
}

func TestViewFitsTerminal(t *testing.T) {
	for _, size := range []struct{ width, height int }{{100, 24}, {180, 56}, {40, 10}, {1, 1}} {
		for _, mode := range []int{modePipelines, modeDetail, modeJobs, modeConfirm, modeLogs, modeCode, modeTheme, modeRefresh, modeLimit} {
			t.Run(fmt.Sprintf("%dx%d/mode%d", size.width, size.height, mode), func(t *testing.T) {
				m := model{width: size.width, height: size.height, mode: mode}
				m.pending = &pendingAction{Verb: "Play", Job: job{Name: "deploy", Status: "manual"}}
				view := m.View()
				lines := strings.Split(view, "\n")
				if len(lines) != size.height {
					t.Errorf("frame has %d rows, want %d", len(lines), size.height)
				}
				for i, line := range lines {
					if got := ansi.StringWidth(line); got != size.width {
						t.Errorf("row %d has %d columns, want %d", i, got, size.width)
						break
					}
				}
			})
		}
	}
}

func TestFillScreenClipsFromBottomAndPreservesANSI(t *testing.T) {
	m := model{width: 6, height: 3}
	view := m.fillScreen("\x1b[31mheader too wide\x1b[m\n界界界界\nlast\ndiscard")
	if got, want := ansi.Strip(view), "header\n界界界\nlast  "; got != want {
		t.Fatalf("frame = %q, want %q", got, want)
	}
	if !strings.Contains(view, "\x1b[31m") {
		t.Fatal("clipping removed the header's color")
	}
}
