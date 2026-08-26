package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPipelineCacheRoundTrip(t *testing.T) {
	cacheRoot := t.TempDir()
	t.Setenv("HOME", cacheRoot)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(cacheRoot, "cache"))

	want := []pipeline{{ID: 7, IID: 3, Ref: "main", CommitTitle: "release"}}
	savePipelineCache("group/project", "active", 10, want)
	got, ok := loadPipelineCache("group/project", "active", 10)
	if !ok || len(got) != 1 || got[0].ID != 7 || got[0].IID != 3 || got[0].Commit.Title != "release" {
		t.Fatalf("loadPipelineCache() = %+v, %v", got, ok)
	}

	path, err := pipelineCachePath("group/project", "active", 10)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("cache permissions = %o, want 600", info.Mode().Perm())
	}
}

func TestPipelineCacheRejectsMismatchedMetadata(t *testing.T) {
	cacheRoot := t.TempDir()
	t.Setenv("HOME", cacheRoot)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(cacheRoot, "cache"))

	path, err := pipelineCachePath("group/project", "active", 10)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte(`{"version":2,"repo":"other/project","status":"active","limit":10,"pipelines":[{"id":7}]}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if got, ok := loadPipelineCache("group/project", "active", 10); ok || got != nil {
		t.Fatalf("mismatched cache accepted: %+v", got)
	}
}
