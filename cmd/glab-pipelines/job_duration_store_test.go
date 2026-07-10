package main

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

func TestJobDurationsAreRepositoryScoped(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	want := map[string]jobDurationStat{
		"test": {Average: 42, Count: 2, LastJobID: 10},
	}
	saveJobDurations("group/one", want)

	got := loadJobDurations("group/one")
	if got["test"] != want["test"] {
		t.Fatalf("loadJobDurations() = %+v, want %+v", got, want)
	}
	if other := loadJobDurations("group/two"); len(other) != 0 {
		t.Fatalf("other repository loaded durations: %+v", other)
	}

	path, _, err := jobDurationStorePath("group/one")
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("store permissions = %o, want 600", info.Mode().Perm())
	}
	if filepath.Dir(filepath.Dir(path)) != filepath.Join(home, ".local", "share", "glab-pipelines") {
		t.Fatalf("unexpected store path: %s", path)
	}
}

func TestRecordJobDurationsAveragesFinishedJobsOnce(t *testing.T) {
	stats := make(map[string]jobDurationStat)
	d30, d50 := 30.0, 50.0
	jobs := []job{
		{ID: 1, Name: "test", Status: "success", FinishedAt: "2026-07-10T10:00:00Z", Duration: &d30},
		{ID: 2, Name: "test", Status: "failed", FinishedAt: "2026-07-10T10:01:00Z", Duration: &d50},
		{ID: 3, Name: "build", Status: "running", Duration: &d30},
	}
	if !recordJobDurations(stats, jobs) {
		t.Fatal("finished jobs were not recorded")
	}
	if got := stats["test"]; got.Average != 40 || got.Count != 2 || got.LastJobID != 2 {
		t.Fatalf("recorded stat = %+v", got)
	}
	if recordJobDurations(stats, jobs) {
		t.Fatal("same jobs were recorded twice")
	}
	if _, ok := stats["build"]; ok {
		t.Fatal("running job was recorded")
	}
}

func TestRenderJobProgress(t *testing.T) {
	started := time.Date(2026, 7, 10, 10, 0, 0, 0, time.UTC)
	j := job{Name: "test", Status: "running", StartedAt: started.Format(time.RFC3339)}
	progress := ansi.Strip(renderJobProgress(j, 100, started.Add(40*time.Second), 80))
	if !strings.Contains(progress, "40%") || !strings.Contains(progress, "40s / usually 1m40s") {
		t.Fatalf("progress = %q", progress)
	}

	j.Status = "success"
	if progress := renderJobProgress(j, math.MaxFloat64, started, 80); progress != "" {
		t.Fatalf("completed job progress = %q", progress)
	}
}
