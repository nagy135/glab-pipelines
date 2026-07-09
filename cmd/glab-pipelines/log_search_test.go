package main

import "testing"

func TestFindLogSearchMatchesCaseInsensitive(t *testing.T) {
	logs := "starting\nERROR failed\ndone\nretry error"
	matches := findLogSearchMatches(logs, "error")
	want := []logSearchMatch{{Line: 1, Start: 0, End: 5}, {Line: 3, Start: 6, End: 11}}
	if len(matches) != len(want) {
		t.Fatalf("len(matches) = %d, want %d", len(matches), len(want))
	}
	for i := range want {
		if matches[i] != want[i] {
			t.Fatalf("matches[%d] = %+v, want %+v", i, matches[i], want[i])
		}
	}
}

func TestLogSearchMatchNearWraps(t *testing.T) {
	matches := []logSearchMatch{{Line: 2}, {Line: 5}, {Line: 9}}
	if got := logSearchMatchNear(matches, 6, 1); got != 2 {
		t.Fatalf("forward match index = %d, want 2", got)
	}
	if got := logSearchMatchNear(matches, 10, 1); got != 0 {
		t.Fatalf("forward wrapped match index = %d, want 0", got)
	}
	if got := logSearchMatchNear(matches, 4, -1); got != 0 {
		t.Fatalf("backward match index = %d, want 0", got)
	}
	if got := logSearchMatchNear(matches, 1, -1); got != 2 {
		t.Fatalf("backward wrapped match index = %d, want 2", got)
	}
}
