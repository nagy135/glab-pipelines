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
	if key.String() == "Q" {
		return m, tea.Quit
	}
	if key.String() == "t" && m.mode != modeConfirm && m.mode != modeTheme && !(m.mode == modeLogs && m.logSearchMode) {
		return m.openThemePicker(), tea.ClearScreen
	}
	if key.String() == "b" && m.mode != modeConfirm && !(m.mode == modeLogs && m.logSearchMode) {
		return m.cycleActiveBorder(), tea.ClearScreen
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
	case modeTheme:
		return m.handleThemeKey(key)
	}
	return m, nil
}

func (m model) cycleActiveBorder() model {
	idx := borderIndex(m.borderName)
	idx = (idx + 1) % len(borderOptions)
	m.borderName = applyActiveBorder(borderOptions[idx].Name)
	m.message = "border: " + m.borderName
	if err := saveBorderName(m.borderName); err != nil {
		m.message += " (not saved)"
	}
	return m
}

func (m model) openThemePicker() model {
	m.themeBackMode = m.mode
	m.themeCursor = themeIndex(m.themeName)
	m.mode = modeTheme
	m.message = ""
	return m
}

func (m model) closeThemePicker() model {
	if m.themeBackMode == 0 {
		m.themeBackMode = modePipelines
	}
	backMode := m.themeBackMode
	if backMode == modeLogs {
		if idx := m.logPaneIndex(m.activeLogPane); idx >= 0 {
			m = m.restoreLogPane(m.logPanes[idx])
		} else {
			m.mode = backMode
		}
	} else {
		m.mode = backMode
	}
	m.themeBackMode = 0
	m.message = ""
	return m
}

