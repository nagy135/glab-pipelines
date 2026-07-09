package main

import (
	"fmt"
	"strings"
	"time"

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
		if m.listCursor >= len(m.list) {
			m.listCursor = max(0, len(m.list)-1)
		}
		for i := range m.logPanes {
			if m.logPanes[i].ListCursor >= len(m.list) {
				m.logPanes[i].ListCursor = max(0, len(m.list)-1)
			}
		}
		m.message = "refreshed " + time.Now().Format("15:04:05")
		return m, nil
	case detailMsg:
		updatedPane := false
		for i := range m.logPanes {
			if m.logPanes[i].DetailID != msg.pid {
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
		if msg.pid != m.detailID {
			if updatedPane && msg.err != nil {
				m.message = msg.err.Error()
			}
			return m, nil
		}
		m.detailLoading = false
		if msg.err != nil {
			m.message = msg.err.Error()
			return m, nil
		}
		m.detail = &msg.detail
		if m.jobsCursor >= len(m.detail.DisplayJobs) {
			m.jobsCursor = max(0, len(m.detail.DisplayJobs)-1)
		}
		m.message = ""
		return m, nil
	case actionMsg:
		if msg.err != nil {
			m.message = msg.err.Error()
			if m.confirmBackMode == 0 {
				m.confirmBackMode = modeJobs
			}
			m.mode = m.confirmBackMode
			return m, nil
		}
		m.message = fmt.Sprintf("%s sent for %s", msg.action.Verb, msg.action.Job.Name)
		m.mode = modeDetail
		m.pending = nil
		m.confirmText = ""
		m.detailLoading = true
		m = m.saveActiveLogPane()
		return m, fetchDetailCmd(m.repo, m.detailID)
	case logsMsg:
		if len(m.logPanes) > 0 {
			updated := false
			for i := range m.logPanes {
				pane := &m.logPanes[i]
				if pane.Mode != modeLogs || pane.Job == nil || pane.Job.ID != msg.jobID {
					continue
				}
				updated = true
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
				return m, nil
			}
		}
		if m.logJob == nil || msg.jobID != m.logJob.ID {
			return m, nil
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
			return m, nil
		}
		m.message = ""
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
			return m.saveActiveLogPane(), nil
		}
		if wasBottom {
			m.logsViewport.GotoBottom()
		} else {
			m.logsViewport.SetYOffset(yOffset)
		}
		return m.saveActiveLogPane(), nil
	case tickMsg:
		if m.mode != modeDetail || msg.pid != m.detailID {
			return m, nil
		}
		m.detailLoading = true
		m = m.saveActiveLogPane()
		return m, tea.Batch(fetchDetailCmd(m.repo, m.detailID), tickDetailCmd(m.detailID, m.refresh))
	case logTickMsg:
		if m.mode != modeLogs || m.logJob == nil || msg.jobID != m.logJob.ID || !shouldAutoRefreshLogs(m.logJob) {
			return m, nil
		}
		m.logsLoading = true
		m = m.saveActiveLogPane()
		return m, tea.Batch(fetchLogsCmd(m.repo, *m.logJob), tickLogsCmd(m.logJob.ID, m.logRefresh))
	}
	return m, nil
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
