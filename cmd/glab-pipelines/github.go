package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

const ghAPITimeout = 60 * time.Second

type githubWorkflowRunsResponse struct {
	WorkflowRuns []githubWorkflowRun `json:"workflow_runs"`
}

type githubWorkflowRun struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	DisplayTitle string `json:"display_title"`
	HeadBranch   string `json:"head_branch"`
	HeadSHA      string `json:"head_sha"`
	Status       string `json:"status"`
	Conclusion   string `json:"conclusion"`
	Event        string `json:"event"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	RunStartedAt string `json:"run_started_at"`
	HTMLURL      string `json:"html_url"`
	Path         string `json:"path"`
	HeadCommit   struct {
		Author struct {
			Name string `json:"name"`
		} `json:"author"`
	} `json:"head_commit"`
}

type githubJobsResponse struct {
	Jobs []githubJob `json:"jobs"`
}

type githubJob struct {
	ID           int64  `json:"id"`
	RunID        int    `json:"run_id"`
	Name         string `json:"name"`
	HTMLURL      string `json:"html_url"`
	WorkflowName string `json:"workflow_name"`
	HeadSHA      string `json:"head_sha"`
	HeadBranch   string `json:"head_branch"`
	Status       string `json:"status"`
	Conclusion   string `json:"conclusion"`
	StartedAt    string `json:"started_at"`
	CompletedAt  string `json:"completed_at"`
}

func fetchGitHubPipelines(repo, status string, limit int) ([]pipeline, error) {
	statuses := githubAPIStatuses(status)
	results := make(chan pipelineFetchResult, len(statuses))
	for _, apiStatus := range statuses {
		go func() {
			pipelines, err := fetchGitHubPipelinesByStatus(repo, apiStatus, limit)
			results <- pipelineFetchResult{pipelines: pipelines, err: err}
		}()
	}

	all := make([]pipeline, 0, limit)
	seen := make(map[int]bool)
	for range statuses {
		result := <-results
		if result.err != nil {
			return nil, result.err
		}
		for _, p := range result.pipelines {
			if seen[p.ID] {
				continue
			}
			seen[p.ID] = true
			all = append(all, p)
		}
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].CreatedAt == all[j].CreatedAt {
			return all[i].ID > all[j].ID
		}
		return all[i].CreatedAt > all[j].CreatedAt
	})
	if len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

func githubAPIStatuses(status string) []string {
	switch strings.ToLower(status) {
	case "active":
		return []string{"in_progress", "queued", "requested", "waiting", "pending"}
	case "all", "":
		return []string{""}
	case "running":
		return []string{"in_progress"}
	case "created", "preparing", "scheduled":
		return []string{"queued"}
	case "canceled":
		return []string{"cancelled"}
	case "failed":
		return []string{"failure"}
	default:
		return []string{status}
	}
}

func fetchGitHubPipelinesByStatus(repo, status string, limit int) ([]pipeline, error) {
	var all []pipeline
	pageSize := min(limit, 100)
	for page, fetched := 1, 0; fetched < limit; page++ {
		query := url.Values{
			"per_page": {fmt.Sprint(pageSize)},
			"page":     {fmt.Sprint(page)},
		}
		if status != "" {
			query.Set("status", status)
		}
		out, err := githubAPI(repo, "", "repos/{owner}/{repo}/actions/runs?"+query.Encode())
		if err != nil {
			return nil, err
		}
		var response githubWorkflowRunsResponse
		if err := json.Unmarshal(out, &response); err != nil {
			return nil, fmt.Errorf("decode GitHub Actions runs page %d: %w", page, err)
		}
		for _, run := range response.WorkflowRuns {
			all = append(all, githubRunToPipeline(run))
		}
		fetched += len(response.WorkflowRuns)
		if len(response.WorkflowRuns) < pageSize {
			break
		}
	}
	return all, nil
}

func fetchGitHubPipeline(repo string, pid int) (pipeline, error) {
	out, err := githubAPI(repo, "", fmt.Sprintf("repos/{owner}/{repo}/actions/runs/%d", pid))
	if err != nil {
		return pipeline{}, err
	}
	var run githubWorkflowRun
	if err := json.Unmarshal(out, &run); err != nil {
		return pipeline{}, fmt.Errorf("decode GitHub Actions run %d: %w", pid, err)
	}
	return githubRunToPipeline(run), nil
}

func fetchGitHubDetail(repo string, pid int) (detail, error) {
	p, err := fetchGitHubPipeline(repo, pid)
	if err != nil {
		return detail{}, err
	}
	d := detail{Pipeline: p}
	for page := 1; ; page++ {
		query := url.Values{
			"filter":   {"all"},
			"per_page": {"100"},
			"page":     {fmt.Sprint(page)},
		}
		out, err := githubAPI(repo, "", fmt.Sprintf("repos/{owner}/{repo}/actions/runs/%d/jobs?%s", pid, query.Encode()))
		if err != nil {
			return detail{}, err
		}
		var response githubJobsResponse
		if err := json.Unmarshal(out, &response); err != nil {
			return detail{}, fmt.Errorf("decode jobs for GitHub Actions run %d page %d: %w", pid, page, err)
		}
		for _, source := range response.Jobs {
			d.Jobs = append(d.Jobs, githubJobToJob(source, p))
		}
		if len(response.Jobs) < 100 {
			break
		}
	}
	sort.SliceStable(d.Jobs, func(i, j int) bool { return d.Jobs[i].ID < d.Jobs[j].ID })
	d.DisplayJobs = buildDisplayJobs(d.Jobs)
	return d, nil
}

func fetchGitHubJob(repo string, jobID int64) (job, error) {
	out, err := githubAPI(repo, "", fmt.Sprintf("repos/{owner}/{repo}/actions/jobs/%d", jobID))
	if err != nil {
		return job{}, err
	}
	var source githubJob
	if err := json.Unmarshal(out, &source); err != nil {
		return job{}, fmt.Errorf("decode GitHub Actions job %d: %w", jobID, err)
	}
	p := pipeline{ID: source.RunID, Ref: source.HeadBranch, SHA: source.HeadSHA}
	return githubJobToJob(source, p), nil
}

func githubRunToPipeline(run githubWorkflowRun) pipeline {
	title := run.DisplayTitle
	if title == "" {
		title = run.Name
	}
	p := pipeline{
		ID:        run.ID,
		Status:    githubStatus(run.Status, run.Conclusion),
		Ref:       run.HeadBranch,
		SHA:       run.HeadSHA,
		Source:    run.Event,
		UpdatedAt: run.UpdatedAt,
		CreatedAt: run.CreatedAt,
		StartedAt: run.RunStartedAt,
		WebURL:    run.HTMLURL,
		Commit: commitInfo{
			Title:      title,
			AuthorName: run.HeadCommit.Author.Name,
		},
		CommitTitle:  title,
		WorkflowPath: run.Path,
	}
	p.Duration = elapsedDuration(run.RunStartedAt, completedAt(run.Status, run.UpdatedAt))
	sanitizePipeline(&p)
	return p
}

func githubJobToJob(source githubJob, p pipeline) job {
	stage := source.WorkflowName
	if stage == "" {
		stage = "jobs"
	}
	if p.ID == 0 {
		p.ID = source.RunID
	}
	if p.Ref == "" {
		p.Ref = source.HeadBranch
	}
	if p.SHA == "" {
		p.SHA = source.HeadSHA
	}
	j := job{
		ID:         source.ID,
		Name:       source.Name,
		WebURL:     source.HTMLURL,
		Status:     githubStatus(source.Status, source.Conclusion),
		Stage:      stage,
		Ref:        source.HeadBranch,
		StartedAt:  source.StartedAt,
		FinishedAt: source.CompletedAt,
		Duration:   elapsedDuration(source.StartedAt, source.CompletedAt),
		Pipeline:   p,
	}
	sanitizeJob(&j)
	return j
}

func githubStatus(status, conclusion string) string {
	if status != "completed" && conclusion == "" {
		switch status {
		case "in_progress":
			return "running"
		case "queued", "requested", "waiting", "pending":
			return "pending"
		default:
			return status
		}
	}
	switch conclusion {
	case "success":
		return "success"
	case "cancelled":
		return "canceled"
	case "failure", "timed_out", "action_required", "startup_failure", "stale":
		return "failed"
	case "neutral", "skipped":
		return conclusion
	case "":
		return status
	default:
		return conclusion
	}
}

func completedAt(status, updatedAt string) string {
	if status == "completed" {
		return updatedAt
	}
	return ""
}

func elapsedDuration(start, finish string) *float64 {
	if start == "" || finish == "" {
		return nil
	}
	started, err := time.Parse(time.RFC3339Nano, start)
	if err != nil {
		return nil
	}
	finished, err := time.Parse(time.RFC3339Nano, finish)
	if err != nil || finished.Before(started) {
		return nil
	}
	seconds := finished.Sub(started).Seconds()
	return &seconds
}

func fetchGitHubJobCode(repo string, j job) (string, error) {
	p := j.Pipeline
	if p.WorkflowPath == "" || p.SHA == "" {
		var err error
		p, err = fetchGitHubPipeline(repo, p.ID)
		if err != nil {
			return "", err
		}
	}
	if p.WorkflowPath == "" {
		return "", fmt.Errorf("GitHub Actions run %d has no workflow path", p.ID)
	}
	query := url.Values{"ref": {p.SHA}}
	endpoint := fmt.Sprintf("repos/{owner}/{repo}/contents/%s?%s", escapeGitHubPath(p.WorkflowPath), query.Encode())
	out, err := githubAPIWithHeaders(repo, "", endpoint, "Accept: application/vnd.github.raw+json")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("# workflow: %s\n# selected job: %s\n\n%s", p.WorkflowPath, j.Name, strings.TrimSpace(string(out))), nil
}

func escapeGitHubPath(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}

func runGitHubAction(repo string, action pendingAction) error {
	var endpoint string
	switch action.Endpoint {
	case "rerun":
		endpoint = fmt.Sprintf("repos/{owner}/{repo}/actions/jobs/%d/rerun", action.Job.ID)
	case "cancel":
		endpoint = fmt.Sprintf("repos/{owner}/{repo}/actions/runs/%d/cancel", action.PipelineID)
	default:
		return fmt.Errorf("unsupported GitHub Actions operation %q", action.Endpoint)
	}
	_, err := githubAPI(repo, "POST", endpoint)
	return err
}

func githubAPI(repo, method, endpoint string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ghAPITimeout)
	defer cancel()
	return githubAPIContext(ctx, repo, method, endpoint)
}

func githubAPIWithHeaders(repo, method, endpoint string, headers ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ghAPITimeout)
	defer cancel()
	return githubAPIContext(ctx, repo, method, endpoint, headers...)
}

func githubAPIContext(ctx context.Context, repo, method, endpoint string, headers ...string) ([]byte, error) {
	args := []string{"api"}
	if method != "" {
		args = append(args, "--method", method)
	}
	for _, header := range headers {
		args = append(args, "-H", header)
	}
	args = append(args, endpoint)
	cmd := exec.CommandContext(ctx, "gh", args...)
	if repo != "" {
		cmd.Env = append(os.Environ(), "GH_REPO="+repo)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("gh api %s: %w", endpoint, ctx.Err())
		}
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return nil, fmt.Errorf("gh api %s: %w: %s", endpoint, err, sanitizeTerminalText(msg))
		}
		return nil, fmt.Errorf("gh api %s: %w", endpoint, err)
	}
	return stdout.Bytes(), nil
}
