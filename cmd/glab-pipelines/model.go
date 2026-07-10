package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m model) Init() tea.Cmd {
	return fetchPipelinesCmd(m.repo, m.status, m.limit, m.listRequest)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = m.configureLogViewport()
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	case pipelinesMsg:
		if msg.requestID != m.listRequest {
			return m, nil
		}
		m.loadingList = false
		if msg.err != nil {
			m.message = msg.err.Error()
			return m, nil
		}
		m.list = msg.pipelines
		savePipelineCache(m.repo, m.status, m.limit, m.list)
		if m.listCursor >= len(m.list) {
			m.listCursor = max(0, len(m.list)-1)
		}
		for i := range m.logPanes {
			if m.logPanes[i].ListCursor >= len(m.list) {
				m.logPanes[i].ListCursor = max(0, len(m.list)-1)
			}
		}
		m.message = ""
		return m, nil
	case detailMsg:
		if msg.requestID != m.detailRequests[msg.pid] || msg.pollID != m.detailPolls[msg.pid] {
			return m, nil
		}
		updatedPane := false
		relevant := m.mode == modeDetail && msg.pid == m.detailID
		for i := range m.logPanes {
			if m.logPanes[i].Mode == modeDetail && m.logPanes[i].DetailID == msg.pid {
				relevant = true
				break
			}
		}
		if !relevant {
			return m, nil
		}
		pollCmd := tickDetailCmd(msg.pid, msg.pollID, m.refresh)
		var soundCmd tea.Cmd
		if msg.err == nil {
			var sounds []jobSound
			m, sounds = m.observeJobStatuses(msg.detail.Jobs)
			soundCmd = playJobSoundsCmd(sounds)
			m = m.updateLogJobSnapshots(msg.detail.Jobs)
		}
		for i := range m.logPanes {
			if m.logPanes[i].Mode != modeDetail || m.logPanes[i].DetailID != msg.pid {
				continue
			}
			updatedPane = true
			m.logPanes[i].Loading = false
			if msg.err == nil {
				detailCopy := msg.detail
				m.logPanes[i].Detail = &detailCopy
				if m.logPanes[i].JobsCursor >= len(msg.detail.DisplayJobs) {
					m.logPanes[i].JobsCursor = max(0, len(msg.detail.DisplayJobs)-1)
				}
			}
		}
		if m.mode != modeDetail || msg.pid != m.detailID {
			if updatedPane && msg.err != nil {
				m.message = msg.err.Error()
			}
			return m, tea.Batch(soundCmd, pollCmd)
		}
		m.detailLoading = false
		if msg.err != nil {
			m.message = msg.err.Error()
			return m, tea.Batch(soundCmd, pollCmd)
		}
		m.detail = &msg.detail
		if m.jobsCursor >= len(m.detail.DisplayJobs) {
			m.jobsCursor = max(0, len(m.detail.DisplayJobs)-1)
		}
		m.message = ""
		return m, tea.Batch(soundCmd, pollCmd)
	case actionMsg:
		if msg.requestID != m.actionRequest {
			return m, nil
		}
		m.actionInFlight = false
		if msg.err != nil {
			m.message = msg.err.Error()
			return m, nil
		}
		m.message = fmt.Sprintf("%s sent for %s", msg.action.Verb, msg.action.Job.Name)
		if !m.watchesDetail(msg.action.PipelineID) {
			return m, nil
		}
		m = m.setDetailLoading(msg.action.PipelineID)
		var cmd tea.Cmd
		m, cmd = m.requestDetail(msg.action.PipelineID, true)
		return m, cmd
	case logsMsg:
		if msg.requestID != m.logRequests[msg.jobID] || msg.pollID != m.logPolls[msg.jobID] {
			return m, nil
		}
		var soundCmd tea.Cmd
		if msg.job != nil && m.hasLogJob(msg.jobID) {
			var sounds []jobSound
			m, sounds = m.observeJobStatuses([]job{*msg.job})
			soundCmd = playJobSoundsCmd(sounds)
		}
		if len(m.logPanes) > 0 {
			updated := false
			for i := range m.logPanes {
				pane := &m.logPanes[i]
				if pane.Mode != modeLogs || pane.Job == nil || pane.Job.ID != msg.jobID {
					continue
				}
				updated = true
				if msg.job != nil {
					pane.Job = cloneJobPtr(msg.job)
				}
				wasBottom := pane.Viewport.AtBottom()
				yOffset := pane.Viewport.YOffset
				pane.Loading = false
				if msg.err != nil {
					if pane.ID == m.activeLogPane {
						m.message = msg.err.Error()
					}
					if pane.Logs == "" {
						pane.Viewport.SetContent(redStyle.Render(msg.err.Error()))
					}
					continue
				}
				pane.Logs = msg.logs
				if strings.TrimSpace(pane.Logs) == "" {
					pane.Logs = "(empty log)"
				}
				if pane.SearchQuery != "" {
					pane.SearchMatches = findLogSearchMatches(pane.Logs, pane.SearchQuery)
					if len(pane.SearchMatches) == 0 {
						pane.SearchIndex = -1
					} else if pane.SearchIndex >= len(pane.SearchMatches) {
						pane.SearchIndex = len(pane.SearchMatches) - 1
					}
				}
				pane.Viewport.SetContent(renderLogContentFor(pane.Logs, pane.SearchMatches, pane.SearchIndex))
				if pane.SearchQuery != "" {
					pane.Viewport.SetYOffset(yOffset)
				} else if wasBottom {
					pane.Viewport.GotoBottom()
				} else {
					pane.Viewport.SetYOffset(yOffset)
				}
			}
			if updated {
				idx := m.logPaneIndex(m.activeLogPane)
				if idx >= 0 {
					m = m.restoreLogPane(m.logPanes[idx])
				}
				if msg.statusErr != nil && m.logJob != nil && m.logJob.ID == msg.jobID {
					m.message = msg.statusErr.Error()
				}
				return m, tea.Batch(soundCmd, m.nextLogPoll(msg.jobID, msg.pollID))
			}
		}
		if m.logJob == nil || msg.jobID != m.logJob.ID {
			return m, nil
		}
		if msg.job != nil {
			m.logJob = cloneJobPtr(msg.job)
		}
		wasBottom := m.logsViewport.AtBottom()
		yOffset := m.logsViewport.YOffset
		m.logsLoading = false
		if msg.err != nil {
			m.message = msg.err.Error()
			m = m.configureLogViewport()
			if m.logs == "" {
				m.logsViewport.SetContent(redStyle.Render(msg.err.Error()))
			}
			return m, tea.Batch(soundCmd, m.nextLogPoll(msg.jobID, msg.pollID))
		}
		m.message = ""
		if msg.statusErr != nil {
			m.message = msg.statusErr.Error()
		}
		m.logs = msg.logs
		if strings.TrimSpace(m.logs) == "" {
			m.logs = "(empty log)"
		}
		m = m.configureLogViewport()
		m.logsViewport.SetContent(m.renderLogContent())
		if m.logSearchQuery != "" {
			m = m.refreshLogSearchMatches()
			m.logsViewport.SetContent(m.renderLogContent())
			m.logsViewport.SetYOffset(yOffset)
			m = m.saveActiveLogPane()
			return m, tea.Batch(soundCmd, m.nextLogPoll(msg.jobID, msg.pollID))
		}
		if wasBottom {
			m.logsViewport.GotoBottom()
		} else {
			m.logsViewport.SetYOffset(yOffset)
		}
		m = m.saveActiveLogPane()
		return m, tea.Batch(soundCmd, m.nextLogPoll(msg.jobID, msg.pollID))
	case tickMsg:
		if msg.pollID != m.detailPolls[msg.pid] || !m.watchesDetail(msg.pid) {
			return m, nil
		}
		m = m.setDetailLoading(msg.pid)
		var cmd tea.Cmd
		m, cmd = m.requestDetail(msg.pid, false)
		return m, cmd
	case logTickMsg:
		if msg.pollID != m.logPolls[msg.jobID] {
			return m, nil
		}
		var pollingJob *job
		if m.mode == modeLogs && m.logJob != nil && msg.jobID == m.logJob.ID && shouldAutoRefreshLogs(m.logJob) {
			pollingJob = cloneJobPtr(m.logJob)
			m.logsLoading = true
		}
		for i := range m.logPanes {
			pane := &m.logPanes[i]
			if pane.Mode != modeLogs || pane.Job == nil || pane.Job.ID != msg.jobID || !shouldAutoRefreshLogs(pane.Job) {
				continue
			}
			if pollingJob == nil {
				pollingJob = cloneJobPtr(pane.Job)
			}
			pane.Loading = true
		}
		if pollingJob == nil {
			return m, nil
		}
		var cmd tea.Cmd
		m, cmd = m.requestLogs(*pollingJob, false)
		return m, cmd
	}
	return m, nil
}

