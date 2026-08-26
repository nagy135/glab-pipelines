package main

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestUpperROpensRefreshPicker(t *testing.T) {
	m := model{mode: modeDetail, refresh: 20 * time.Second}

	updated, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	m = updated.(model)

	if cmd == nil || m.mode != modeRefresh || m.refreshBackMode != modeDetail || m.refreshCursor != 2 {
		t.Fatalf("refresh picker state = %+v, cmd=%v", m, cmd)
	}
}

func TestRefreshPickerAppliesPredefinedInterval(t *testing.T) {
	m := model{
		mode:            modeRefresh,
		refresh:         20 * time.Second,
		refreshCursor:   3,
		refreshBackMode: modeJobs,
	}

	updated, cmd := m.handleRefreshKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)

	if cmd == nil || m.mode != modeJobs || m.refresh != 30*time.Second || m.refreshBackMode != 0 {
		t.Fatalf("applied refresh state = %+v, cmd=%v", m, cmd)
	}
	if m.message != "" {
		t.Fatalf("applying refresh interval left pane message %q", m.message)
	}
}

func TestRefreshPickerCancelKeepsInterval(t *testing.T) {
	m := model{
		mode:            modeRefresh,
		refresh:         20 * time.Second,
		refreshCursor:   3,
		refreshBackMode: modePipelines,
	}

	updated, _ := m.handleRefreshKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(model)

	if m.mode != modePipelines || m.refresh != 20*time.Second {
		t.Fatalf("canceled refresh state = %+v", m)
	}
}

func TestRefreshPickerShowsOptionsAndCurrentInterval(t *testing.T) {
	m := model{mode: modeRefresh, width: 80, height: 24, refresh: 20 * time.Second, refreshCursor: 2}
	view := ansi.Strip(m.View())

	for _, want := range []string{"Refetch interval", "current 20s", "5s", "30s", "1m0s", "5m0s", "* 20s"} {
		if !strings.Contains(view, want) {
			t.Fatalf("refresh picker view does not contain %q:\n%s", want, view)
		}
	}
}

func TestDetailRemainsVisibleBehindRefreshPicker(t *testing.T) {
	m := model{mode: modeRefresh, refreshBackMode: modeDetail, detailID: 10}
	if !m.detailVisible(10) {
		t.Fatal("detail was not considered visible behind refresh picker")
	}
}