func (m model) handleThemeKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "q", "esc":
		return m.closeThemePicker(), tea.ClearScreen
	case "up", "k":
		m.themeCursor = moveUp(m.themeCursor, len(themeOptions))
	case "down", "j":
		m.themeCursor = moveDown(m.themeCursor, len(themeOptions))
	case "enter":
		theme := themeOptions[m.themeCursor]
		applied, err := applyTheme(theme.Name)
		if err != nil {
			m.message = err.Error()
			return m, nil
		}
		m.themeName = applied
		m = m.closeThemePicker()
		m.message = "theme: " + applied
		if err := saveThemeName(applied); err != nil {
			m.message += " (not saved)"
		}
		return m, tea.ClearScreen
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
	case "o":
		return m.keepActiveLogPaneOnly(), tea.ClearScreen
	case "ctrl+h":
		return m.focusLogPane("h"), nil
	case "ctrl+j":
		return m.focusLogPane("j"), nil
	case "ctrl+k":
		return m.focusLogPane("k"), nil
	case "ctrl+l":
		return m.focusLogPane("l"), nil
	case "q":
		if len(m.logPanes) > 1 {
			return m.closeActiveLogPane(), tea.ClearScreen
		}
		return m, tea.Quit
	case "esc":
		return m, tea.Quit
	case "r", "R":
		m.listRequest++
		m.loadingList = true
		m.message = ""
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
		var cmd tea.Cmd
		m, cmd = m.requestDetail(m.detailID, true)
		return m, tea.Batch(tea.ClearScreen, cmd)
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
	case "o":
		return m.keepActiveLogPaneOnly(), tea.ClearScreen
	case "ctrl+h":
		return m.focusLogPane("h"), nil
	case "ctrl+j":
		return m.focusLogPane("j"), nil
	case "ctrl+k":
		return m.focusLogPane("k"), nil
	case "ctrl+l":
		return m.focusLogPane("l"), nil
	case "q":
		if len(m.logPanes) > 1 {
			return m.closeActiveLogPane(), tea.ClearScreen
		}
		return m, tea.Quit
	case "esc":
		m.mode = modePipelines
		m.message = ""
		return m, tea.ClearScreen
	case "r":
		m.detailLoading = true
		m = m.saveActiveLogPane()
		var cmd tea.Cmd
		m, cmd = m.requestDetail(m.detailID, true)
		return m, cmd
	case "L":
		m.showInlineLogs = !m.showInlineLogs
		m = m.saveActiveLogPane()
		m.inlineLogPollID++
		m.inlineLogsLoading = make(map[int64]bool)
		if !m.inlineLogsEnabled() {
			return m, nil
		}
		var cmd tea.Cmd
		m, cmd = m.requestInlineLogs(m.visibleInlineLogJobs())
		return m, tea.Batch(cmd, tickInlineLogsCmd(m.inlineLogPollID, m.logRefresh))
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
		if m.actionInFlight {
			m.message = "a job action is already in progress"
			return m, nil
		}
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
		action.PipelineID = m.detailID
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
		m.detailLoading = true
		var cmd tea.Cmd
		m, cmd = m.requestDetail(m.detailID, true)
		return m, cmd
	case "up", "k":
		m.jobsCursor = moveUp(m.jobsCursor, len(m.detail.DisplayJobs))
	case "down", "j":
		m.jobsCursor = moveDown(m.jobsCursor, len(m.detail.DisplayJobs))
	case "s", "c":
		if m.actionInFlight {
			m.message = "a job action is already in progress"
			return m, nil
		}
		job := m.detail.DisplayJobs[m.jobsCursor].Current
		action, ok := resolveAction(key.String(), job)
		if !ok {
			m.message = fmt.Sprintf("action not available for %s (%s)", job.Name, job.Status)
			return m, nil
		}
		action.PipelineID = m.detailID
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
	var fetchCmd tea.Cmd
	m, fetchCmd = m.requestLogs(job, true)
	cmds := []tea.Cmd{tea.ClearScreen, fetchCmd}
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
	case "o":
		return m.keepActiveLogPaneOnly(), tea.ClearScreen
	case "ctrl+h":
		return m.focusLogPane("h"), nil
	case "ctrl+j":
		return m.focusLogPane("j"), nil
	case "ctrl+k":
		return m.focusLogPane("k"), nil
	case "ctrl+l":
		return m.focusLogPane("l"), nil
	case "q":
		if len(m.logPanes) > 1 {
			return m.closeActiveLogPane(), tea.ClearScreen
		}
		return m, tea.Quit
	case "esc":
		return m.leaveLogs(m.logBackMode), tea.ClearScreen
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
		var cmd tea.Cmd
		m, cmd = m.requestLogs(*m.logJob, true)
		return m, cmd
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
		return m.leaveLogs(modeDetail), tea.ClearScreen
	case "N":
		if m.logSearchActive {
			return m.jumpLogSearchMatch(-1).saveActiveLogPane(), nil
		}
	}
	var cmd tea.Cmd
	m.logsViewport, cmd = m.logsViewport.Update(key)
	return m.saveActiveLogPane(), cmd
}

func (m model) leaveLogs(backMode int) model {
	if backMode == 0 {
		backMode = modeJobs
	}
	m.mode = backMode
	if idx := m.logPaneIndex(m.activeLogPane); idx >= 0 {
		pane := m.currentLogPaneState(m.activeLogPane)
		pane.Mode = backMode
		if pane.Mode == modeJobs {
			pane.Mode = modeDetail
		}
		pane.Loading = m.paneModeLoading(pane.Mode)
		m.logPanes[idx] = pane
	}
	m.message = ""
	return m
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
			if m.confirmBackMode == 0 {
				m.confirmBackMode = modeDetail
			}
			m.mode = m.confirmBackMode
			m.pending = nil
			m.confirmText = ""
			m.actionInFlight = true
			m.actionRequest++
			return m, runActionCmd(m.repo, action, m.actionRequest)
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
		if m.confirmBackMode == 0 {
			m.confirmBackMode = modeDetail
		}
		m.mode = m.confirmBackMode
		m.pending = nil
		m.confirmText = ""
		m.actionInFlight = true
		m.actionRequest++
		return m, runActionCmd(m.repo, action, m.actionRequest)
	}
	return m, nil
}
