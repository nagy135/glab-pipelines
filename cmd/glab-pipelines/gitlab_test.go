package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildPipelineListGraphQLQueryActive(t *testing.T) {
	query, aliases, err := buildPipelineListGraphQLQuery("active")
	if err != nil {
		t.Fatal(err)
	}
	if len(aliases) != len(activeStatuses) {
		t.Fatalf("aliases = %v, want one for each active status", aliases)
	}
	for _, status := range activeStatuses {
		statusEnum := strings.ToUpper(status)
		if !strings.Contains(query, "status: "+statusEnum) {
			t.Errorf("query does not include %s status: %s", statusEnum, query)
		}
	}
	for _, field := range []string{"startedAt", "duration", "commit { title }"} {
		if !strings.Contains(query, field) {
			t.Errorf("query does not include %q: %s", field, query)
		}
	}
}

func TestFetchPipelinesUsesOneEnrichedGraphQLRequest(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "glab")
	contents := `#!/bin/sh
if [ "$1" != "api" ] || [ "$2" != "-R" ] || [ "$3" != "group/project" ] || [ "$4" != "graphql" ]; then
  echo "unexpected arguments: $*" >&2
  exit 1
fi
shift 4
query=""
full_path=""
limit=""
while [ "$#" -gt 0 ]; do
  flag="$1"
  value="$2"
  shift 2
  case "$flag:$value" in
    -f:query=*) query="${value#query=}" ;;
    -f:fullPath=*) full_path="${value#fullPath=}" ;;
    -F:limit=*) limit="${value#limit=}" ;;
    *) echo "unexpected GraphQL argument: $flag $value" >&2; exit 1 ;;
  esac
done
case "$query" in
  *"status: RUNNING"*"startedAt"*"commit { title }"*) ;;
  *) echo "unexpected query: $query" >&2; exit 1 ;;
esac
if [ "$full_path" != "group/project" ] || [ "$limit" != "2" ]; then
  echo "unexpected variables: fullPath=$full_path limit=$limit" >&2
  exit 1
fi
printf '%s\n' '{"data":{"project":{"pipelines0":{"nodes":[{"id":"gid://gitlab/Ci::Pipeline/41","status":"RUNNING","ref":"main","sha":"abc123","source":"push","updatedAt":"2026-08-26T17:01:00Z","createdAt":"2026-08-26T17:00:00Z","startedAt":"2026-08-26T17:00:10Z","duration":12,"commit":{"title":"Run GraphQL"}}]},"pipelines1":{"nodes":[{"id":"gid://gitlab/Ci::Pipeline/43","status":"PENDING","ref":"feature","sha":"def456","source":"merge_request_event","updatedAt":"2026-08-26T17:03:00Z","createdAt":"2026-08-26T17:02:00Z","startedAt":null,"duration":null,"commit":{"title":"Pending pipeline"}}]},"pipelines2":{"nodes":[]},"pipelines3":{"nodes":[]},"pipelines4":{"nodes":[]},"pipelines5":{"nodes":[]},"pipelines6":{"nodes":[]}}}}'
`
	if err := os.WriteFile(script, []byte(contents), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	pipelines, err := fetchPipelines("group/project", "active", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(pipelines) != 2 || pipelines[0].ID != 43 || pipelines[1].ID != 41 {
		t.Fatalf("pipelines = %+v", pipelines)
	}
	if pipelines[0].Status != "pending" || pipelines[1].Status != "running" {
		t.Fatalf("pipeline statuses = %q, %q", pipelines[0].Status, pipelines[1].Status)
	}
	if pipelines[1].StartedAt != "2026-08-26T17:00:10Z" || pipelines[1].Duration == nil || *pipelines[1].Duration != 12 || pipelines[1].CommitTitle != "Run GraphQL" {
		t.Fatalf("enriched pipeline = %+v", pipelines[1])
	}
}

func TestGitLabProjectFullPath(t *testing.T) {
	tests := map[string]string{
		"group/project":                         "group/project",
		"group/subgroup/project":                "group/subgroup/project",
		"git@gitlab.example.com:group/proj.git": "group/proj",
		"https://gitlab.com/group/proj.git":     "group/proj",
		"gitlab.example.com/group/proj":         "group/proj",
	}
	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			got, err := gitLabProjectFullPath(input)
			if err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Fatalf("gitLabProjectFullPath(%q) = %q, want %q", input, got, want)
			}
		})
	}
}
