package main

import (
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestLatestLogLines(t *testing.T) {
	logs := "one\ntwo\nthree\nfour\nfive\nsix\n"
	want := []string{"two", "three", "four", "five", "six"}
	if got := latestLogLines(logs, inlineLogLineCount); !reflect.DeepEqual(got, want) {
		t.Fatalf("latestLogLines() = %#v, want %#v", got, want)
	}
}

func TestToggleInlineLogsRequestsRunningAndFailedJobs(t *testing.T) {
	m := model{
		mode:       modeDetail,
		detailID:   10,
		logRefresh: time.Second,
		detail: &detail{DisplayJobs: []uiJob{
			{Current: job{ID: 1, Status: "running"}},
			{Current: job{ID: 2, Status: "failed"}},
			{Current: job{ID: 3, Status: "success"}},
		}},
	}
	updated, cmd := m.handleDetailKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	m = updated.(model)
	if !m.showInlineLogs || cmd == nil {
		t.Fatal("inline logs were not enabled or polling was not started")
	}
	if !m.inlineLogsLoading[1] || !m.inlineLogsLoading[2] || m.inlineLogsLoading[3] {
		t.Fatalf("requested jobs = %#v", m.inlineLogsLoading)
	}

	updated, cmd = m.handleDetailKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	m = updated.(model)
	if m.showInlineLogs || cmd != nil || len(m.inlineLogsLoading) != 0 {
		t.Fatalf("inline logs were not disabled cleanly: enabled=%v loading=%#v", m.showInlineLogs, m.inlineLogsLoading)
	}
}

func TestToggleInlineLogsOnlyChangesFocusedPane(t *testing.T) {
	d := &detail{Pipeline: pipeline{ID: 10}, DisplayJobs: []uiJob{{Current: job{ID: 1, Status: "running"}}}}
	m := model{
		mode:          modeDetail,
		detailID:      10,
		detail:        d,
		logRefresh:    time.Second,
		activeLogPane: 1,
		logPanes: []logPane{
			{ID: 1, Mode: modeDetail, DetailID: 10, Detail: cloneDetailPtr(d)},
			{ID: 2, Mode: modeDetail, DetailID: 10, Detail: cloneDetailPtr(d)},
		},
	}
	updated, _ := m.handleDetailKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	m = updated.(model)
	if !m.logPanes[0].ShowInlineLogs || m.logPanes[1].ShowInlineLogs {
		t.Fatalf("pane inline modes = %v, %v", m.logPanes[0].ShowInlineLogs, m.logPanes[1].ShowInlineLogs)
	}

	m = m.restoreLogPane(m.logPanes[1])
	if m.showInlineLogs {
		t.Fatal("unfocused pane inherited inline log mode")
	}
}

func TestRequestInlineLogsSkipsCachedFailedJob(t *testing.T) {
	m := model{
		inlineLogs: map[int64]inlineLogSnippet{
			1: {Lines: []string{"failure"}, Status: "failed"},
		},
	}
	updated, cmd := m.requestInlineLogs([]uiJob{
		{Current: job{ID: 1, Status: "failed"}},
		{Current: job{ID: 2, Status: "running"}},
	})
	m = updated
	if cmd == nil || m.inlineLogsLoading[1] || !m.inlineLogsLoading[2] {
		t.Fatalf("inline log requests = %#v", m.inlineLogsLoading)
	}
}

func TestInlineLogUpdateIgnoresStaleResponse(t *testing.T) {
	m := model{
		showInlineLogs:    true,
		inlineLogPollID:   2,
		inlineLogRequests: map[int64]int{5: 3},
		inlineLogsLoading: map[int64]bool{5: true},
	}
	updated, _ := m.Update(inlineLogMsg{
		jobID: 5, requestID: 2, pollID: 2, status: "running", lines: []string{"old"},
	})
	m = updated.(model)
	if _, ok := m.inlineLogs[5]; ok || !m.inlineLogsLoading[5] {
		t.Fatalf("stale response changed state: logs=%#v loading=%#v", m.inlineLogs, m.inlineLogsLoading)
	}

	updated, _ = m.Update(inlineLogMsg{
		jobID: 5, requestID: 3, pollID: 2, status: "failed", lines: []string{"final"},
	})
	m = updated.(model)
	if got := m.inlineLogs[5]; got.Status != "failed" || !reflect.DeepEqual(got.Lines, []string{"final"}) || m.inlineLogsLoading[5] {
		t.Fatalf("current response state = logs=%#v loading=%#v", got, m.inlineLogsLoading)
	}
}

func TestDetailViewRendersInlineLogsForSupportedStatuses(t *testing.T) {
	m := model{
		mode:           modeDetail,
		width:          120,
		detailID:       10,
		showInlineLogs: true,
		inlineLogs: map[int64]inlineLogSnippet{
			1: {Lines: []string{"running output"}, Status: "running"},
			2: {Lines: []string{"failed output"}, Status: "failed"},
			3: {Lines: []string{"successful output"}, Status: "success"},
		},
		detail: &detail{
			Pipeline: pipeline{ID: 10, Status: "running"},
			DisplayJobs: []uiJob{
				{Current: job{ID: 1, Name: "build", Stage: "test", Status: "running"}},
				{Current: job{ID: 2, Name: "package", Stage: "test", Status: "failed"}},
				{Current: job{ID: 3, Name: "deploy", Stage: "test", Status: "success"}},
			},
		},
	}
	view := ansi.Strip(m.viewDetail())
	for _, want := range []string{"running output", "failed output", "inline logs on"} {
		if !strings.Contains(view, want) {
			t.Fatalf("detail view does not contain %q: %q", want, view)
		}
	}
	if strings.Contains(view, "successful output") {
		t.Fatalf("detail view rendered logs for a successful job: %q", view)
	}
}
