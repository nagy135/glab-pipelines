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
		m.message = "refreshed " + time.Now().Format("15:04:05")
		return m, nil
	case detailMsg:
		if msg.pid != m.detailID {
			return m, nil
		}
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
		return m, fetchDetailCmd(m.repo, m.detailID)
	case logsMsg:
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
			return m, nil
		}
		if wasBottom {
			m.logsViewport.GotoBottom()
		} else {
			m.logsViewport.SetYOffset(yOffset)
		}
		return m, nil
	case tickMsg:
		if m.mode != modeDetail || msg.pid != m.detailID {
			return m, nil
		}
		return m, tea.Batch(fetchDetailCmd(m.repo, m.detailID), tickDetailCmd(m.detailID, m.refresh))
	case logTickMsg:
		if m.mode != modeLogs || m.logJob == nil || msg.jobID != m.logJob.ID {
			return m, nil
		}
		return m, tea.Batch(fetchLogsCmd(m.repo, *m.logJob), tickLogsCmd(m.logJob.ID, m.logRefresh))
	}
	return m, nil
}

func (m model) configureLogViewport() model {
	if m.width <= 0 || m.height <= 0 {
		return m
	}
	height := m.height - 4
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
	m.logsViewport.Width = m.width
	m.logsViewport.Height = height
	return m
}

func (m model) fillScreen(view string) string {
	if m.height <= 0 || m.width <= 0 {
		return view
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, strings.TrimRight(view, "\n"))
}
