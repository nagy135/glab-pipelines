package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	inlineLogLineCount = 5
	inlineLogTailBytes = 64 * 1024
)

func fetchPipelinesCmd(repo, status string, limit int, requestID int) tea.Cmd {
	return func() tea.Msg {
		pipelines, err := fetchPipelines(repo, status, limit)
		return pipelinesMsg{requestID: requestID, pipelines: pipelines, err: err}
	}
}

func fetchDetailCmd(repo string, pid, requestID, pollID int) tea.Cmd {
	return func() tea.Msg {
		d, err := fetchDetail(repo, pid)
		return detailMsg{pid: pid, requestID: requestID, pollID: pollID, detail: d, err: err}
	}
}

func fetchLogsCmd(repo string, j job, requestID, pollID int) tea.Cmd {
	return func() tea.Msg {
		out, err := glabAPI(repo, "", fmt.Sprintf("projects/:id/jobs/%d/trace", j.ID))
		updated, statusErr := fetchJob(repo, j.ID)
		var updatedPtr *job
		if statusErr == nil {
			updatedPtr = &updated
		}
		return logsMsg{jobID: j.ID, requestID: requestID, pollID: pollID, logs: sanitizeTerminalText(string(out)), job: updatedPtr, err: err, statusErr: statusErr}
	}
}

func tickDetailCmd(pid, pollID int, refresh time.Duration) tea.Cmd {
	return tea.Tick(refresh, func(time.Time) tea.Msg { return tickMsg{pid: pid, pollID: pollID} })
}

func tickLogsCmd(jobID int64, pollID int, refresh time.Duration, force bool) tea.Cmd {
	return tea.Tick(refresh, func(time.Time) tea.Msg {
		return logTickMsg{jobID: jobID, pollID: pollID, force: force}
	})
}

func fetchInlineLogCmd(repo string, j job, requestID, pollID int) tea.Cmd {
	return func() tea.Msg {
		endpoint := fmt.Sprintf("projects/:id/jobs/%d/trace", j.ID)
		header := fmt.Sprintf("Range: bytes=-%d", inlineLogTailBytes)
		out, err := glabAPIWithHeaders(repo, "", endpoint, header)
		return inlineLogMsg{
			jobID:     j.ID,
			requestID: requestID,
			pollID:    pollID,
			status:    j.Status,
			lines:     latestLogLines(sanitizeTerminalText(string(out)), inlineLogLineCount),
			err:       err,
		}
	}
}

func tickInlineLogsCmd(pollID int, refresh time.Duration) tea.Cmd {
	return tea.Tick(refresh, func(time.Time) tea.Msg { return inlineLogTickMsg{pollID: pollID} })
}

func runActionCmd(repo string, action pendingAction, requestID int) tea.Cmd {
	return func() tea.Msg {
		endpoint := fmt.Sprintf("projects/:id/jobs/%d/%s", action.Job.ID, action.Endpoint)
		_, err := glabAPI(repo, "POST", endpoint)
		return actionMsg{requestID: requestID, action: action, err: err}
	}
}

func (m model) requestDetail(pid int, restartPolling bool) (model, tea.Cmd) {
	if m.detailRequests == nil {
		m.detailRequests = make(map[int]int)
	}
	if m.detailPolls == nil {
		m.detailPolls = make(map[int]int)
	}
	if restartPolling || m.detailPolls[pid] == 0 {
		m.detailPolls[pid]++
	}
	m.nextRequestID++
	m.detailRequests[pid] = m.nextRequestID
	return m, fetchDetailCmd(m.repo, pid, m.nextRequestID, m.detailPolls[pid])
}

func (m model) requestLogs(j job, restartPolling bool) (model, tea.Cmd) {
	if m.logRequests == nil {
		m.logRequests = make(map[int64]int)
	}
	if m.logPolls == nil {
		m.logPolls = make(map[int64]int)
	}
	if restartPolling || m.logPolls[j.ID] == 0 {
		m.logPolls[j.ID]++
	}
	if restartPolling && m.logFailures != nil {
		delete(m.logFailures, j.ID)
	}
	m.nextRequestID++
	m.logRequests[j.ID] = m.nextRequestID
	return m, fetchLogsCmd(m.repo, j, m.nextRequestID, m.logPolls[j.ID])
}

func (m model) requestInlineLogs(rows []uiJob) (model, tea.Cmd) {
	if m.inlineLogRequests == nil {
		m.inlineLogRequests = make(map[int64]int)
	}
	if m.inlineLogsLoading == nil {
		m.inlineLogsLoading = make(map[int64]bool)
	}
	var cmds []tea.Cmd
	for _, row := range rows {
		j := row.Current
		if !supportsInlineLogs(j) || m.inlineLogsLoading[j.ID] {
			continue
		}
		if snippet, ok := m.inlineLogs[j.ID]; j.Status == "success" && ok && snippet.Status == "success" {
			continue
		}
		m.nextRequestID++
		m.inlineLogRequests[j.ID] = m.nextRequestID
		m.inlineLogsLoading[j.ID] = true
		cmds = append(cmds, fetchInlineLogCmd(m.repo, j, m.nextRequestID, m.inlineLogPollID))
	}
	return m, tea.Batch(cmds...)
}

func latestLogLines(logs string, count int) []string {
	logs = strings.TrimRight(logs, "\r\n")
	if logs == "" || count <= 0 {
		return nil
	}
	lines := strings.Split(logs, "\n")
	if len(lines) > count {
		lines = lines[len(lines)-count:]
	}
	return lines
}
