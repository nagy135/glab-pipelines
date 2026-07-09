package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

func fetchPipelines(repo, status string, limit int) ([]pipeline, error) {
	var all []pipeline
	statuses := []string{status}
	if status == "active" {
		statuses = activeStatuses
	}
	if status == "all" {
		statuses = []string{""}
	}
	seen := map[int]bool{}
	for _, st := range statuses {
		query := fmt.Sprintf("order_by=id&sort=desc&per_page=%d", limit)
		if st != "" {
			query = "status=" + st + "&" + query
		}
		out, err := glabAPI(repo, "", "projects/:id/pipelines?"+query)
		if err != nil {
			return nil, err
		}
		var pipelines []pipeline
		if err := json.Unmarshal(out, &pipelines); err != nil {
			return nil, err
		}
		for _, p := range pipelines {
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
	enrichPipelineCommitTitles(repo, all)
	savePipelineCache(repo, status, limit, all)
	return all, nil
}

func enrichPipelineCommitTitles(repo string, pipelines []pipeline) {
	titles := map[string]string{}
	for i := range pipelines {
		if pipelines[i].Commit.Title != "" {
			pipelines[i].CommitTitle = pipelines[i].Commit.Title
			continue
		}
		if pipelines[i].CommitTitle != "" {
			pipelines[i].Commit.Title = pipelines[i].CommitTitle
			continue
		}
		sha := pipelines[i].SHA
		if sha == "" {
			continue
		}
		if title, ok := titles[sha]; ok {
			pipelines[i].CommitTitle = title
			continue
		}
		out, err := glabAPI(repo, "", fmt.Sprintf("projects/:id/repository/commits/%s", sha))
		if err != nil {
			continue
		}
		var commit commitInfo
		if err := json.Unmarshal(out, &commit); err != nil {
			continue
		}
		titles[sha] = commit.Title
		pipelines[i].Commit.Title = commit.Title
		pipelines[i].CommitTitle = commit.Title
	}
}

func fetchDetail(repo string, pid int) (detail, error) {
	var d detail
	pipelineJSON, err := glabAPI(repo, "", fmt.Sprintf("projects/:id/pipelines/%d", pid))
	if err != nil {
		return d, err
	}
	jobsJSON, err := glabAPI(repo, "", fmt.Sprintf("projects/:id/pipelines/%d/jobs?per_page=100&include_retried=true", pid))
	if err != nil {
		return d, err
	}
	if err := json.Unmarshal(pipelineJSON, &d.Pipeline); err != nil {
		return d, err
	}
	if err := json.Unmarshal(jobsJSON, &d.Jobs); err != nil {
		return d, err
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
		return nil, err
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
	args := []string{"api"}
	if repo != "" {
		args = append(args, "-R", repo)
	}
	if method != "" {
		args = append(args, "--method", method)
	}
	args = append(args, endpoint)
	cmd := exec.Command("glab", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("glab api failed: %s", msg)
	}
	return stdout.Bytes(), nil
}
