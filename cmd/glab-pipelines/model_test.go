package main

import (
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDetailUpdateIgnoresStaleResponse(t *testing.T) {
	m := model{
		mode:           modeDetail,
		detailID:       10,
		refresh:        time.Second,
		detailRequests: map[int]int{10: 2},
		detailPolls:    map[int]int{10: 1},
	}
	newer := detailMsg{
		pid:       10,
		requestID: 2,
		pollID:    1,
		detail:    detail{Pipeline: pipeline{ID: 10, Status: "success"}},
	}
	updated, _ := m.Update(newer)
	m = updated.(model)

	older := newer
	older.requestID = 1
	older.detail.Pipeline.Status = "running"
	updated, _ = m.Update(older)
	m = updated.(model)
	if m.detail == nil || m.detail.Pipeline.Status != "success" {
		t.Fatalf("stale response replaced detail: %+v", m.detail)
	}
}

func TestDetailUpdatePreservesTitleFromPipelineList(t *testing.T) {
	m := model{
		mode:           modeDetail,
		detailID:       10,
		refresh:        time.Second,
		list:           []pipeline{{ID: 10, CommitTitle: "Merge branch 'feature'"}},
		detailRequests: map[int]int{10: 1},
		detailPolls:    map[int]int{10: 1},
	}

	updated, _ := m.Update(detailMsg{
		pid:       10,
		requestID: 1,
		pollID:    1,
		detail:    detail{Pipeline: pipeline{ID: 10, Status: "success"}},
	})
	m = updated.(model)

	if m.detail == nil || m.detail.Pipeline.CommitTitle != "Merge branch 'feature'" {
		t.Fatalf("detail title was not preserved from list: %+v", m.detail)
	}
}

func TestDetailUpdateDoesNotMutateLogPane(t *testing.T) {
	m := model{
		mode:           modePipelines,
		refresh:        time.Second,
		detailRequests: map[int]int{10: 1},
		detailPolls:    map[int]int{10: 1},
		logPanes: []logPane{{
			ID:       1,
			Mode:     modeLogs,
			DetailID: 10,
			Loading:  true,
			Job:      &job{ID: 5, Status: "running"},
		}},
	}
	updated, _ := m.Update(detailMsg{
		pid:       10,
		requestID: 1,
		pollID:    1,
		detail:    detail{Jobs: []job{{ID: 5, Status: "success"}}},
	})
	m = updated.(model)
	if !m.logPanes[0].Loading || m.logPanes[0].Detail != nil {
		t.Fatalf("detail response mutated log pane: %+v", m.logPanes[0])
	}
}

func TestLogUpdateIgnoresStaleResponse(t *testing.T) {
	m := model{
		mode:        modeLogs,
		logRefresh:  time.Second,
		logJob:      &job{ID: 5, Status: "running"},
		logRequests: map[int64]int{5: 2},
		logPolls:    map[int64]int{5: 1},
	}
	updated, _ := m.Update(logsMsg{jobID: 5, requestID: 2, pollID: 1, logs: "new", job: &job{ID: 5, Status: "success"}})
	m = updated.(model)
	updated, _ = m.Update(logsMsg{jobID: 5, requestID: 1, pollID: 1, logs: "old", job: &job{ID: 5, Status: "running"}})
	m = updated.(model)
	if m.logs != "new" || m.logJob.Status != "success" {
		t.Fatalf("stale response replaced logs: logs=%q job=%+v", m.logs, m.logJob)
	}
}

func TestLogUpdateRemembersFinishedJobDuration(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	duration := 75.0
	m := model{
		repo:        "group/project",
		mode:        modeLogs,
		logRefresh:  time.Second,
		logJob:      &job{ID: 5, Name: "test", Status: "running"},
		logRequests: map[int64]int{5: 1},
		logPolls:    map[int64]int{5: 1},
	}
	updated, _ := m.Update(logsMsg{
		jobID:     5,
		requestID: 1,
		pollID:    1,
		job: &job{
			ID:         5,
			Name:       "test",
			Status:     "success",
			FinishedAt: "2026-07-10T10:00:00Z",
			Duration:   &duration,
		},
	})
	m = updated.(model)
	if got := m.jobDurations["test"]; got.Average != 75 || got.Count != 1 || got.LastJobID != 5 {
		t.Fatalf("recorded duration = %+v", got)
	}
	if got := loadJobDurations("group/project")["test"]; got.Average != 75 {
		t.Fatalf("persisted duration = %+v", got)
	}
}

func TestStalePollDoesNotStartRequest(t *testing.T) {
	m := model{
		mode:        modeDetail,
		detailID:    10,
		detailPolls: map[int]int{10: 2},
	}
	updated, cmd := m.Update(tickMsg{pid: 10, pollID: 1})
	if cmd != nil || updated.(model).nextRequestID != 0 {
		t.Fatal("stale poll started a request")
	}
}

func TestDetailUpdateRefreshesJobsOverlay(t *testing.T) {
	m := model{
		mode:           modeJobs,
		detailID:       10,
		refresh:        time.Second,
		detail:         &detail{Pipeline: pipeline{ID: 10, Status: "running"}},
		detailRequests: map[int]int{10: 1},
		detailPolls:    map[int]int{10: 1},
	}
	updated, _ := m.Update(detailMsg{
		pid:       10,
		requestID: 1,
		pollID:    1,
		detail:    detail{Pipeline: pipeline{ID: 10, Status: "success"}},
	})
	m = updated.(model)
	if m.mode != modeJobs || m.detail.Pipeline.Status != "success" {
		t.Fatalf("jobs overlay was not refreshed: mode=%d detail=%+v", m.mode, m.detail)
	}
}

func TestInactiveLogUpdateDoesNotReplaceActiveOverlay(t *testing.T) {
	m := model{
		mode:          modeTheme,
		themeBackMode: modeDetail,
		activeLogPane: 2,
		logRefresh:    time.Second,
		logRequests:   map[int64]int{5: 1},
		logPolls:      map[int64]int{5: 1},
		logPanes: []logPane{
			{ID: 1, Mode: modeLogs, Job: &job{ID: 5, Status: "running"}},
			{ID: 2, Mode: modeDetail, DetailID: 10},
		},
	}
	updated, _ := m.Update(logsMsg{jobID: 5, requestID: 1, pollID: 1, logs: "new", job: &job{ID: 5, Status: "running"}})
	m = updated.(model)
	if m.mode != modeTheme || m.activeLogPane != 2 {
		t.Fatalf("inactive update replaced active overlay: mode=%d pane=%d", m.mode, m.activeLogPane)
	}
	if m.logPanes[0].Logs != "new" {
		t.Fatalf("inactive pane was not updated: %+v", m.logPanes[0])
	}
}

func TestTraceFailureRetriesAfterTerminalStatus(t *testing.T) {
	m := model{
		mode:        modeLogs,
		logRefresh:  time.Nanosecond,
		logJob:      &job{ID: 5, Status: "running"},
		logRequests: map[int64]int{5: 1},
		logPolls:    map[int64]int{5: 1},
	}
	updated, cmd := m.Update(logsMsg{
		jobID:     5,
		requestID: 1,
		pollID:    1,
		job:       &job{ID: 5, Status: "success"},
		err:       errors.New("trace unavailable"),
	})
	m = updated.(model)
	if cmd == nil || m.logJob.Status != "success" {
		t.Fatalf("trace failure did not retain status or schedule retry: job=%+v", m.logJob)
	}
	result := cmd()
	var msg tea.Msg
	switch result := result.(type) {
	case logTickMsg:
		msg = result
	case tea.BatchMsg:
		if len(result) != 1 {
			t.Fatalf("retry batch has %d commands, want 1", len(result))
		}
		msg = result[0]()
	default:
		t.Fatalf("retry command result = %T", result)
	}
	tick, ok := msg.(logTickMsg)
	if !ok || !tick.force || tick.jobID != 5 {
		t.Fatalf("retry message = %#v", msg)
	}
}

func TestActionFailureKeepsOriginatingMode(t *testing.T) {
	m := model{mode: modeJobs, actionInFlight: true, actionRequest: 1}
	updated, _ := m.Update(actionMsg{requestID: 1, err: errors.New("denied")})
	m = updated.(model)
	if m.mode != modeJobs || m.actionInFlight || m.message != "denied" {
		t.Fatalf("action failure state = %+v", m)
	}
}

func TestConfirmationMessageSurvivesDetailPoll(t *testing.T) {
	m := model{
		mode:            modeConfirm,
		confirmBackMode: modeDetail,
		detailID:        10,
		message:         "confirmation did not match",
		refresh:         time.Second,
		detailRequests:  map[int]int{10: 1},
		detailPolls:     map[int]int{10: 1},
	}
	updated, _ := m.Update(detailMsg{pid: 10, requestID: 1, pollID: 1, detail: detail{Pipeline: pipeline{ID: 10}}})
	m = updated.(model)
	if m.message != "confirmation did not match" {
		t.Fatalf("confirmation message = %q", m.message)
	}
}

func TestCompletedTraceRetriesAreBounded(t *testing.T) {
	m := model{
		mode:        modeLogs,
		logRefresh:  time.Second,
		logJob:      &job{ID: 5, Status: "success"},
		logRequests: map[int64]int{5: 1},
		logPolls:    map[int64]int{5: 1},
	}
	msg := logsMsg{
		jobID:     5,
		requestID: 1,
		pollID:    1,
		job:       &job{ID: 5, Status: "success"},
		err:       errors.New("trace unavailable"),
	}
	for attempt := 1; attempt <= maxTraceRetries+1; attempt++ {
		updated, cmd := m.Update(msg)
		m = updated.(model)
		if attempt <= maxTraceRetries && cmd == nil {
			t.Fatalf("retry %d was not scheduled", attempt)
		}
		if attempt > maxTraceRetries && cmd != nil {
			t.Fatalf("retry %d exceeded retry limit", attempt)
		}
	}
}

func TestLeaveLogsStopsPanePolling(t *testing.T) {
	m := model{
		mode:          modeLogs,
		activeLogPane: 1,
		logJob:        &job{ID: 5, Status: "running"},
		logPanes:      []logPane{{ID: 1, Mode: modeLogs, Job: &job{ID: 5, Status: "running"}}},
	}
	m = m.leaveLogs(modeDetail)
	if m.mode != modeDetail || m.logPanes[0].Mode != modeDetail || m.hasLogJob(5) {
		t.Fatalf("logs remained active after leaving: mode=%d pane=%+v", m.mode, m.logPanes[0])
	}
}
