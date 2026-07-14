package main

import (
	"strings"
	"testing"
	"time"

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

func TestDetailViewShowsTypicalDurationProgress(t *testing.T) {
	started := time.Now().Add(-30 * time.Second)
	m := model{
		mode:         modeDetail,
		width:        100,
		height:       30,
		detailID:     10,
		jobDurations: map[string]jobDurationStat{"test": {Average: 60, Count: 1}},
		detail: &detail{
			Pipeline: pipeline{ID: 10, Status: "running"},
			DisplayJobs: []uiJob{{Current: job{
				Name:      "test",
				Stage:     "test",
				Status:    "running",
				StartedAt: started.Format(time.RFC3339Nano),
			}}},
		},
	}

	view := ansi.Strip(m.viewDetail())
	if !strings.Contains(view, "usually 1m00s") || !strings.Contains(view, "%") {
		t.Fatalf("detail view does not show expected progress: %q", view)
	}
}

func TestPipelineListShowsWhenPipelineStarted(t *testing.T) {
	started := time.Now().Add(-12 * time.Minute)
	m := model{
		width:  140,
		height: 20,
		list: []pipeline{{
			ID:        10,
			Status:    "running",
			Ref:       "main",
			StartedAt: started.Format(time.RFC3339Nano),
		}},
	}

	view := ansi.Strip(m.viewPipelines())
	if !strings.Contains(view, "STARTED") || !strings.Contains(view, "12m ago") {
		t.Fatalf("pipeline list does not show relative start time: %q", view)
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
			DisplayJobs: []uiJob{{Current: job{Name: "test pipeline detail", Stage: "test", Status: "running"}}},
		},
	}

	view := ansi.Strip(m.viewDetail())
	for _, want := range []string{"title Make pipeline details clearer", "branch", "feature/visible-branch", "╭"} {
		if !strings.Contains(view, want) {
			t.Fatalf("detail view does not contain %q: %q", want, view)
		}
	}
	if strings.Count(view, "╭") < 2 || !strings.Contains(view, "test pipeline detail") {
		t.Fatalf("detail view does not render bordered job names: %q", view)
	}
}

func TestScrollableBodyAddsScrollbarAndUsesOffset(t *testing.T) {
	body := strings.Join([]string{"one", "two", "three", "four", "five"}, "\n")
	view := ansi.Strip(renderScrollableBody(body, 8, 3, 2, 0, false))

	if !strings.Contains(view, "three") || !strings.Contains(view, "five") {
		t.Fatalf("scrolled view has the wrong lines: %q", view)
	}
	if strings.Contains(view, "one") || !strings.Contains(view, "█") {
		t.Fatalf("scrolled view is missing clipping or scrollbar: %q", view)
	}
}

func TestScrollableBodySupportsHorizontalOverflow(t *testing.T) {
	body := "ID  STATUS  REF  STARTED  DURATION  TITLE"
	view := ansi.Strip(renderScrollableBody(body, 16, 3, 0, 12, false))

	if !strings.Contains(view, "REF") || strings.Contains(view, "ID  STATUS") || !strings.Contains(view, "█") {
		t.Fatalf("horizontal viewport has the wrong content or scrollbar: %q", view)
	}
}

func TestScrollableBodyWrapsInsteadOfAddingHorizontalOverflow(t *testing.T) {
	body := "0123456789abcdefghijklmnop"
	view := ansi.Strip(renderScrollableBody(body, 10, 6, 0, 8, true))

	if strings.Contains(view, "─") || !strings.Contains(view, "0123456789\n") {
		t.Fatalf("wrapped viewport still has horizontal overflow: %q", view)
	}
	for _, line := range strings.Split(view, "\n") {
		if ansi.StringWidth(line) > 10 {
			t.Fatalf("wrapped line width = %d, want <= 10: %q", ansi.StringWidth(line), line)
		}
	}
}
