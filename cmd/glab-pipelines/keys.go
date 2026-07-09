package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}
	switch m.mode {
	case modePipelines:
		return m.handlePipelineKey(key)
	case modeDetail:
		return m.handleDetailKey(key)
	case modeJobs:
		return m.handleJobsKey(key)
	case modeConfirm:
		return m.handleConfirmKey(key)
	case modeLogs:
		return m.handleLogsKey(key)
	}
	return m, nil
}

func (m model) handlePipelineKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "s":
		return m.splitActiveLogPane(logSplitHorizontal), tea.ClearScreen
	case "v":
		return m.splitActiveLogPane(logSplitVertical), tea.ClearScreen
	case "x":
		return m.closeActiveLogPane(), tea.ClearScreen
	case "ctrl+h":
		return m.focusLogPane("h"), nil
	case "ctrl+j":
		return m.focusLogPane("j"), nil
	case "ctrl+k":
		return m.focusLogPane("k"), nil
	case "ctrl+l":
		return m.focusLogPane("l"), nil
	case "q", "esc":
		return m, tea.Quit
	case "r", "R":
		m.listRequest++
		m.loadingList = true
		m.message = "refreshing pipelines..."
		return m, fetchPipelinesCmd(m.repo, m.status, m.limit, m.listRequest)
	case "up", "k":
		m.listCursor = moveUp(m.listCursor, len(m.list))
		m = m.saveActiveLogPane()
	case "down", "j":
		m.listCursor = moveDown(m.listCursor, len(m.list))
		m = m.saveActiveLogPane()
	case "enter":
		if len(m.list) == 0 {
			return m, nil
		}
		m.detailID = m.list[m.listCursor].ID
		m.detail = nil
		m.detailLoading = true
		m.jobsCursor = 0
		m.message = ""
		m.mode = modeDetail
		if m.logSplitRoot == nil || m.activeLogPane == 0 {
			m = m.initLogPanes()
		} else {
			m = m.saveActiveLogPane()
		}
		return m, tea.Batch(tea.ClearScreen, fetchDetailCmd(m.repo, m.detailID), tickDetailCmd(m.detailID, m.refresh))
	}
	return m, nil
}

func (m model) handleDetailKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "s":
		return m.splitActiveLogPane(logSplitHorizontal), tea.ClearScreen
	case "v":
		return m.splitActiveLogPane(logSplitVertical), tea.ClearScreen
	case "x":
		return m.closeActiveLogPane(), tea.ClearScreen
	case "ctrl+h":
		return m.focusLogPane("h"), nil
	case "ctrl+j":
		return m.focusLogPane("j"), nil
	case "ctrl+k":
		return m.focusLogPane("k"), nil
	case "ctrl+l":
		return m.focusLogPane("l"), nil
	case "q", "esc":
		if len(m.logPanes) > 1 {
			m = m.setActivePaneMode(modePipelines)
			m.detail = nil
			m.detailID = 0
			m.detailLoading = false
			m.jobsCursor = 0
			m.message = ""
			return m.saveActiveLogPane(), tea.ClearScreen
		}
		m.mode = modePipelines
		m.message = ""
		return m, tea.ClearScreen
	case "r":
		m.detailLoading = true
		m = m.saveActiveLogPane()
		return m, fetchDetailCmd(m.repo, m.detailID)
	case "up", "k":
		if m.detail != nil {
			m.jobsCursor = moveDetailJobCursor(m.jobsCursor, m.detail.DisplayJobs, -1)
			m = m.saveActiveLogPane()
		}
	case "down", "j":
		if m.detail != nil {
			m.jobsCursor = moveDetailJobCursor(m.jobsCursor, m.detail.DisplayJobs, 1)
			m = m.saveActiveLogPane()
		}
	case "S", "c":
		if m.detail == nil || len(m.detail.DisplayJobs) == 0 {
			m.message = "no jobs loaded"
			return m, nil
		}
		job := m.detail.DisplayJobs[m.jobsCursor].Current
		actionKey := key.String()
		if actionKey == "S" {
			actionKey = "s"
		}
		action, ok := resolveAction(actionKey, job)
		if !ok {
			m.message = fmt.Sprintf("action not available for %s (%s)", job.Name, job.Status)
			return m, nil
		}
		m.pending = &action
		m.confirmText = ""
		m.confirmBackMode = modeDetail
		m.mode = modeConfirm
	case "l":
		if m.detail == nil || len(m.detail.DisplayJobs) == 0 {
			m.message = "no jobs loaded"
			return m, nil
		}
		job := logTarget(m.detail.DisplayJobs[m.jobsCursor])
		return m.openLogs(job, modeDetail)
	case "p":
		if m.detail == nil || len(m.detail.DisplayJobs) == 0 {
			m.message = "no jobs loaded"
			return m, nil
		}
		m.mode = modeJobs
		m.message = ""
		return m, tea.ClearScreen
	case "o":
		if m.detail != nil && m.detail.Pipeline.WebURL != "" {
			openURL(m.detail.Pipeline.WebURL)
		}
	}
	return m, nil
}

