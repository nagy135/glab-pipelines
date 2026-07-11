package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestLoadingPaneShowsSpinnerAtTopLeft(t *testing.T) {
	m := model{activity: newActivitySpinner()}
	view := ansi.Strip(m.renderPaneBox("title\nbody", true, true, 20, 4))
	lines := strings.Split(view, "\n")
	indicator := ansi.Strip(m.activityIndicator())
	if len(lines) < 2 || !strings.HasPrefix(lines[1], "║"+indicator+" ") {
		t.Fatalf("loading pane does not have a top-left spinner: %q", view)
	}
}

func TestIdlePaneDoesNotShowSpinner(t *testing.T) {
	m := model{activity: newActivitySpinner()}
	view := ansi.Strip(m.renderPaneBox("title\nbody", true, false, 20, 4))
	indicator := ansi.Strip(m.activityIndicator())
	if strings.Contains(view, indicator) {
		t.Fatalf("idle pane contains spinner %q: %q", indicator, view)
	}
}

func TestPaneReservesSpinnerSpaceWhileIdle(t *testing.T) {
	m := model{activity: newActivitySpinner()}
	loading := strings.Split(ansi.Strip(m.renderPaneBox("title\nbody", true, true, 20, 4)), "\n")
	idle := strings.Split(ansi.Strip(m.renderPaneBox("title\nbody", true, false, 20, 4)), "\n")
	loadingPrefix, _, _ := strings.Cut(loading[1], "title")
	idlePrefix, _, _ := strings.Cut(idle[1], "title")
	if ansi.StringWidth(loadingPrefix) != ansi.StringWidth(idlePrefix) {
		t.Fatalf("title shifts when loading changes: loading=%q idle=%q", loading[1], idle[1])
	}
}

func TestSpinnerTickAdvancesFrame(t *testing.T) {
	m := model{activity: newActivitySpinner()}
	before := ansi.Strip(m.activityIndicator())
	updated, cmd := m.Update(m.activity.Tick())
	m = updated.(model)
	if after := ansi.Strip(m.activityIndicator()); after == before {
		t.Fatalf("spinner frame did not advance from %q", before)
	}
	if cmd == nil {
		t.Fatal("spinner tick did not schedule the next frame")
	}
}
