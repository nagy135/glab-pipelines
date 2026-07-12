package main

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

func TestKeepActiveLogPaneOnlyRetainsFocusedPane(t *testing.T) {
	m := model{
		mode:          modeDetail,
		activeLogPane: 2,
		logPanes: []logPane{
			{ID: 1, Mode: modePipelines, ListCursor: 3},
			{ID: 2, Mode: modeDetail, DetailID: 10, JobsCursor: 4, ShowInlineLogs: true},
			{ID: 3, Mode: modeLogs, Job: &job{ID: 5, Name: "build"}},
		},
		logSplitRoot: &logSplitNode{
			Direction: logSplitHorizontal,
			First:     &logSplitNode{PaneID: 1},
			Second: &logSplitNode{
				Direction: logSplitVertical,
				First:     &logSplitNode{PaneID: 2},
				Second:    &logSplitNode{PaneID: 3},
			},
		},
	}
	m = m.restoreLogPane(m.logPanes[1])
	m = m.keepActiveLogPaneOnly()
	if len(m.logPanes) != 1 || m.logPanes[0].ID != 2 {
		t.Fatalf("remaining panes = %#v", m.logPanes)
	}
	if m.logSplitRoot == nil || !m.logSplitRoot.isLeaf() || m.logSplitRoot.PaneID != 2 {
		t.Fatalf("split root = %#v", m.logSplitRoot)
	}
	if m.mode != modeDetail || m.detailID != 10 || m.jobsCursor != 4 || !m.showInlineLogs {
		t.Fatalf("focused pane state was not restored: %+v", m)
	}
}

func TestSplitPipelinePaneShowsRelativeStartTime(t *testing.T) {
	started := time.Now().Add(-12 * time.Minute)
	m := model{
		list: []pipeline{{
			ID:        10,
			Status:    "running",
			Ref:       "main",
			StartedAt: started.Format(time.RFC3339Nano),
		}},
	}
	view := ansi.Strip(m.renderPipelinePane(logPane{Mode: modePipelines}, true, 120, 8))
	if !strings.Contains(view, "STARTED") || !strings.Contains(view, "12m ago") {
		t.Fatalf("split pipeline pane does not show relative start time: %q", view)
	}
}

func TestSplitDetailPaneShowsLastRun(t *testing.T) {
	duration := 75.0
	detail := &detail{
		Pipeline: pipeline{ID: 10},
		DisplayJobs: []uiJob{{Current: job{
			Name:       "selected-job",
			Stage:      "test",
			Status:     "success",
			FinishedAt: time.Now().Add(-2 * time.Minute).Format(time.RFC3339Nano),
			Duration:   &duration,
		}}},
	}
	m := model{detail: detail, jobsCursor: 0}

	view := m.renderDetailPane(logPane{Mode: modeDetail, Detail: detail}, true, 80, 20)
	plain := ansi.Strip(view)
	if !strings.Contains(plain, "last duration 1m 15s") || !strings.Contains(plain, "ran 2m 0s ago") {
		t.Fatalf("split detail pane does not show last-run metadata: %q", plain)
	}
}
