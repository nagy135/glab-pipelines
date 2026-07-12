package main

import (
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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
