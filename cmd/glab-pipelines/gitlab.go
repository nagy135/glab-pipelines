package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const glabAPITimeout = 60 * time.Second
const gitLabGraphQLPageSize = 100

// Keep metadata lookups concurrent without creating an unbounded burst of API calls.
const pipelineMetadataWorkers = 4

func fetchPipelines(repo, status string, limit int) ([]pipeline, error) {
	if limit <= 0 {
		return nil, nil
	}

	var graphQLErr error
	if limit <= gitLabGraphQLPageSize {
		pipelines, err := fetchPipelinesGraphQL(repo, status, limit)
		if err == nil {
			return pipelines, nil
		}
		graphQLErr = err
	}

	pipelines, err := fetchPipelinesREST(repo, status, limit)
	if err != nil && graphQLErr != nil {
		return nil, fmt.Errorf("GitLab GraphQL pipeline list failed: %v; REST fallback failed: %w", graphQLErr, err)
	}
	return pipelines, err
}

func fetchPipelinesGraphQL(repo, status string, limit int) ([]pipeline, error) {
	query, aliases, err := buildPipelineListGraphQLQuery(status)
	if err != nil {
		return nil, err
	}
	out, err := glabGraphQL(repo, query, limit)
	if err != nil {
		return nil, err
	}

	var response gitLabGraphQLResponse
	if err := json.Unmarshal(out, &response); err != nil {
		return nil, fmt.Errorf("decode GitLab GraphQL pipeline list: %w", err)
	}
	if len(response.Errors) > 0 {
		messages := make([]string, 0, len(response.Errors))
		for _, graphQLError := range response.Errors {
			messages = append(messages, sanitizeTerminalText(graphQLError.Message))
		}
		return nil, fmt.Errorf("GitLab GraphQL pipeline list: %s", strings.Join(messages, "; "))
	}
	if response.Data.Project == nil {
		return nil, gitLabGraphQLProjectNotFoundError(repo)
	}

	all := make([]pipeline, 0, limit)
	seen := make(map[int]bool, limit)
	for _, alias := range aliases {
		connection, ok := response.Data.Project[alias]
		if !ok {
			return nil, fmt.Errorf("GitLab GraphQL pipeline list omitted %s", alias)
		}
		for _, source := range connection.Nodes {
			p, err := graphQLPipelineToPipeline(source)
			if err != nil {
				return nil, err
			}
			if seen[p.ID] {
				continue
			}
			seen[p.ID] = true
			all = append(all, p)
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID > all[j].ID })
	if len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

func fetchPipelinesREST(repo, status string, limit int) ([]pipeline, error) {
	statuses := pipelineStatusFilters(status)
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

func pipelineStatusFilters(status string) []string {
	switch status {
	case "active":
		return activeStatuses
	case "all":
		return []string{""}
	default:
		return []string{status}
	}
}

type gitLabGraphQLResponse struct {
	Data struct {
		Project map[string]gitLabPipelineConnection `json:"project"`
	} `json:"data"`
	Errors []gitLabGraphQLError `json:"errors"`
}

type gitLabGraphQLError struct {
	Message string `json:"message"`
}

type gitLabPipelineConnection struct {
	Nodes []gitLabGraphQLPipeline `json:"nodes"`
}

type gitLabGraphQLPipeline struct {
	ID        string     `json:"id"`
	Status    string     `json:"status"`
	Ref       string     `json:"ref"`
	SHA       string     `json:"sha"`
	Source    string     `json:"source"`
	UpdatedAt string     `json:"updatedAt"`
	CreatedAt string     `json:"createdAt"`
	StartedAt string     `json:"startedAt"`
	Duration  *float64   `json:"duration"`
	Commit    commitInfo `json:"commit"`
}

func buildPipelineListGraphQLQuery(status string) (string, []string, error) {
	statuses := pipelineStatusFilters(status)
	aliases := make([]string, 0, len(statuses))
	var query strings.Builder
	query.WriteString("query PipelineList($fullPath: ID!, $limit: Int!) {\n  project(fullPath: $fullPath) {\n")
	for i, pipelineStatus := range statuses {
		alias := fmt.Sprintf("pipelines%d", i)
		aliases = append(aliases, alias)
		if pipelineStatus == "" {
			fmt.Fprintf(&query, "    %s: pipelines(first: $limit) { nodes { ...PipelineListFields } }\n", alias)
			continue
		}
		statusEnum, ok := gitLabPipelineStatusEnum(pipelineStatus)
		if !ok {
			return "", nil, fmt.Errorf("unsupported GitLab pipeline status %q", pipelineStatus)
		}
		fmt.Fprintf(&query, "    %s: pipelines(first: $limit, status: %s) { nodes { ...PipelineListFields } }\n", alias, statusEnum)
	}
	query.WriteString("  }\n}\nfragment PipelineListFields on Pipeline {\n  id\n  status\n  ref\n  sha\n  source\n  updatedAt\n  createdAt\n  startedAt\n  duration\n  commit { title }\n}\n")
	return query.String(), aliases, nil
}

func gitLabPipelineStatusEnum(status string) (string, bool) {
	switch status {
	case "created", "waiting_for_resource", "preparing", "waiting_for_callback", "pending", "running", "success", "failed", "canceling", "canceled", "skipped", "manual", "scheduled":
		return strings.ToUpper(status), true
	default:
		return "", false
	}
}

func graphQLPipelineToPipeline(source gitLabGraphQLPipeline) (pipeline, error) {
	id, err := graphQLPipelineID(source.ID)
	if err != nil {
		return pipeline{}, err
	}
	p := pipeline{
		ID:          id,
		Status:      strings.ToLower(source.Status),
		Ref:         source.Ref,
		SHA:         source.SHA,
		Source:      source.Source,
		UpdatedAt:   source.UpdatedAt,
		CreatedAt:   source.CreatedAt,
		StartedAt:   source.StartedAt,
		Duration:    source.Duration,
		Commit:      source.Commit,
		CommitTitle: source.Commit.Title,
	}
	sanitizePipeline(&p)
	return p, nil
}

func graphQLPipelineID(globalID string) (int, error) {
	value := globalID
	if slash := strings.LastIndex(value, "/"); slash >= 0 {
		value = value[slash+1:]
	}
	id, err := strconv.Atoi(value)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("decode GitLab GraphQL pipeline ID %q", sanitizeTerminalText(globalID))
	}
	return id, nil
}

func gitLabGraphQLProjectNotFoundError(repo string) error {
	if repo == "" {
		return fmt.Errorf("GitLab GraphQL project was not found for the current repository")
	}
	return fmt.Errorf("GitLab GraphQL project %q was not found", sanitizeTerminalText(repo))
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

func glabGraphQL(repo, query string, limit int) ([]byte, error) {
	fullPath, err := gitLabProjectFullPath(repo)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), glabAPITimeout)
	defer cancel()

	args := []string{"api"}
	if repo != "" {
		args = append(args, "-R", repo)
	}
	args = append(args,
		"graphql",
		"-f", "query="+query,
		"-f", "fullPath="+fullPath,
		"-F", fmt.Sprintf("limit=%d", limit),
	)
	cmd := exec.CommandContext(ctx, "glab", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("glab api graphql: %w", ctx.Err())
		}
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return nil, fmt.Errorf("glab api graphql: %w: %s", err, sanitizeTerminalText(msg))
		}
		return nil, fmt.Errorf("glab api graphql: %w", err)
	}
	return stdout.Bytes(), nil
}

func gitLabProjectFullPath(repo string) (string, error) {
	value := strings.TrimSpace(repo)
	if value == "" {
		out, err := exec.Command("git", "remote", "get-url", "origin").Output()
		if err != nil {
			return "", fmt.Errorf("resolve GitLab project path: %w", err)
		}
		value = strings.TrimSpace(string(out))
	}

	if parsed, err := url.Parse(value); err == nil && parsed.Hostname() != "" {
		value = parsed.Path
	} else {
		if at := strings.LastIndex(value, "@"); at >= 0 {
			value = value[at+1:]
		}
		if colon := strings.Index(value, ":"); colon >= 0 {
			value = value[colon+1:]
		}
	}

	value = strings.Trim(strings.TrimSuffix(value, ".git"), "/")
	parts := strings.Split(value, "/")
	if len(parts) >= 3 && strings.Contains(parts[0], ".") {
		parts = parts[1:]
	}
	value = strings.Join(parts, "/")
	if !strings.Contains(value, "/") {
		return "", fmt.Errorf("cannot determine GitLab project path from %q", sanitizeTerminalText(repo))
	}
	return value, nil
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
