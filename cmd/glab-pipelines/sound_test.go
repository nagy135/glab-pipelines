package main

import (
	"encoding/binary"
	"testing"
	"time"
)

func TestJobSoundForTransition(t *testing.T) {
	tests := []struct {
		name       string
		previous   string
		current    string
		wantSound  jobSound
		wantNotify bool
	}{
		{name: "running succeeds", previous: "running", current: "success", wantSound: jobSoundSuccess, wantNotify: true},
		{name: "running fails", previous: "running", current: "failed", wantSound: jobSoundFailure, wantNotify: true},
		{name: "pending completes between polls", previous: "pending", current: "success", wantSound: jobSoundSuccess, wantNotify: true},
		{name: "first observation is silent", current: "success"},
		{name: "unchanged success is silent", previous: "success", current: "success"},
		{name: "success does not become failure notification", previous: "success", current: "failed"},
		{name: "active transition is silent", previous: "pending", current: "running"},
		{name: "canceled is silent", previous: "running", current: "canceled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := jobSoundForTransition(tt.previous, tt.current)
			if ok != tt.wantNotify || got != tt.wantSound {
				t.Fatalf("jobSoundForTransition(%q, %q) = (%v, %v), want (%v, %v)", tt.previous, tt.current, got, ok, tt.wantSound, tt.wantNotify)
			}
		})
	}
}

func TestObserveJobStatusesNotifiesOnce(t *testing.T) {
	m := model{}
	m, sounds := m.observeJobStatuses([]job{{ID: 1, Status: "running"}, {ID: 2, Status: "pending"}})
	if len(sounds) != 0 {
		t.Fatalf("initial observation produced %d sounds", len(sounds))
	}

	m, sounds = m.observeJobStatuses([]job{{ID: 1, Status: "success"}, {ID: 2, Status: "failed"}})
	if len(sounds) != 2 || sounds[0] != jobSoundSuccess || sounds[1] != jobSoundFailure {
		t.Fatalf("completion sounds = %v, want success then failure", sounds)
	}

	_, sounds = m.observeJobStatuses([]job{{ID: 1, Status: "success"}, {ID: 2, Status: "failed"}})
	if len(sounds) != 0 {
		t.Fatalf("repeated terminal observation produced %d sounds", len(sounds))
	}

	m, sounds = m.observeJobStatuses([]job{{ID: 1, Status: "running"}})
	if len(sounds) != 0 || m.jobStatuses[1] != "success" {
		t.Fatalf("stale observation regressed terminal status: sounds=%v status=%q", sounds, m.jobStatuses[1])
	}
}

func TestDetailUpdateQueuesCompletionSound(t *testing.T) {
	m := model{
		mode:        modeDetail,
		detailID:    10,
		jobStatuses: map[int64]string{1: "running"},
		logPanes: []logPane{{
			ID:       1,
			Mode:     modeLogs,
			DetailID: 10,
			Job:      &job{ID: 1, Status: "running"},
		}},
	}

	updated, cmd := m.Update(detailMsg{
		pid: 10,
		detail: detail{
			Jobs:        []job{{ID: 1, Status: "success"}},
			DisplayJobs: []uiJob{{Current: job{ID: 1, Status: "success"}}},
		},
	})
	got := updated.(model)
	if cmd == nil {
		t.Fatal("detail completion did not queue a sound command")
	}
	if got.logPanes[0].Job.Status != "success" {
		t.Fatalf("split log job status = %q, want success", got.logPanes[0].Job.Status)
	}
}

func TestLogUpdateQueuesCompletionSoundForInactiveSplit(t *testing.T) {
	m := model{
		mode:          modePipelines,
		activeLogPane: 2,
		jobStatuses:   map[int64]string{1: "running"},
		logPanes: []logPane{
			{ID: 1, Mode: modeLogs, Job: &job{ID: 1, Status: "running"}},
			{ID: 2, Mode: modePipelines},
		},
	}

	updated, cmd := m.Update(logsMsg{
		jobID: 1,
		logs:  "complete",
		job:   &job{ID: 1, Status: "failed"},
	})
	got := updated.(model)
	if cmd == nil {
		t.Fatal("inactive split completion did not queue a sound command")
	}
	if got.logPanes[0].Job.Status != "failed" {
		t.Fatalf("split log job status = %q, want failed", got.logPanes[0].Job.Status)
	}
}

func TestInactiveSplitsKeepPolling(t *testing.T) {
	m := model{
		mode:       modePipelines,
		refresh:    time.Second,
		logRefresh: time.Second,
		logPanes: []logPane{
			{ID: 1, Mode: modeLogs, Job: &job{ID: 1, Status: "running"}},
			{ID: 2, Mode: modeDetail, DetailID: 10},
		},
	}

	if _, cmd := m.Update(logTickMsg{jobID: 1}); cmd == nil {
		t.Fatal("inactive running log split stopped polling")
	}
	if _, cmd := m.Update(tickMsg{pid: 10}); cmd == nil {
		t.Fatal("inactive detail split stopped polling")
	}
}

func TestSynthesizeWAV(t *testing.T) {
	wav := synthesizeWAV([]soundNote{{frequency: 440, duration: 0.05}})
	if len(wav) <= 44 {
		t.Fatalf("WAV length = %d, want audio data", len(wav))
	}
	if string(wav[:4]) != "RIFF" || string(wav[8:12]) != "WAVE" {
		t.Fatalf("invalid WAV header: %q ... %q", wav[:4], wav[8:12])
	}
	dataLength := binary.LittleEndian.Uint32(wav[40:44])
	if int(dataLength) != len(wav)-44 {
		t.Fatalf("WAV data length = %d, want %d", dataLength, len(wav)-44)
	}
}