func (m model) watchesDetail(pid int) bool {
	if m.mode == modeDetail && m.detailID == pid {
		return true
	}
	for _, pane := range m.logPanes {
		if pane.Mode == modeDetail && pane.DetailID == pid {
			return true
		}
	}
	return false
}

func (m model) setDetailLoading(pid int) model {
	if m.mode == modeDetail && m.detailID == pid {
		m.detailLoading = true
	}
	for i := range m.logPanes {
		if m.logPanes[i].Mode == modeDetail && m.logPanes[i].DetailID == pid {
			m.logPanes[i].Loading = true
		}
	}
	return m
}

func (m model) nextLogPoll(jobID int64, pollID int) tea.Cmd {
	if pollID != m.logPolls[jobID] {
		return nil
	}
	if m.mode == modeLogs && m.logJob != nil && m.logJob.ID == jobID && shouldAutoRefreshLogs(m.logJob) {
		return tickLogsCmd(jobID, pollID, m.logRefresh)
	}
	for _, pane := range m.logPanes {
		if pane.Mode == modeLogs && pane.Job != nil && pane.Job.ID == jobID && shouldAutoRefreshLogs(pane.Job) {
			return tickLogsCmd(jobID, pollID, m.logRefresh)
		}
	}
	return nil
}

func (m model) hasLogJob(jobID int64) bool {
	if m.mode == modeLogs && m.logJob != nil && m.logJob.ID == jobID {
		return true
	}
	for _, pane := range m.logPanes {
		if pane.Mode == modeLogs && pane.Job != nil && pane.Job.ID == jobID {
			return true
		}
	}
	return false
}

