package main

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	inlineLogLineCount = 5
	inlineLogTailBytes = 64 * 1024
)

func fetchPipelinesCmd(provider ciProvider, repo, status string, limit int, requestID int) tea.Cmd {
	return func() tea.Msg {
		pipelines, err := fetchProviderPipelines(provider, repo, status, limit)
		return pipelinesMsg{requestID: requestID, pipelines: pipelines, err: err}
	}
}

func fetchDetailCmd(provider ciProvider, repo string, selected pipeline, requestID, pollID int) tea.Cmd {
	return func() tea.Msg {
		d, err := fetchProviderDetail(provider, repo, selected)
		return detailMsg{pid: selected.ID, requestID: requestID, pollID: pollID, detail: d, err: err}
	}
}

func fetchLogsCmd(provider ciProvider, repo string, j job, requestID, pollID int) tea.Cmd {
	return func() tea.Msg {
		out, err := fetchProviderLogs(provider, repo, j.ID, false)
		updated, statusErr := fetchProviderJob(provider, repo, j.ID)
		var updatedPtr *job
		if statusErr == nil {
			updatedPtr = &updated
		}
		return logsMsg{jobID: j.ID, requestID: requestID, pollID: pollID, logs: sanitizeTerminalText(string(out)), job: updatedPtr, err: err, statusErr: statusErr}
	}
}

func fetchJobCodeCmd(provider ciProvider, repo string, j job, requestID int) tea.Cmd {
	return func() tea.Msg {
		code, err := fetchProviderJobCode(provider, repo, j)
		return codeMsg{jobID: j.ID, requestID: requestID, code: sanitizeTerminalText(code), err: err}
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

func fetchInlineLogCmd(provider ciProvider, repo string, j job, requestID, pollID int) tea.Cmd {
	return func() tea.Msg {
		out, err := fetchProviderLogs(provider, repo, j.ID, true)
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

func runActionCmd(provider ciProvider, repo string, action pendingAction, requestID int) tea.Cmd {
	return func() tea.Msg {
		updatedPipeline, err := runProviderAction(provider, repo, action)
		return actionMsg{requestID: requestID, action: action, pipeline: updatedPipeline, err: err}
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
	selected := pipeline{ID: pid}
	for _, p := range m.list {
		if p.ID == pid {
			selected = p
			break
		}
	}
	if selected.IID == 0 && m.detail != nil && m.detail.Pipeline.ID == pid {
		selected = m.detail.Pipeline
	}
	return m, fetchDetailCmd(m.provider, m.repo, selected, m.nextRequestID, m.detailPolls[pid])
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
	return m, fetchLogsCmd(m.provider, m.repo, j, m.nextRequestID, m.logPolls[j.ID])
}

func (m model) requestJobCode(j job) (model, tea.Cmd) {
	if m.codeRequests == nil {
		m.codeRequests = make(map[int64]int)
	}
	m.nextRequestID++
	m.codeRequests[j.ID] = m.nextRequestID
	return m, fetchJobCodeCmd(m.provider, m.repo, j, m.nextRequestID)
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
		if snippet, ok := m.inlineLogs[j.ID]; j.Status == "failed" && ok && snippet.Status == "failed" {
			continue
		}
		m.nextRequestID++
		m.inlineLogRequests[j.ID] = m.nextRequestID
		m.inlineLogsLoading[j.ID] = true
		cmds = append(cmds, fetchInlineLogCmd(m.provider, m.repo, j, m.nextRequestID, m.inlineLogPollID))
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
