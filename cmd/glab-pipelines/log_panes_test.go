package main

import "testing"

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
