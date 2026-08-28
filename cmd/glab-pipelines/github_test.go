package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInitialModelExplicitGitHubProvider(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", filepath.Join(t.TempDir(), "cache"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("CI_TUI_PROVIDER", "")
	m, err := initialModel([]string{"--github", "-R", "owner/repo", "all"})
	if err != nil {
		t.Fatal(err)
	}
	if m.provider != providerGitHub || m.repo != "owner/repo" || m.status != "all" {
		t.Fatalf("initialModel() provider=%q repo=%q status=%q", m.provider.name(), m.repo, m.status)
	}
	if !m.showInlineLogs {
		t.Fatal("initialModel() did not enable inline logs by default")
	}
}

func TestProviderFromRemoteURL(t *testing.T) {
	tests := []struct {
		remote string
		want   ciProvider
	}{
		{"git@github.com:owner/repo.git", providerGitHub},
		{"https://github.com/owner/repo.git", providerGitHub},
		{"github.com/owner/repo", providerGitHub},
		{"git@gitlab.com:group/project.git", providerGitLab},
		{"group/project", providerGitLab},
	}
	for _, tt := range tests {
		t.Run(tt.remote, func(t *testing.T) {
			if got := providerFromRemoteURL(tt.remote); got != tt.want {
				t.Fatalf("providerFromRemoteURL(%q) = %q, want %q", tt.remote, got.name(), tt.want.name())
			}
		})
	}
}

func TestGitHubStatus(t *testing.T) {
	tests := []struct {
		status     string
		conclusion string
		want       string
	}{
		{"in_progress", "", "running"},
		{"queued", "", "pending"},
		{"completed", "success", "success"},
		{"completed", "failure", "failed"},
		{"completed", "timed_out", "failed"},
		{"completed", "cancelled", "canceled"},
		{"completed", "skipped", "skipped"},
	}
	for _, tt := range tests {
		if got := githubStatus(tt.status, tt.conclusion); got != tt.want {
			t.Errorf("githubStatus(%q, %q) = %q, want %q", tt.status, tt.conclusion, got, tt.want)
		}
	}
}

func TestGitHubRunAndJobConversion(t *testing.T) {
	run := githubWorkflowRun{
		ID:           123,
		Name:         "CI",
		DisplayTitle: "Test GitHub support",
		HeadBranch:   "feature/github",
		HeadSHA:      "abcdef123456",
		Status:       "completed",
		Conclusion:   "success",
		Event:        "pull_request",
		CreatedAt:    "2026-08-04T10:00:00Z",
		UpdatedAt:    "2026-08-04T10:02:00Z",
		RunStartedAt: "2026-08-04T10:00:30Z",
		HTMLURL:      "https://github.com/owner/repo/actions/runs/123",
		Path:         ".github/workflows/ci.yml",
	}
	p := githubRunToPipeline(run)
	if p.Status != "success" || p.Ref != "feature/github" || p.CommitTitle != run.DisplayTitle || p.WorkflowPath != run.Path {
		t.Fatalf("githubRunToPipeline() = %+v", p)
	}
	if p.Duration == nil || *p.Duration != 90 {
		t.Fatalf("pipeline duration = %v, want 90", p.Duration)
	}

	j := githubJobToJob(githubJob{
		ID:           456,
		RunID:        run.ID,
		Name:         "test (go-1.22)",
		HTMLURL:      "https://github.com/owner/repo/actions/runs/123/job/456",
		WorkflowName: "CI",
		Status:       "completed",
		Conclusion:   "failure",
		StartedAt:    "2026-08-04T10:00:40Z",
		CompletedAt:  "2026-08-04T10:01:10Z",
	}, p)
	if j.Status != "failed" || j.Stage != "CI" || j.Pipeline.ID != run.ID {
		t.Fatalf("githubJobToJob() = %+v", j)
	}
	if j.WebURL != "https://github.com/owner/repo/actions/runs/123/job/456" {
		t.Fatalf("job web URL = %q", j.WebURL)
	}
	if j.Duration == nil || *j.Duration != 30 {
		t.Fatalf("job duration = %v, want 30", j.Duration)
	}
}

func TestGitHubActionsUseJobRerunAndRunCancel(t *testing.T) {
	finished := job{ID: 42, Status: "failed"}
	action, ok := resolveAction(providerGitHub, "s", finished)
	if !ok || action.Endpoint != "rerun" || action.Verb != "Rerun" {
		t.Fatalf("rerun action = %+v, %v", action, ok)
	}

	running := job{ID: 43, Status: "running"}
	action, ok = resolveAction(providerGitHub, "c", running)
	if !ok || action.Endpoint != "cancel" {
		t.Fatalf("cancel action = %+v, %v", action, ok)
	}
}

func TestGitHubAPIUsesGHRepoAndExpectedArguments(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "gh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf '%s\\n' \"$GH_REPO|$*\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	out, err := githubAPIContext(ctx, "owner/repo", "POST", "repos/{owner}/{repo}/actions/jobs/42/rerun", "Accept: application/vnd.github+json")
	if err != nil {
		t.Fatal(err)
	}
	want := "owner/repo|api --method POST -H Accept: application/vnd.github+json repos/{owner}/{repo}/actions/jobs/42/rerun"
	if got := strings.TrimSpace(string(out)); got != want {
		t.Fatalf("gh invocation = %q, want %q", got, want)
	}
}
