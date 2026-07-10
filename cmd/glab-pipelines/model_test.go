package main

import (
	"testing"
	"time"
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
