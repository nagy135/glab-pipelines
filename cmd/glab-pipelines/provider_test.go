package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunProviderPipelineCancelUsesGitLabPipelineEndpoint(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "glab")
	contents := `#!/bin/sh
if [ "$*" != "api -R group/project --method POST projects/:id/pipelines/123/cancel" ]; then
  echo "unexpected arguments: $*" >&2
  exit 1
fi
printf '{"id":123,"status":"canceled","updated_at":"2026-08-26T17:00:00Z"}\n'
`
	if err := os.WriteFile(script, []byte(contents), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	action := pendingAction{
		Target:     actionTargetPipeline,
		Pipeline:   pipeline{ID: 123, Status: "running", Ref: "main"},
		PipelineID: 123,
		Endpoint:   "cancel",
		Verb:       "Cancel",
	}
	updated, err := runProviderAction(providerGitLab, "group/project", action)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != 123 || updated.Status != "canceled" || updated.Ref != "main" || updated.UpdatedAt != "2026-08-26T17:00:00Z" {
		t.Fatalf("updated pipeline = %+v", updated)
	}
}
