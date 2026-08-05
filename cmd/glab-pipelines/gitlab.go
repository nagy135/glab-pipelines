package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
)

const glabAPITimeout = 60 * time.Second

// Keep metadata lookups concurrent without creating an unbounded burst of API calls.
const pipelineMetadataWorkers = 4

func fetchPipelines(repo, status string, limit int) ([]pipeline, error) {
	statuses := []string{status}
	if status == "active" {
		statuses = activeStatuses
	}
	if status == "all" {
		statuses = []string{""}
	}
	results := make(chan pipelineFetchResult, len(statuses))
	for _, status := range statuses {
		go func(status string) {
			pipelines, err := fetchPipelinesByStatus(repo, status, limit)
			results <- pipelineFetchResult{pipelines: pipelines, err: err}
		}(status)
	}

	var all []pipeline
	seen := map[int]bool{}
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
		return all[i].ID > all[j].ID
	})
	if len(all) > limit {
		all = all[:limit]
	}
	enrichPipelineMetadata(repo, all)
	return all, nil
}

type pipelineFetchResult struct {
	pipelines []pipeline
	err       error
}

func fetchPipelinesByStatus(repo, status string, limit int) ([]pipeline, error) {
	var all []pipeline
	pageSize := min(limit, 100)
	for page, fetched := 1, 0; fetched < limit; page++ {
		query := url.Values{
			"order_by": {"id"},
			"sort":     {"desc"},
			"per_page": {fmt.Sprint(pageSize)},
			"page":     {fmt.Sprint(page)},
		}
		if status != "" {
			query.Set("status", status)
		}
		out, err := glabAPI(repo, "", "projects/:id/pipelines?"+query.Encode())
		if err != nil {
			return nil, err
		}
		var pipelines []pipeline
		if err := json.Unmarshal(out, &pipelines); err != nil {
			return nil, fmt.Errorf("decode pipelines page %d: %w", page, err)
		}
		for i := range pipelines {
			sanitizePipeline(&pipelines[i])
		}
		all = append(all, pipelines...)
		fetched += len(pipelines)
		if len(pipelines) < pageSize {
			break
		}
	}
	return all, nil
}

func enrichPipelineMetadata(repo string, pipelines []pipeline) {
	workers := min(pipelineMetadataWorkers, len(pipelines))
	if workers == 0 {
		return
	}

	indexes := make(chan int)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range indexes {
				enrichPipelineMetadataItem(repo, &pipelines[i])
			}
		}()
	}
	for i := range pipelines {
		indexes <- i
	}
	close(indexes)
	wg.Wait()
}

func enrichPipelineMetadataItem(repo string, p *pipeline) {
	if p.Duration == nil || p.StartedAt == "" {
		if detail, err := fetchPipeline(repo, p.ID); err == nil {
			if p.Duration == nil {
				p.Duration = detail.Duration
			}
			p.StartedAt = detail.StartedAt
			if p.Commit.Title == "" {
				p.Commit.Title = detail.Commit.Title
			}
			if p.CommitTitle == "" {
				p.CommitTitle = detail.CommitTitle
			}
		}
	}
	if p.Commit.Title != "" {
		p.CommitTitle = p.Commit.Title
		return
	}
	if p.CommitTitle != "" {
		p.Commit.Title = p.CommitTitle
		return
	}
	if p.SHA == "" {
		return
	}
	out, err := glabAPI(repo, "", fmt.Sprintf("projects/:id/repository/commits/%s", p.SHA))
	if err != nil {
		return
	}
	var commit commitInfo
	if err := json.Unmarshal(out, &commit); err != nil {
		return
	}
	commit.Title = sanitizeTerminalText(commit.Title)
	p.Commit.Title = commit.Title
	p.CommitTitle = commit.Title
}

func fetchPipeline(repo string, pid int) (pipeline, error) {
	out, err := glabAPI(repo, "", fmt.Sprintf("projects/:id/pipelines/%d", pid))
	if err != nil {
		return pipeline{}, err
	}
	var p pipeline
	if err := json.Unmarshal(out, &p); err != nil {
		return pipeline{}, fmt.Errorf("decode pipeline %d: %w", pid, err)
	}
	sanitizePipeline(&p)
	return p, nil
}

func fetchJob(repo string, jobID int64) (job, error) {
	out, err := glabAPI(repo, "", fmt.Sprintf("projects/:id/jobs/%d", jobID))
	if err != nil {
		return job{}, err
	}
	var j job
	if err := json.Unmarshal(out, &j); err != nil {
		return job{}, fmt.Errorf("decode job %d: %w", jobID, err)
	}
	sanitizeJob(&j)
	return j, nil
}

