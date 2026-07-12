package main

import (
	"testing"
	"time"
)

func TestBuildDisplayJobsNewRunSupersedesManualJob(t *testing.T) {
	rows := buildDisplayJobs([]job{
		{ID: 10, Name: "deploy", Stage: "deploy", Status: "manual"},
		{ID: 11, Name: "deploy", Stage: "deploy", Status: "running"},
	})
	if len(rows) != 1 || rows[0].Current.ID != 11 {
		t.Fatalf("current job = %+v, want job 11", rows)
	}
}

func TestRenderJobLastRunUsesPreviousCompletedRun(t *testing.T) {
	duration := 75.0
	now := time.Date(2026, 7, 12, 10, 2, 3, 0, time.UTC)
	row := uiJob{
		Current: job{Name: "deploy", Status: "manual"},
		Previous: &job{
			Name:       "deploy",
			Status:     "success",
			FinishedAt: "2026-07-12T10:00:00Z",
			Duration:   &duration,
		},
	}

	if got, want := renderJobLastRun(row, now), "last duration 1m 15s   ran 2m 3s ago"; got != want {
		t.Fatalf("renderJobLastRun() = %q, want %q", got, want)
	}
}