func (m model) handleJobsKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.detail == nil || len(m.detail.DisplayJobs) == 0 {
		m.mode = modeDetail
		return m, nil
	}
	switch key.String() {
	case "q", "esc":
		m.mode = modeDetail
		m.message = ""
		return m, tea.ClearScreen
	case "r", "R":
		m.message = "refreshing jobs..."
		return m, fetchDetailCmd(m.repo, m.detailID)
	case "up", "k":
		m.jobsCursor = moveUp(m.jobsCursor, len(m.detail.DisplayJobs))
	case "down", "j":
		m.jobsCursor = moveDown(m.jobsCursor, len(m.detail.DisplayJobs))
	case "s", "c":
		job := m.detail.DisplayJobs[m.jobsCursor].Current
		action, ok := resolveAction(key.String(), job)
		if !ok {
			m.message = fmt.Sprintf("action not available for %s (%s)", job.Name, job.Status)
			return m, nil
		}
		m.pending = &action
		m.confirmText = ""
		m.confirmBackMode = modeJobs
		m.mode = modeConfirm
	case "l":
		job := logTarget(m.detail.DisplayJobs[m.jobsCursor])
		return m.openLogs(job, modeJobs)
	}
	return m, nil
}

func (m model) openLogs(job job, backMode int) (tea.Model, tea.Cmd) {
	if m.logSplitRoot == nil || m.activeLogPane == 0 {
		m = m.initLogPanes()
	}
	m.logsViewport = viewport.New(max(1, m.width), max(1, m.height-4))
	m = m.setActivePaneLogs(job, backMode)
	cmds := []tea.Cmd{tea.ClearScreen, fetchLogsCmd(m.repo, job)}
	if shouldAutoRefreshLogs(&job) {
		cmds = append(cmds, tickLogsCmd(job.ID, m.logRefresh))
	}
	return m, tea.Batch(cmds...)
}

func (m model) handleLogsKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.logSearchMode {
		return m.handleLogSearchKey(key)
	}
	switch key.String() {
	case "s":
		return m.splitActiveLogPane(logSplitHorizontal), tea.ClearScreen
	case "v":
		return m.splitActiveLogPane(logSplitVertical), tea.ClearScreen
	case "x":
		return m.closeActiveLogPane(), tea.ClearScreen
	case "ctrl+h":
		return m.focusLogPane("h"), nil
	case "ctrl+j":
		return m.focusLogPane("j"), nil
	case "ctrl+k":
		return m.focusLogPane("k"), nil
	case "ctrl+l":
		return m.focusLogPane("l"), nil
	case "q", "esc":
		if len(m.logPanes) > 1 {
			m = m.setActivePaneMode(modeDetail)
			m.message = ""
			return m, tea.ClearScreen
		}
		if m.logBackMode == 0 {
			m.logBackMode = modeJobs
		}
		m.mode = m.logBackMode
		m.message = ""
		return m, tea.ClearScreen
	case "r":
		if m.logJob == nil {
			return m, nil
		}
		m.logsLoading = true
		m.message = ""
		m = m.configureLogViewport()
		if m.logs == "" {
			m.logsViewport.SetContent("loading logs...")
		}
		m = m.saveActiveLogPane()
		return m, fetchLogsCmd(m.repo, *m.logJob)
	case "/":
		return m.beginLogSearch().saveActiveLogPane(), nil
	case "g":
		m.logsViewport.GotoTop()
		return m.saveActiveLogPane(), nil
	case "G":
		m.logsViewport.GotoBottom()
		return m.saveActiveLogPane(), nil
	case "n":
		if m.logSearchActive {
			return m.jumpLogSearchMatch(1).saveActiveLogPane(), nil
		}
		if m.detail != nil && len(m.detail.DisplayJobs) > 0 {
			m.jobsCursor = moveDetailJobCursor(m.jobsCursor, m.detail.DisplayJobs, 1)
		}
		m.mode = modeDetail
		m.message = ""
		return m, tea.ClearScreen
	case "N":
		if m.logSearchActive {
			return m.jumpLogSearchMatch(-1).saveActiveLogPane(), nil
		}
	}
	var cmd tea.Cmd
	m.logsViewport, cmd = m.logsViewport.Update(key)
	return m.saveActiveLogPane(), cmd
}

func (m model) handleConfirmKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.pending == nil {
		if m.confirmBackMode == 0 {
			m.confirmBackMode = modeJobs
		}
		m.mode = m.confirmBackMode
		return m, tea.ClearScreen
	}
	prodPlay := strings.Contains(strings.ToLower(m.pending.Job.Name), "prod") && m.pending.Endpoint == "play"
	if !prodPlay {
		switch key.String() {
		case "y":
			action := *m.pending
			m.mode = modeDetail
			m.pending = nil
			return m, runActionCmd(m.repo, action)
		case "n", "q", "esc":
			m.pending = nil
			if m.confirmBackMode == 0 {
				m.confirmBackMode = modeJobs
			}
			m.mode = m.confirmBackMode
			return m, tea.ClearScreen
		}
		return m, nil
	}

	switch key.Type {
	case tea.KeyEsc:
		m.pending = nil
		if m.confirmBackMode == 0 {
			m.confirmBackMode = modeJobs
		}
		m.mode = m.confirmBackMode
		return m, tea.ClearScreen
	case tea.KeyBackspace, tea.KeyCtrlH:
		if len(m.confirmText) > 0 {
			runes := []rune(m.confirmText)
			m.confirmText = string(runes[:len(runes)-1])
		}
	case tea.KeySpace:
		m.confirmText += " "
	case tea.KeyRunes:
		m.confirmText += key.String()
	case tea.KeyEnter:
		if m.confirmText != m.pending.Job.Name {
			m.message = "confirmation did not match"
			m.confirmText = ""
			return m, nil
		}
		action := *m.pending
		m.mode = modeDetail
		m.pending = nil
		m.confirmText = ""
		return m, runActionCmd(m.repo, action)
	}
	return m, nil
}