func fetchDetail(repo string, pid int) (detail, error) {
	var d detail
	p, err := fetchPipeline(repo, pid)
	if err != nil {
		return d, err
	}
	d.Pipeline = p
	for page := 1; ; page++ {
		query := url.Values{
			"include_retried": {"true"},
			"per_page":        {"100"},
			"page":            {fmt.Sprint(page)},
		}
		jobsJSON, err := glabAPI(repo, "", fmt.Sprintf("projects/:id/pipelines/%d/jobs?%s", pid, query.Encode()))
		if err != nil {
			return d, err
		}
		var jobs []job
		if err := json.Unmarshal(jobsJSON, &jobs); err != nil {
			return d, fmt.Errorf("decode jobs for pipeline %d page %d: %w", pid, page, err)
		}
		for i := range jobs {
			sanitizeJob(&jobs[i])
		}
		d.Jobs = append(d.Jobs, jobs...)
		if len(jobs) < 100 {
			break
		}
	}
	sort.SliceStable(d.Jobs, func(i, j int) bool { return d.Jobs[i].ID < d.Jobs[j].ID })
	d.DisplayJobs = buildDisplayJobs(d.Jobs)
	if history, err := fetchProjectJobHistory(repo, d.Pipeline); err == nil {
		augmentPreviousRuns(d.DisplayJobs, history, d.Pipeline)
	}
	return d, nil
}

func fetchProjectJobHistory(repo string, p pipeline) ([]job, error) {
	endpoint := "projects/:id/jobs?per_page=100&order_by=id&sort=desc"
	out, err := glabAPI(repo, "", endpoint)
	if err != nil {
		return nil, err
	}
	var jobs []job
	if err := json.Unmarshal(out, &jobs); err != nil {
		return nil, fmt.Errorf("decode project job history: %w", err)
	}
	for i := range jobs {
		sanitizeJob(&jobs[i])
	}
	filtered := jobs[:0]
	for _, j := range jobs {
		if samePipelineJob(j, p) {
			filtered = append(filtered, j)
		}
	}
	return filtered, nil
}

func samePipelineJob(j job, p pipeline) bool {
	if j.Pipeline.ID == p.ID {
		return true
	}
	if j.Ref != "" && p.Ref != "" && j.Ref != p.Ref {
		return false
	}
	if j.Pipeline.SHA != "" && p.SHA != "" && j.Pipeline.SHA == p.SHA {
		return true
	}
	return j.Ref != "" && p.Ref != "" && j.Ref == p.Ref && j.Pipeline.ID == 0
}

func glabAPI(repo, method, endpoint string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), glabAPITimeout)
	defer cancel()
	return glabAPIContext(ctx, repo, method, endpoint)
}

func glabAPIWithHeaders(repo, method, endpoint string, headers ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), glabAPITimeout)
	defer cancel()
	return glabAPIContext(ctx, repo, method, endpoint, headers...)
}

func glabAPIContext(ctx context.Context, repo, method, endpoint string, headers ...string) ([]byte, error) {
	args := []string{"api"}
	if repo != "" {
		args = append(args, "-R", repo)
	}
	if method != "" {
		args = append(args, "--method", method)
	}
	for _, header := range headers {
		args = append(args, "-H", header)
	}
	args = append(args, endpoint)
	cmd := exec.CommandContext(ctx, "glab", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("glab api %s: %w", endpoint, ctx.Err())
		}
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return nil, fmt.Errorf("glab api %s: %w: %s", endpoint, err, sanitizeTerminalText(msg))
		}
		return nil, fmt.Errorf("glab api %s: %w", endpoint, err)
	}
	return stdout.Bytes(), nil
}

func sanitizePipeline(p *pipeline) {
	p.Status = sanitizeTerminalText(p.Status)
	p.Ref = sanitizeTerminalText(p.Ref)
	p.SHA = sanitizeTerminalText(p.SHA)
	p.Source = sanitizeTerminalText(p.Source)
	p.WebURL = sanitizeTerminalText(p.WebURL)
	p.Commit.Title = sanitizeTerminalText(p.Commit.Title)
	p.CommitTitle = sanitizeTerminalText(p.CommitTitle)
	p.WorkflowPath = sanitizeTerminalText(p.WorkflowPath)
	if p.CommitTitle == "" {
		p.CommitTitle = p.Commit.Title
	}
	if p.Commit.Title == "" {
		p.Commit.Title = p.CommitTitle
	}
}

func sanitizeJob(j *job) {
	j.Name = sanitizeTerminalText(j.Name)
	j.Status = sanitizeTerminalText(j.Status)
	j.Stage = sanitizeTerminalText(j.Stage)
	j.Ref = sanitizeTerminalText(j.Ref)
	sanitizePipeline(&j.Pipeline)
}
