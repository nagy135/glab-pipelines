package main

import (
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"
)

func (p ciProvider) name() string {
	if p == providerGitHub {
		return "github"
	}
	return "gitlab"
}

func providerDisplayName(provider ciProvider) string {
	if provider == providerGitHub {
		return "GitHub"
	}
	return "GitLab"
}

func parseProvider(value string) (ciProvider, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "auto":
		return providerGitLab, nil
	case "gitlab", "glab":
		return providerGitLab, nil
	case "github", "gh":
		return providerGitHub, nil
	default:
		return providerGitLab, fmt.Errorf("unknown provider %q (available: auto, gitlab, github)", value)
	}
}

func detectProvider(repo string) ciProvider {
	if provider := providerFromRemoteURL(repo); provider != providerGitLab || looksLikeGitLabRemote(repo) {
		return provider
	}
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return providerGitLab
	}
	return providerFromRemoteURL(strings.TrimSpace(string(out)))
}

func providerFromRemoteURL(remote string) ciProvider {
	host := remoteHost(remote)
	if host == "github.com" || strings.HasSuffix(host, ".github.com") || strings.Contains(host, "github") {
		return providerGitHub
	}
	return providerGitLab
}

func looksLikeGitLabRemote(remote string) bool {
	host := remoteHost(remote)
	return host == "gitlab.com" || strings.HasSuffix(host, ".gitlab.com") || strings.Contains(host, "gitlab")
}

func remoteHost(remote string) string {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return ""
	}
	if parsed, err := url.Parse(remote); err == nil && parsed.Hostname() != "" {
		return strings.ToLower(parsed.Hostname())
	}
	// SCP-style Git URLs, for example git@github.com:owner/repo.git.
	if at := strings.LastIndex(remote, "@"); at >= 0 {
		remote = remote[at+1:]
	}
	if colon := strings.Index(remote, ":"); colon >= 0 {
		return strings.ToLower(remote[:colon])
	}
	parts := strings.Split(filepath.ToSlash(remote), "/")
	if len(parts) >= 3 && strings.Contains(parts[0], ".") {
		return strings.ToLower(parts[0])
	}
	return ""
}

func providerScope(provider ciProvider, repo string) string {
	if provider == providerGitHub {
		return "github:" + repo
	}
	return repo
}

func fetchProviderPipelines(provider ciProvider, repo, status string, limit int) ([]pipeline, error) {
	if provider == providerGitHub {
		return fetchGitHubPipelines(repo, status, limit)
	}
	return fetchPipelines(repo, status, limit)
}

func fetchProviderDetail(provider ciProvider, repo string, pid int) (detail, error) {
	if provider == providerGitHub {
		return fetchGitHubDetail(repo, pid)
	}
	return fetchDetail(repo, pid)
}

func fetchProviderJob(provider ciProvider, repo string, jobID int64) (job, error) {
	if provider == providerGitHub {
		return fetchGitHubJob(repo, jobID)
	}
	return fetchJob(repo, jobID)
}

func fetchProviderLogs(provider ciProvider, repo string, jobID int64, tail bool) ([]byte, error) {
	if provider == providerGitHub {
		return githubAPI(repo, "", fmt.Sprintf("repos/{owner}/{repo}/actions/jobs/%d/logs", jobID))
	}
	endpoint := fmt.Sprintf("projects/:id/jobs/%d/trace", jobID)
	if tail {
		return glabAPIWithHeaders(repo, "", endpoint, fmt.Sprintf("Range: bytes=-%d", inlineLogTailBytes))
	}
	return glabAPI(repo, "", endpoint)
}

func fetchProviderJobCode(provider ciProvider, repo string, j job) (string, error) {
	if provider == providerGitHub {
		return fetchGitHubJobCode(repo, j)
	}
	return fetchJobCode(repo, j)
}

func runProviderAction(provider ciProvider, repo string, action pendingAction) error {
	if provider == providerGitHub {
		return runGitHubAction(repo, action)
	}
	if action.Target == actionTargetPipeline {
		endpoint := fmt.Sprintf("projects/:id/pipelines/%d/cancel", action.PipelineID)
		_, err := glabAPI(repo, "POST", endpoint)
		return err
	}
	endpoint := fmt.Sprintf("projects/:id/jobs/%d/%s", action.Job.ID, action.Endpoint)
	_, err := glabAPI(repo, "POST", endpoint)
	return err
}
