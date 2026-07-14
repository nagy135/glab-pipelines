package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type lintJob struct {
	Name         string   `json:"name"`
	Stage        string   `json:"stage"`
	BeforeScript []string `json:"before_script"`
	Script       []string `json:"script"`
	AfterScript  []string `json:"after_script"`
}

type lintResponse struct {
	Valid  bool      `json:"valid"`
	Errors []string  `json:"errors"`
	Jobs   []lintJob `json:"jobs"`
}

func fetchJobCode(repo string, j job) (string, error) {
	ref := j.Pipeline.SHA
	if ref == "" {
		ref = j.Ref
	}
	if ref == "" {
		return "", fmt.Errorf("job %s has no pipeline commit or ref", j.Name)
	}
	query := url.Values{
		"content_ref":  {ref},
		"include_jobs": {"true"},
	}
	out, err := glabAPI(repo, "", "projects/:id/ci/lint?"+query.Encode())
	if err != nil {
		return "", err
	}
	return decodeJobCode(out, j.Name)
}

func decodeJobCode(data []byte, jobName string) (string, error) {
	var response lintResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return "", fmt.Errorf("decode CI configuration: %w", err)
	}
	if !response.Valid {
		if len(response.Errors) > 0 {
			return "", fmt.Errorf("invalid CI configuration: %s", strings.Join(response.Errors, "; "))
		}
		return "", fmt.Errorf("invalid CI configuration")
	}
	for _, candidate := range response.Jobs {
		if candidate.Name == jobName {
			return formatJobCode(candidate), nil
		}
	}
	return "", fmt.Errorf("job %q was not found in the resolved CI configuration", jobName)
}

func formatJobCode(j lintJob) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# job: %s\n# stage: %s\n", j.Name, j.Stage)
	writeScriptSection(&b, "before_script", j.BeforeScript)
	writeScriptSection(&b, "script", j.Script)
	writeScriptSection(&b, "after_script", j.AfterScript)
	return strings.TrimRight(b.String(), "\n")
}

func writeScriptSection(b *strings.Builder, name string, commands []string) {
	if len(commands) == 0 && name != "script" {
		return
	}
	fmt.Fprintf(b, "\n# %s\n", name)
	if len(commands) == 0 {
		b.WriteString("# (none)\n")
		return
	}
	for _, command := range commands {
		b.WriteString(command)
		b.WriteByte('\n')
	}
}
