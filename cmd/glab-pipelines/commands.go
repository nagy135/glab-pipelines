package main

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func fetchPipelinesCmd(repo, status string, limit int, requestID int) tea.Cmd {
	return func() tea.Msg {
		pipelines, err := fetchPipelines(repo, status, limit)
		return pipelinesMsg{requestID: requestID, pipelines: pipelines, err: err}
	}
}

func fetchDetailCmd(repo string, pid int) tea.Cmd {
	return func() tea.Msg {
		d, err := fetchDetail(repo, pid)
		return detailMsg{pid: pid, detail: d, err: err}
	}
}

func fetchLogsCmd(repo string, job job) tea.Cmd {
	return func() tea.Msg {
		out, err := glabAPI(repo, "", fmt.Sprintf("projects/:id/jobs/%d/trace", job.ID))
		return logsMsg{jobID: job.ID, logs: string(out), err: err}
	}
}

func tickDetailCmd(pid int, refresh time.Duration) tea.Cmd {
	return tea.Tick(refresh, func(time.Time) tea.Msg { return tickMsg{pid: pid} })
}

func tickLogsCmd(jobID int64, refresh time.Duration) tea.Cmd {
	return tea.Tick(refresh, func(time.Time) tea.Msg { return logTickMsg{jobID: jobID} })
}

func runActionCmd(repo string, action pendingAction) tea.Cmd {
	return func() tea.Msg {
		endpoint := fmt.Sprintf("projects/:id/jobs/%d/%s", action.Job.ID, action.Endpoint)
		_, err := glabAPI(repo, "POST", endpoint)
		return actionMsg{action: action, err: err}
	}
}
