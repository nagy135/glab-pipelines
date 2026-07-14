package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

func TestDecodeJobCodeReturnsResolvedScripts(t *testing.T) {
	data := []byte(`{
		"valid": true,
		"jobs": [{
			"name": "test",
			"stage": "verify",
			"before_script": ["bundle install"],
			"script": ["go test ./...", "go vet ./..."],
			"after_script": ["rm -rf tmp"]
		}]
	}`)

	code, err := decodeJobCode(data, "test")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"# job: test",
		"# stage: verify",
		"# before_script\nbundle install",
		"# script\ngo test ./...\ngo vet ./...",
		"# after_script\nrm -rf tmp",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("resolved code %q does not contain %q", code, want)
		}
	}
}

func TestDecodeJobCodeReportsInvalidConfiguration(t *testing.T) {
	_, err := decodeJobCode([]byte(`{"valid":false,"errors":["include failed"]}`), "test")
	if err == nil || !strings.Contains(err.Error(), "include failed") {
		t.Fatalf("error = %v", err)
	}
}

func TestDecodeJobCodeReportsMissingJob(t *testing.T) {
	_, err := decodeJobCode([]byte(`{"valid":true,"jobs":[]}`), "test")
	if err == nil || !strings.Contains(err.Error(), `job "test" was not found`) {
		t.Fatalf("error = %v", err)
	}
}

func TestCodeUpdateIgnoresStaleResponse(t *testing.T) {
	v := viewport.New(40, 5)
	m := model{
		mode:          modeCode,
		activeLogPane: 1,
		logJob:        &job{ID: 5, Name: "test"},
		logsLoading:   true,
		logsViewport:  v,
		codeRequests:  map[int64]int{5: 2},
		logPanes: []logPane{{
			ID:       1,
			Mode:     modeCode,
			Job:      &job{ID: 5, Name: "test"},
			Loading:  true,
			Viewport: v,
		}},
	}

	updated, _ := m.Update(codeMsg{jobID: 5, requestID: 2, code: "new"})
	m = updated.(model)
	updated, _ = m.Update(codeMsg{jobID: 5, requestID: 1, code: "old", err: errors.New("old error")})
	m = updated.(model)
	if m.logs != "new" || m.logsLoading || m.message != "" {
		t.Fatalf("stale response changed code state: code=%q loading=%v message=%q", m.logs, m.logsLoading, m.message)
	}
}

func TestDetailCodeKeyOpensSelectedJob(t *testing.T) {
	m := model{
		mode:   modeDetail,
		width:  80,
		height: 24,
		detail: &detail{DisplayJobs: []uiJob{{Current: job{
			ID:   5,
			Name: "test",
			Pipeline: pipeline{
				SHA: "abc123",
			},
		}}}},
	}

	updated, cmd := m.handleDetailKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})
	m = updated.(model)
	if cmd == nil || m.mode != modeCode || m.logJob == nil || m.logJob.ID != 5 || !m.logsLoading {
		t.Fatalf("code view was not opened: mode=%d job=%+v loading=%v", m.mode, m.logJob, m.logsLoading)
	}
}
