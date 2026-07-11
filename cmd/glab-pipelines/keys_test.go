package main

import (
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

func TestEmacsPageKeysScrollPane(t *testing.T) {
	m := model{mode: modePipelines, width: 80, height: 12}

	updated, _ := m.handlePipelineKey(tea.KeyMsg{Type: tea.KeyCtrlF})
	m = updated.(model)
	if m.scrollOffset == 0 {
		t.Fatal("ctrl+f did not scroll forward")
	}

	updated, _ = m.handlePipelineKey(tea.KeyMsg{Type: tea.KeyCtrlB})
	m = updated.(model)
	if m.scrollOffset != 0 {
		t.Fatalf("ctrl+b scroll offset = %d, want 0", m.scrollOffset)
	}
}

func TestEmacsPageKeysScrollLogs(t *testing.T) {
	v := viewport.New(40, 3)
	v.SetContent("one\ntwo\nthree\nfour\nfive\nsix")
	m := model{mode: modeLogs, width: 40, height: 8, logsViewport: v}

	updated, _ := m.handleLogsKey(tea.KeyMsg{Type: tea.KeyCtrlF})
	m = updated.(model)
	if m.logsViewport.YOffset == 0 {
		t.Fatal("ctrl+f did not page the log viewport forward")
	}

	updated, _ = m.handleLogsKey(tea.KeyMsg{Type: tea.KeyCtrlB})
	m = updated.(model)
	if m.logsViewport.YOffset != 0 {
		t.Fatalf("ctrl+b log offset = %d, want 0", m.logsViewport.YOffset)
	}
}