func (m model) updateLogJobSnapshots(jobs []job) model {
	byID := make(map[int64]job, len(jobs))
	for _, j := range jobs {
		byID[j.ID] = j
	}
	if m.logJob != nil {
		if updated, ok := byID[m.logJob.ID]; ok {
			m.logJob = cloneJobPtr(&updated)
		}
	}
	for i := range m.logPanes {
		pane := &m.logPanes[i]
		if pane.Job == nil {
			continue
		}
		if updated, ok := byID[pane.Job.ID]; ok {
			pane.Job = cloneJobPtr(&updated)
		}
	}
	return m
}

func (m model) configureLogViewport() model {
	if m.width <= 0 || m.height <= 0 {
		return m
	}
	if len(m.logPanes) > 1 {
		rect, ok := m.activeLogPaneRect()
		if !ok {
			return m
		}
		height := rect.Height - 2 - logPaneHeaderHeight
		if height < 1 {
			height = 1
		}
		m.logsViewport.Width = max(1, rect.Width-2)
		m.logsViewport.Height = height
		return m
	}
	height := m.height - 5
	if m.message != "" {
		height--
	}
	if m.logsLoading {
		height--
	}
	if m.logSearchMode || m.logSearchQuery != "" {
		height--
	}
	if height < 1 {
		height = 1
	}
	m.logsViewport.Width = max(1, m.width-2)
	m.logsViewport.Height = height
	return m
}

func (m model) fillScreen(view string) string {
	if m.height <= 0 || m.width <= 0 {
		return view
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, strings.TrimRight(view, "\n"))
}
