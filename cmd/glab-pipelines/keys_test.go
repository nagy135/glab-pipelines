package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestPageKeysScrollPane(t *testing.T) {
	m := model{mode: modePipelines, width: 80, height: 12}

	updated, _ := m.handlePipelineKey(tea.KeyMsg{Type: tea.KeyPgDown})
	m = updated.(model)
	if m.scrollOffset == 0 {
		t.Fatal("pgdown did not scroll forward")
	}

	updated, _ = m.handlePipelineKey(tea.KeyMsg{Type: tea.KeyPgUp})
	m = updated.(model)
	if m.scrollOffset != 0 {
		t.Fatalf("pgup scroll offset = %d, want 0", m.scrollOffset)
	}
}

func TestPageKeysScrollLogs(t *testing.T) {
	v := viewport.New(40, 3)
	v.SetContent("one\ntwo\nthree\nfour\nfive\nsix")
	m := model{mode: modeLogs, width: 40, height: 8, logsViewport: v}

	updated, _ := m.handleLogsKey(tea.KeyMsg{Type: tea.KeyPgDown})
	m = updated.(model)
	if m.logsViewport.YOffset == 0 {
		t.Fatal("pgdown did not page the log viewport forward")
	}

	updated, _ = m.handleLogsKey(tea.KeyMsg{Type: tea.KeyPgUp})
	m = updated.(model)
	if m.logsViewport.YOffset != 0 {
		t.Fatalf("pgup log offset = %d, want 0", m.logsViewport.YOffset)
	}
}

func TestVimDirectionKeysFocusSplitPanes(t *testing.T) {
	m := model{
		mode:          modePipelines,
		width:         80,
		height:        24,
		activeLogPane: 1,
		logPanes:      []logPane{{ID: 1, Mode: modePipelines}, {ID: 2, Mode: modePipelines}},
		logSplitRoot: &logSplitNode{
			Direction: logSplitVertical,
			First:     &logSplitNode{PaneID: 1},
			Second:    &logSplitNode{PaneID: 2},
		},
	}

	updated, _ := m.handlePipelineKey(tea.KeyMsg{Type: tea.KeyCtrlL})
	m = updated.(model)
	if m.activeLogPane != 2 {
		t.Fatalf("ctrl+l focused pane %d, want right pane 2", m.activeLogPane)
	}
	updated, _ = m.handlePipelineKey(tea.KeyMsg{Type: tea.KeyCtrlH})
	m = updated.(model)
	if m.activeLogPane != 1 {
		t.Fatalf("ctrl+h focused pane %d, want left pane 1", m.activeLogPane)
	}
}

func TestVimDirectionKeysFocusVerticalSplitPanes(t *testing.T) {
	m := model{
		mode:          modePipelines,
		width:         80,
		height:        24,
		activeLogPane: 1,
		logPanes:      []logPane{{ID: 1, Mode: modePipelines}, {ID: 2, Mode: modePipelines}},
		logSplitRoot: &logSplitNode{
			Direction: logSplitHorizontal,
			First:     &logSplitNode{PaneID: 1},
			Second:    &logSplitNode{PaneID: 2},
		},
	}

	updated, _ := m.handlePipelineKey(tea.KeyMsg{Type: tea.KeyCtrlJ})
	m = updated.(model)
	if m.activeLogPane != 2 {
		t.Fatalf("ctrl+j focused pane %d, want lower pane 2", m.activeLogPane)
	}
	updated, _ = m.handlePipelineKey(tea.KeyMsg{Type: tea.KeyCtrlK})
	m = updated.(model)
	if m.activeLogPane != 1 {
		t.Fatalf("ctrl+k focused pane %d, want upper pane 1", m.activeLogPane)
	}
}

func TestEmacsDirectionKeysScrollPane(t *testing.T) {
	m := model{mode: modePipelines, width: 80, height: 12}

	updated, _ := m.handlePipelineKey(tea.KeyMsg{Type: tea.KeyCtrlN})
	m = updated.(model)
	if m.scrollOffset != 1 {
		t.Fatalf("ctrl+n scroll offset = %d, want 1", m.scrollOffset)
	}
	updated, _ = m.handlePipelineKey(tea.KeyMsg{Type: tea.KeyCtrlP})
	m = updated.(model)
	if m.scrollOffset != 0 {
		t.Fatalf("ctrl+p scroll offset = %d, want 0", m.scrollOffset)
	}
	updated, _ = m.handlePipelineKey(tea.KeyMsg{Type: tea.KeyCtrlF})
	m = updated.(model)
	if m.horizontalOffset == 0 {
		t.Fatal("ctrl+f did not scroll right")
	}
	updated, _ = m.handlePipelineKey(tea.KeyMsg{Type: tea.KeyCtrlB})
	m = updated.(model)
	if m.horizontalOffset != 0 {
		t.Fatalf("ctrl+b horizontal offset = %d, want 0", m.horizontalOffset)
	}
}

func TestDetailArrowKeysMoveJobsWithoutScrolling(t *testing.T) {
	m := model{
		mode: modeDetail,
		detail: &detail{DisplayJobs: []uiJob{
			{Current: job{ID: 1, Status: "success"}},
			{Current: job{ID: 2, Status: "running"}},
		}},
	}

	updated, _ := m.handleDetailKey(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(model)
	if m.jobsCursor != 1 || m.scrollOffset != 0 {
		t.Fatalf("down resulted in jobs cursor %d and scroll offset %d", m.jobsCursor, m.scrollOffset)
	}
}

func TestWrapKeyTogglesFocusedPaneAndDisablesHorizontalScroll(t *testing.T) {
	v := viewport.New(12, 5)
	v.SetContent("a very long log line")
	m := model{
		mode:             modeLogs,
		width:            14,
		height:           10,
		logs:             "a very long log line",
		logsViewport:     v,
		activeLogPane:    1,
		horizontalOffset: 16,
		logPanes:         []logPane{{ID: 1, Mode: modeLogs, Viewport: v}},
	}

	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	m = updated.(model)
	if !m.wrapContent || m.horizontalOffset != 0 || !m.logPanes[0].WrapContent {
		t.Fatalf("wrap state was not saved: wrap=%v offset=%d pane=%+v", m.wrapContent, m.horizontalOffset, m.logPanes[0])
	}
	for _, line := range strings.Split(ansi.Strip(m.logsViewport.View()), "\n") {
		if ansi.StringWidth(line) > m.logsViewport.Width {
			t.Fatalf("wrapped line width = %d, want <= %d: %q", ansi.StringWidth(line), m.logsViewport.Width, line)
		}
	}

	updated, _ = m.handleLogsKey(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(model)
	if m.horizontalOffset != 0 {
		t.Fatalf("wrapped pane scrolled horizontally to %d", m.horizontalOffset)
	}
}

func TestNumberKeyTogglesLineNumbersInLogsAndCode(t *testing.T) {
	for _, mode := range []int{modeLogs, modeCode} {
		v := viewport.New(20, 5)
		m := model{mode: mode, logs: "first\nsecond", logsViewport: v}
		key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'#'}}
		var updated tea.Model
		if mode == modeLogs {
			updated, _ = m.handleLogsKey(key)
		} else {
			updated, _ = m.handleCodeKey(key)
		}
		m = updated.(model)
		plain := ansi.Strip(m.logsViewport.View())
		if !m.showLineNumbers || !strings.Contains(plain, "1 | first") || !strings.Contains(plain, "2 | second") {
			t.Fatalf("mode %d did not show line numbers: %q", mode, plain)
		}
	}
}
