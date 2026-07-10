package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestConfirmViewIsCenteredOverOriginatingView(t *testing.T) {
	m := model{
		mode:            modeConfirm,
		confirmBackMode: modeDetail,
		width:           100,
		height:          30,
		detailID:        10,
		detail: &detail{
			Pipeline: pipeline{ID: 10, Status: "running", Ref: "main"},
		},
		pending: &pendingAction{
			Job:        job{ID: 5, Name: "deploy", Status: "manual"},
			PipelineID: 10,
			Endpoint:   "play",
			Verb:       "Play",
		},
	}
	view := ansi.Strip(m.viewConfirm())
	if !strings.Contains(view, "Pipeline #10") || !strings.Contains(view, "Play job?") {
		t.Fatalf("confirmation does not preserve background and modal: %q", view)
	}
	lines := strings.Split(view, "\n")
	modalRow := -1
	for i, line := range lines {
		if strings.Contains(line, "Play job?") {
			modalRow = i
			break
		}
	}
	if modalRow < m.height/4 || modalRow > 3*m.height/4 {
		t.Fatalf("modal row = %d, want centered", modalRow)
	}
	modalLine := strings.TrimSpace(lines[modalRow])
	if !strings.HasPrefix(modalLine, "║") || !strings.HasSuffix(modalLine, "║") {
		t.Fatalf("modal line is missing a wall: %q", modalLine)
	}
}

func TestConfirmViewFitsNarrowTerminal(t *testing.T) {
	m := model{
		mode:            modeConfirm,
		confirmBackMode: modeDetail,
		width:           20,
		height:          12,
		detailID:        10,
		pending: &pendingAction{
			Job:      job{ID: 5, Name: "deploy-production", Status: "manual"},
			Endpoint: "play",
			Verb:     "Play",
		},
	}
	for i, line := range strings.Split(m.viewConfirm(), "\n") {
		if width := ansi.StringWidth(line); width > m.width {
			t.Fatalf("line %d width = %d, want <= %d", i, width, m.width)
		}
	}
}

func TestDetailViewShowsPipelineTitleAndBranch(t *testing.T) {
	m := model{
		mode:     modeDetail,
		width:    100,
		height:   30,
		detailID: 10,
		detail: &detail{
			Pipeline: pipeline{
				ID:          10,
				Status:      "running",
				Ref:         "feature/visible-branch",
				CommitTitle: "Make pipeline details clearer",
			},
		},
	}

	view := ansi.Strip(m.viewDetail())
	for _, want := range []string{"title Make pipeline details clearer", "branch", "feature/visible-branch", "╭"} {
		if !strings.Contains(view, want) {
			t.Fatalf("detail view does not contain %q: %q", want, view)
		}
	}
}
