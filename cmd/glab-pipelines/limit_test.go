package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestUpperLOpensLimitPickerFromPipelineList(t *testing.T) {
	m := model{mode: modePipelines, limit: 10}

	updated, cmd := m.handlePipelineKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	m = updated.(model)

	if cmd == nil || m.mode != modeLimit || m.limitCursor != 1 {
		t.Fatalf("limit picker state = %+v, cmd=%v", m, cmd)
	}
}

func TestLimitPickerAppliesOptionAndReloadsPipelines(t *testing.T) {
	m := model{
		mode:        modeLimit,
		limit:       10,
		limitCursor: 2,
		listRequest: 4,
		loadingList: false,
		status:      "active",
	}

	updated, cmd := m.handleLimitKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)

	if cmd == nil || m.mode != modePipelines || m.limit != 20 || m.listRequest != 5 || !m.loadingList {
		t.Fatalf("applied limit state = %+v, cmd=%v", m, cmd)
	}
	if m.message != "loading up to 20 pipelines..." {
		t.Fatalf("message = %q", m.message)
	}
}

func TestLimitPickerCancelKeepsLimit(t *testing.T) {
	m := model{mode: modeLimit, limit: 10, limitCursor: 2}

	updated, _ := m.handleLimitKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(model)

	if m.mode != modePipelines || m.limit != 10 {
		t.Fatalf("canceled limit state = %+v", m)
	}
}

func TestLimitPickerShowsOptionsAndCurrentLimit(t *testing.T) {
	m := model{mode: modeLimit, width: 80, height: 24, limit: 10, limitCursor: 1}
	view := ansi.Strip(m.View())

	for _, want := range []string{"Pipeline limit", "current 10", "* 10", "20", "50", "100"} {
		if !strings.Contains(view, want) {
			t.Fatalf("limit picker view does not contain %q:\n%s", want, view)
		}
	}
}

func TestDetailUpperLKeepsInlineLogToggle(t *testing.T) {
	m := model{mode: modeDetail, showInlineLogs: true}

	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	m = updated.(model)

	if m.mode != modeDetail || m.showInlineLogs {
		t.Fatalf("detail L did not retain inline log behavior: %+v", m)
	}
}
