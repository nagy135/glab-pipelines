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
`
	if err := os.WriteFile(script, []byte(contents), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	action := pendingAction{
		Target:     actionTargetPipeline,
		PipelineID: 123,
		Endpoint:   "cancel",
		Verb:       "Cancel",
	}
	if err := runProviderAction(providerGitLab, "group/project", action); err != nil {
		t.Fatal(err)
	}
}
