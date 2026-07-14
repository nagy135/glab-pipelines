package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const logPaneHeaderHeight = 2

type borderOption struct {
	Name   string
	Border lipgloss.Border
}

var (
	activePaneBorderName = "double"
	borderOptions        = []borderOption{
		{Name: "normal", Border: lipgloss.NormalBorder()},
		{Name: "rounded", Border: lipgloss.RoundedBorder()},
		{Name: "thick", Border: lipgloss.ThickBorder()},
		{Name: "double", Border: lipgloss.DoubleBorder()},
	}
)

func applyActiveBorder(name string) string {
	normalized := normalizeBorderName(name)
	if normalized == "" {
		normalized = activePaneBorderName
	}
	for _, option := range borderOptions {
		if option.Name == normalized {
			activePaneBorderName = option.Name
			return option.Name
		}
	}
	activePaneBorderName = "double"
	return activePaneBorderName
}

func normalizeBorderName(name string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), "_", "-")
}

func borderIndex(name string) int {
	normalized := normalizeBorderName(name)
	for i, option := range borderOptions {
		if option.Name == normalized {
			return i
		}
	}
	return borderIndex(activePaneBorderName)
}

func activePaneBorder() lipgloss.Border {
	for _, option := range borderOptions {
		if option.Name == activePaneBorderName {
			return option.Border
		}
	}
	return lipgloss.DoubleBorder()
}

type logPaneRect struct {
	PaneID int
	X      int
	Y      int
	Width  int
	Height int
}

func (m model) initLogPanes() model {
	m.nextLogPaneID = 2
	m.activeLogPane = 1
	m.logSplitRoot = &logSplitNode{PaneID: 1}
	m.logPanes = []logPane{m.currentLogPaneState(1)}
	return m
}

func (m model) currentLogPaneState(id int) logPane {
	loading := false
	switch m.mode {
	case modePipelines:
		loading = m.loadingList
	case modeDetail:
		loading = m.detailLoading
	case modeLogs, modeCode:
		loading = m.logsLoading
	}
	return logPane{
		ID:               id,
		Mode:             m.mode,
		ListCursor:       m.listCursor,
		DetailID:         m.detailID,
		Detail:           cloneDetailPtr(m.detail),
		JobsCursor:       m.jobsCursor,
		Job:              cloneJobPtr(m.logJob),
		BackMode:         m.logBackMode,
		Logs:             m.logs,
		Loading:          loading,
		Viewport:         m.logsViewport,
		SearchMode:       m.logSearchMode,
		SearchActive:     m.logSearchActive,
		SearchQuery:      m.logSearchQuery,
		SearchMatches:    append([]logSearchMatch(nil), m.logSearchMatches...),
		SearchIndex:      m.logSearchIndex,
		ShowInlineLogs:   m.showInlineLogs,
		WrapContent:      m.wrapContent,
		ShowLineNumbers:  m.showLineNumbers,
		ScrollOffset:     m.scrollOffset,
		HorizontalOffset: m.horizontalOffset,
	}
}

func cloneDetailPtr(d *detail) *detail {
	if d == nil {
		return nil
	}
	copy := *d
	copy.Jobs = append([]job(nil), d.Jobs...)
	copy.DisplayJobs = append([]uiJob(nil), d.DisplayJobs...)
	return &copy
}

func cloneJobPtr(j *job) *job {
	if j == nil {
		return nil
	}
	copy := *j
	return &copy
}

func (m model) saveActiveLogPane() model {
	if m.activeLogPane == 0 {
		return m
	}
	idx := m.logPaneIndex(m.activeLogPane)
	if idx < 0 {
		return m
	}
	m.logPanes[idx] = m.currentLogPaneState(m.activeLogPane)
	return m
}

func (m model) restoreLogPane(p logPane) model {
	m.activeLogPane = p.ID
	m.mode = p.Mode
	m.listCursor = p.ListCursor
	m.detailID = p.DetailID
	m.detail = cloneDetailPtr(p.Detail)
	m.detailLoading = p.Mode == modeDetail && p.Loading
	m.jobsCursor = p.JobsCursor
	m.logJob = cloneJobPtr(p.Job)
	m.logBackMode = p.BackMode
	m.logs = p.Logs
	m.logsLoading = (p.Mode == modeLogs || p.Mode == modeCode) && p.Loading
	m.logsViewport = p.Viewport
	m.logSearchMode = p.SearchMode
	m.logSearchActive = p.SearchActive
	m.logSearchQuery = p.SearchQuery
	m.logSearchMatches = append([]logSearchMatch(nil), p.SearchMatches...)
	m.logSearchIndex = p.SearchIndex
	m.showInlineLogs = p.ShowInlineLogs
	m.wrapContent = p.WrapContent
	m.showLineNumbers = p.ShowLineNumbers
	m.scrollOffset = p.ScrollOffset
	m.horizontalOffset = p.HorizontalOffset
	m = m.configureLogViewport()
	if (m.mode == modeLogs || m.mode == modeCode) && m.logs != "" {
		m.logsViewport.SetContent(m.renderLogContent())
	}
	return m
}

func (m model) logPaneIndex(id int) int {
	for i, pane := range m.logPanes {
		if pane.ID == id {
			return i
		}
	}
	return -1
}

func (m model) splitActiveLogPane(direction int) model {
	if m.logSplitRoot == nil || m.activeLogPane == 0 {
		m = m.initLogPanes()
	}
	m = m.saveActiveLogPane()
	newID := m.nextLogPaneID
	if newID == 0 {
		newID = len(m.logPanes) + 1
	}
	m.nextLogPaneID = newID + 1
	newPane := m.currentLogPaneState(newID)
	m.logPanes = append(m.logPanes, newPane)
	m.logSplitRoot = splitLogNode(m.logSplitRoot, m.activeLogPane, newID, direction)
	m.activeLogPane = newID
	m = m.restoreLogPane(newPane)
	m.message = ""
	return m
}

func (m model) paneModeLoading(mode int) bool {
	switch mode {
	case modePipelines:
		return m.loadingList
	case modeDetail:
		return m.detailLoading
	case modeJobs:
		return m.detailLoading
	case modeLogs, modeCode:
		return m.logsLoading
	}
	return false
}

func (m model) setActivePaneLogs(job job, backMode int) model {
	m.logJob = &job
	m.logBackMode = backMode
	m.logs = ""
	m.logsLoading = true
	m = m.clearLogSearch()
	m.message = ""
	m.mode = modeLogs
	m.logsViewport.Width = max(1, m.width)
	m.logsViewport.Height = max(1, m.height-4)
	m = m.configureLogViewport()
	m.logsViewport.SetContent("loading logs...")
	idx := m.logPaneIndex(m.activeLogPane)
	if idx >= 0 {
		m.logPanes[idx] = m.currentLogPaneState(m.activeLogPane)
	}
	return m
}

func (m model) setActivePaneCode(job job, backMode int) model {
	m.logJob = &job
	m.logBackMode = backMode
	m.logs = ""
	m.logsLoading = true
	m = m.clearLogSearch()
	m.message = ""
	m.mode = modeCode
	m.logsViewport.Width = max(1, m.width)
	m.logsViewport.Height = max(1, m.height-4)
	m = m.configureLogViewport()
	m.logsViewport.SetContent("loading job code...")
	if idx := m.logPaneIndex(m.activeLogPane); idx >= 0 {
		m.logPanes[idx] = m.currentLogPaneState(m.activeLogPane)
	}
	return m
}

func splitLogNode(node *logSplitNode, activeID int, newID int, direction int) *logSplitNode {
	if node == nil {
		return &logSplitNode{PaneID: newID}
	}
	if node.isLeaf() {
		if node.PaneID != activeID {
			return node
		}
		return &logSplitNode{
			Direction: direction,
			First:     &logSplitNode{PaneID: activeID},
			Second:    &logSplitNode{PaneID: newID},
		}
	}
	node.First = splitLogNode(node.First, activeID, newID, direction)
	node.Second = splitLogNode(node.Second, activeID, newID, direction)
	return node
}

func (m model) closeActiveLogPane() model {
	if len(m.logPanes) <= 1 || m.logSplitRoot == nil {
		m.message = "cannot close the last split"
		return m
	}
	m = m.saveActiveLogPane()
	newRoot, replacementID := removeLogPaneNode(m.logSplitRoot, m.activeLogPane)
	if newRoot == nil || replacementID == 0 {
		m.message = "cannot close split"
		return m
	}
	m.logSplitRoot = newRoot
	idx := m.logPaneIndex(m.activeLogPane)
	if idx >= 0 {
		m.logPanes = append(m.logPanes[:idx], m.logPanes[idx+1:]...)
	}
	idx = m.logPaneIndex(replacementID)
	if idx < 0 && len(m.logPanes) > 0 {
		idx = 0
	}
	if idx >= 0 {
		m = m.restoreLogPane(m.logPanes[idx])
	}
	m.message = ""
	return m
}

func (m model) keepActiveLogPaneOnly() model {
	if len(m.logPanes) <= 1 || m.activeLogPane == 0 {
		return m
	}
	m = m.saveActiveLogPane()
	idx := m.logPaneIndex(m.activeLogPane)
	if idx < 0 {
		return m
	}
	pane := m.logPanes[idx]
	m.logPanes = []logPane{pane}
	m.logSplitRoot = &logSplitNode{PaneID: pane.ID}
	m.message = ""
	return m.restoreLogPane(pane)
}

func removeLogPaneNode(node *logSplitNode, paneID int) (*logSplitNode, int) {
	if node == nil {
		return nil, 0
	}
	if node.isLeaf() {
		if node.PaneID == paneID {
			return nil, 0
		}
		return node, node.PaneID
	}
	if node.First != nil && node.First.contains(paneID) {
		newFirst, _ := removeLogPaneNode(node.First, paneID)
		if newFirst == nil {
			return node.Second, firstPaneID(node.Second)
		}
		node.First = newFirst
		return node, firstPaneID(newFirst)
	}
	if node.Second != nil && node.Second.contains(paneID) {
		newSecond, _ := removeLogPaneNode(node.Second, paneID)
		if newSecond == nil {
			return node.First, firstPaneID(node.First)
		}
		node.Second = newSecond
		return node, firstPaneID(newSecond)
	}
	return node, firstPaneID(node)
}

func firstPaneID(node *logSplitNode) int {
	if node == nil {
		return 0
	}
	if node.isLeaf() {
		return node.PaneID
	}
	if id := firstPaneID(node.First); id != 0 {
		return id
	}
	return firstPaneID(node.Second)
}

func (n *logSplitNode) isLeaf() bool {
	return n != nil && n.First == nil && n.Second == nil
}

func (n *logSplitNode) contains(paneID int) bool {
	if n == nil {
		return false
	}
	if n.isLeaf() {
		return n.PaneID == paneID
	}
	return n.First.contains(paneID) || n.Second.contains(paneID)
}

func (m model) focusLogPane(direction string) model {
	if len(m.logPanes) <= 1 {
		return m
	}
	m = m.saveActiveLogPane()
	_, height := m.logSplitAreaSize()
	rects := collectLogPaneRects(m.logSplitRoot, 0, 0, m.width, height)
	targetID := directionalLogPane(rects, m.activeLogPane, direction)
	if targetID == 0 || targetID == m.activeLogPane {
		return m
	}
	idx := m.logPaneIndex(targetID)
	if idx < 0 {
		return m
	}
	m.message = ""
	return m.restoreLogPane(m.logPanes[idx])
}

func directionalLogPane(rects []logPaneRect, activeID int, direction string) int {
	active, ok := findLogPaneRect(rects, activeID)
	if !ok {
		return 0
	}
	bestID := 0
	bestDistance := int(^uint(0) >> 1)
	bestOverlap := -1
	for _, rect := range rects {
		if rect.PaneID == activeID {
			continue
		}
		distance, overlap, ok := logPaneDirectionScore(active, rect, direction)
		if !ok {
			continue
		}
		if distance < bestDistance || (distance == bestDistance && overlap > bestOverlap) {
			bestID = rect.PaneID
			bestDistance = distance
			bestOverlap = overlap
		}
	}
	return bestID
}

func logPaneDirectionScore(active, candidate logPaneRect, direction string) (int, int, bool) {
	switch direction {
	case "h":
		if candidate.X+candidate.Width > active.X {
			return 0, 0, false
		}
		return active.X - (candidate.X + candidate.Width), lineOverlap(active.Y, active.Y+active.Height, candidate.Y, candidate.Y+candidate.Height), true
	case "l":
		if candidate.X < active.X+active.Width {
			return 0, 0, false
		}
		return candidate.X - (active.X + active.Width), lineOverlap(active.Y, active.Y+active.Height, candidate.Y, candidate.Y+candidate.Height), true
	case "k":
		if candidate.Y+candidate.Height > active.Y {
			return 0, 0, false
		}
		return active.Y - (candidate.Y + candidate.Height), lineOverlap(active.X, active.X+active.Width, candidate.X, candidate.X+candidate.Width), true
	case "j":
		if candidate.Y < active.Y+active.Height {
			return 0, 0, false
		}
		return candidate.Y - (active.Y + active.Height), lineOverlap(active.X, active.X+active.Width, candidate.X, candidate.X+candidate.Width), true
	}
	return 0, 0, false
}

func lineOverlap(aStart, aEnd, bStart, bEnd int) int {
	start := max(aStart, bStart)
	end := min(aEnd, bEnd)
	if end <= start {
		return 0
	}
	return end - start
}

func findLogPaneRect(rects []logPaneRect, paneID int) (logPaneRect, bool) {
	for _, rect := range rects {
		if rect.PaneID == paneID {
			return rect, true
		}
	}
	return logPaneRect{}, false
}

func (m model) activeLogPaneRect() (logPaneRect, bool) {
	_, height := m.logSplitAreaSize()
	rects := collectLogPaneRects(m.logSplitRoot, 0, 0, m.width, height)
	return findLogPaneRect(rects, m.activeLogPane)
}

func collectLogPaneRects(node *logSplitNode, x, y, width, height int) []logPaneRect {
	if node == nil || width <= 0 || height <= 0 {
		return nil
	}
	if node.isLeaf() {
		return []logPaneRect{{PaneID: node.PaneID, X: x, Y: y, Width: width, Height: height}}
	}
	if node.Direction == logSplitVertical {
		firstWidth := max(1, width/2)
		secondWidth := max(1, width-firstWidth)
		return append(
			collectLogPaneRects(node.First, x, y, firstWidth, height),
			collectLogPaneRects(node.Second, x+firstWidth, y, secondWidth, height)...,
		)
	}
	firstHeight := max(1, height/2)
	secondHeight := max(1, height-firstHeight)
	return append(
		collectLogPaneRects(node.First, x, y, width, firstHeight),
		collectLogPaneRects(node.Second, x, y+firstHeight, width, secondHeight)...,
	)
}

func (m model) logSplitAreaSize() (int, int) {
	height := m.height - 2
	if m.mode == modePipelines && m.repo != "" {
		height--
	}
	if height < 1 {
		height = 1
	}
	return max(1, m.width), height
}

func (m model) viewLogSplits() string {
	if m.logSplitRoot == nil {
		return ""
	}
	width, height := m.logSplitAreaSize()
	return m.renderLogSplitNode(m.logSplitRoot, width, height)
}

func (m model) renderSinglePane(body string) string {
	width, height := m.logSplitAreaSize()
	contentWidth := max(1, width-2)
	contentHeight := max(1, height-2)
	if m.height > 0 {
		wrap := m.wrapContent && m.mode != modeLogs && m.mode != modeCode
		body = renderScrollableBody(body, contentWidth, contentHeight, m.scrollOffset, m.horizontalOffset, wrap)
	}
	return m.renderPaneBox(body, true, m.paneModeLoading(m.mode), contentWidth, contentHeight)
}

func (m model) renderLogSplitNode(node *logSplitNode, width, height int) string {
	if node == nil || width <= 0 || height <= 0 {
		return ""
	}
	if node.isLeaf() {
		pane, ok := m.logPaneForRender(node.PaneID)
		if !ok {
			return ""
		}
		return m.renderLogPane(pane, node.PaneID == m.activeLogPane, width, height)
	}
	if node.Direction == logSplitVertical {
		firstWidth := max(1, width/2)
		secondWidth := max(1, width-firstWidth)
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.renderLogSplitNode(node.First, firstWidth, height),
			m.renderLogSplitNode(node.Second, secondWidth, height),
		)
	}
	firstHeight := max(1, height/2)
	secondHeight := max(1, height-firstHeight)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderLogSplitNode(node.First, width, firstHeight),
		m.renderLogSplitNode(node.Second, width, secondHeight),
	)
}

func (m model) logPaneForRender(id int) (logPane, bool) {
	if id == m.activeLogPane {
		return m.currentLogPaneState(id), true
	}
	idx := m.logPaneIndex(id)
	if idx < 0 {
		return logPane{}, false
	}
	return m.logPanes[idx], true
}

func (m model) renderLogPane(pane logPane, active bool, width, height int) string {
	if height < logPaneHeaderHeight+1 {
		height = logPaneHeaderHeight + 1
	}
	contentWidth := max(1, width-2)
	contentHeight := max(1, height-2)
	innerHeight := max(1, contentHeight-logPaneHeaderHeight)
	if pane.Mode == modePipelines {
		return m.renderPipelinePane(pane, active, contentWidth, contentHeight)
	}
	if pane.Mode == modeDetail {
		return m.renderDetailPane(pane, active, contentWidth, contentHeight)
	}
	rendered := renderViewerContent(pane.Logs, pane.Mode, pane.SearchMatches, pane.SearchIndex, pane.ShowLineNumbers, pane.WrapContent, contentWidth, innerHeight)
	logWidth := contentWidth
	if !pane.WrapContent {
		logWidth = max(contentWidth, widestLineWidth(rendered))
	}
	visibleHeight := innerHeight
	if logWidth > contentWidth {
		visibleHeight = max(1, visibleHeight-1)
	}
	pane.Viewport.Width = logWidth
	pane.Viewport.Height = visibleHeight
	pane.Viewport.SetContent(renderViewerContent(pane.Logs, pane.Mode, pane.SearchMatches, pane.SearchIndex, pane.ShowLineNumbers, pane.WrapContent, contentWidth, visibleHeight))
	title := logPaneTitle(pane)
	if active {
		title = selectedStyle.Render(truncate(title, contentWidth))
	} else {
		title = metaStyle.Render(truncate(title, contentWidth))
	}
	status := truncate(logPaneStatus(pane), contentWidth)
	body := title + "\n" + dimStyle.Render(status) + "\n" + renderViewportBody(pane.Viewport.View(), logWidth, visibleHeight, pane.Viewport.YOffset, pane.Viewport.TotalLineCount())
	body = renderScrollableBody(body, contentWidth, contentHeight, 0, pane.HorizontalOffset, false)
	return m.renderPaneBox(body, active, m.paneIsLoading(pane, active), contentWidth, contentHeight)
}

func (m model) paneIsLoading(pane logPane, active bool) bool {
	if pane.Mode == modePipelines {
		return m.loadingList
	}
	if active {
		return m.paneModeLoading(pane.Mode)
	}
	return pane.Loading
}

func logPaneTitle(pane logPane) string {
	if pane.Mode == modePipelines {
		return fmt.Sprintf(" pane %d  pipeline list", pane.ID)
	}
	if pane.Mode == modeDetail {
		if pane.DetailID != 0 {
			return fmt.Sprintf(" pane %d  pipeline #%d", pane.ID, pane.DetailID)
		}
		return fmt.Sprintf(" pane %d  pipeline detail", pane.ID)
	}
	name := "unknown job"
	if pane.Job != nil {
		name = pane.Job.Name
	}
	view := "logs"
	if pane.Mode == modeCode {
		view = "code"
	}
	return fmt.Sprintf(" pane %d  %s %s", pane.ID, name, view)
}

func logPaneStatus(pane logPane) string {
	if pane.Mode == modePipelines {
		if pane.Loading {
			return "loading pipelines..."
		}
		return "select a pipeline, then enter opens details in this pane"
	}
	if pane.Mode == modeDetail {
		if pane.Loading {
			return "loading pipeline..."
		}
		return "select a job, then l opens logs or C opens code in this pane"
	}
	parts := make([]string, 0, 3)
	if pane.Job != nil && pane.Job.Status != "" {
		parts = append(parts, pane.Job.Status)
	}
	if pane.Loading {
		parts = append(parts, "loading")
	}
	if pane.SearchMode || pane.SearchQuery != "" {
		if pane.SearchQuery == "" {
			parts = append(parts, "/")
		} else if len(pane.SearchMatches) == 0 {
			parts = append(parts, "/"+pane.SearchQuery+" no matches")
		} else {
			parts = append(parts, fmt.Sprintf("/%s %d/%d", pane.SearchQuery, max(1, pane.SearchIndex+1), len(pane.SearchMatches)))
		}
	}
	if pane.WrapContent {
		parts = append(parts, "wrap")
	}
	if pane.ShowLineNumbers && (pane.Mode == modeLogs || pane.Mode == modeCode) {
		parts = append(parts, "lines")
	}
	if len(parts) == 0 {
		return " "
	}
	return strings.Join(parts, "  ")
}

func (m model) renderDetailPane(pane logPane, active bool, width, height int) string {
	title := logPaneTitle(pane)
	if active {
		title = selectedStyle.Render(truncate(title, width))
	} else {
		title = metaStyle.Render(truncate(title, width))
	}
	var b strings.Builder
	b.WriteString(title + "\n")
	statusPane := pane
	statusPane.Loading = m.paneIsLoading(pane, active)
	b.WriteString(dimStyle.Render(truncate(logPaneStatus(statusPane), width)) + "\n")
	detail := pane.Detail
	if active {
		detail = m.detail
	}
	if detail == nil {
		b.WriteString(dimStyle.Render("loading pipeline...") + "\n")
		return m.renderPaneBox(b.String(), active, m.paneIsLoading(pane, active), width, height)
	}
	p := detail.Pipeline
	fmt.Fprintf(&b, "%s  %s\n", boldStyle.Render(fmt.Sprintf("Pipeline #%d", p.ID)), statusStyle(p.Status).Render(p.Status))
	if p.CommitTitle != "" {
		fmt.Fprintf(&b, "%s %s\n", dimStyle.Render("title"), boldStyle.Render(truncate(p.CommitTitle, max(1, width-7))))
	}
	branchMeta := fmt.Sprintf("  %s %s", dimStyle.Render("@"), shortSHA(p.SHA))
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, dimStyle.Render("branch")+" ", branchBadge(p.Ref, max(5, width-20)), branchMeta) + "\n")
	succeeded := 0
	for _, j := range detail.DisplayJobs {
		if strings.HasPrefix(combinedStatus(j), "success") {
			succeeded++
		}
	}
	fmt.Fprintf(&b, "%s %d/%d\n", dimStyle.Render("jobs succeeded:"), succeeded, len(detail.DisplayJobs))
	now := time.Now()
	for i := range detail.DisplayJobs {
		row := detail.DisplayJobs[i]
		j := row.Current
		progress := renderJobProgress(j, m.jobDurations[j.Name].Average, now, max(1, width-6))
		lastRun := renderJobLastRun(row, now)
		if i == pane.JobsCursor {
			details := renderCombinedStatus(row) + "  " + truncate(j.Stage, 16)
			nameWidth := max(1, min(24, width-lipgloss.Width(details)-6))
			name := lipgloss.NewStyle().Width(nameWidth + 2).Render(selectedStyle.Render(truncate(j.Name, nameWidth)))
			line := lipgloss.JoinHorizontal(lipgloss.Center, name, details)
			if lastRun != "" {
				line += "\n" + dimStyle.Render(truncate(lastRun, max(1, width-6)))
			}
			if progress != "" {
				line += "\n" + truncate(progress, max(1, width-6))
			}
			b.WriteString(jobRowCard(line, max(5, width), true) + "\n")
		} else {
			details := renderCombinedStatus(row) + "  " + truncate(j.Stage, 16)
			nameWidth := max(1, min(24, width-lipgloss.Width(details)-6))
			name := lipgloss.NewStyle().Width(nameWidth + 2).Render(cyanStyle.Bold(true).Render(truncate(j.Name, nameWidth)))
			line := lipgloss.JoinHorizontal(lipgloss.Center, name, details)
			if lastRun != "" {
				line += "\n" + dimStyle.Render(truncate(lastRun, max(1, width-6)))
			}
			if progress != "" {
				line += "\n" + truncate(progress, max(1, width-6))
			}
			b.WriteString(jobRowCard(line, max(5, width), false) + "\n")
		}
		b.WriteString(m.renderInlineLogLines(j, width, pane.ShowInlineLogs))
	}
	body := renderScrollableBody(b.String(), width, height, pane.ScrollOffset, pane.HorizontalOffset, pane.WrapContent)
	return m.renderPaneBox(body, active, m.paneIsLoading(pane, active), width, height)
}

func (m model) renderPipelinePane(pane logPane, active bool, width, height int) string {
	title := logPaneTitle(pane)
	if active {
		title = selectedStyle.Render(truncate(title, width))
	} else {
		title = metaStyle.Render(truncate(title, width))
	}
	var b strings.Builder
	b.WriteString(title + "\n")
	statusPane := pane
	statusPane.Loading = m.paneIsLoading(pane, active)
	b.WriteString(dimStyle.Render(truncate(logPaneStatus(statusPane), width)) + "\n")
	if m.loadingList && len(m.list) == 0 {
		b.WriteString(dimStyle.Render("loading pipelines...") + "\n")
		return m.renderPaneBox(b.String(), active, m.paneIsLoading(pane, active), width, height)
	}
	if len(m.list) == 0 {
		b.WriteString(yellowStyle.Render("no pipelines found") + "\n")
		return m.renderPaneBox(b.String(), active, m.paneIsLoading(pane, active), width, height)
	}
	b.WriteString(dimStyle.Render(fmt.Sprintf("%-10s %-16s %-24s %-10s %-14s %-12s %-11s %s", "ID", "STATUS", "REF", "SHA", "SOURCE", "STARTED", "DURATION", "TITLE")) + "\n")
	now := time.Now()
	cursor := pane.ListCursor
	if active {
		cursor = m.listCursor
	}
	if cursor >= len(m.list) {
		cursor = len(m.list) - 1
	}
	if cursor < 0 {
		cursor = 0
	}
	for i := range m.list {
		p := m.list[i]
		sha := p.SHA
		if len(sha) > 8 {
			sha = sha[:8]
		}
		line := fmt.Sprintf("%-10s %-16s %-24s %-10s %-14s %-12s %-11s %s", fmt.Sprintf("#%d", p.ID), stripStatus(p.Status), truncate(p.Ref, 24), sha, truncate(p.Source, 14), timeAgo(p.StartedAt, now), formatPipelineDuration(p.Duration), truncate(p.CommitTitle, 72))
		if i == cursor {
			line = colorStatusInSelectedLine(line, p.Status)
			b.WriteString(selectedStyle.Render(line) + "\n")
		} else {
			line = colorStatusInLine(line, p.Status)
			b.WriteString(line + "\n")
		}
	}
	body := renderScrollableBody(b.String(), width, height, pane.ScrollOffset, pane.HorizontalOffset, pane.WrapContent)
	return m.renderPaneBox(body, active, m.paneIsLoading(pane, active), width, height)
}

func (m model) renderPaneBox(body string, active bool, loading bool, width, height int) string {
	borderColor := paneBorderColor
	border := lipgloss.RoundedBorder()
	if active {
		border = activePaneBorder()
	}
	if active {
		borderColor = paneBorderActiveColor
	} else if loading {
		borderColor = paneBorderLoadingColor
	}
	indicator := m.activityIndicator()
	if !loading {
		indicator = strings.Repeat(" ", ansi.StringWidth(indicator))
	}
	body = renderTopLeftIndicator(body, indicator, width)
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(border).
		BorderForeground(borderColor).
		Render(body)
}

func renderTopLeftIndicator(body, indicator string, width int) string {
	indicatorWidth := ansi.StringWidth(indicator)
	if width <= 0 || indicatorWidth <= 0 {
		return body
	}
	parts := strings.SplitN(body, "\n", 2)
	if indicatorWidth >= width {
		parts[0] = truncate(indicator, width)
	} else {
		parts[0] = indicator + " " + truncate(parts[0], width-indicatorWidth-1)
	}
	return strings.Join(parts, "\n")
}

func renderScrollableBody(body string, width, height, offset, horizontalOffset int, wrap bool) string {
	if wrap {
		body = wrapContent(body, width, height)
		horizontalOffset = 0
	}
	lines := strings.Split(strings.TrimRight(body, "\n"), "\n")
	maxWidth := 0
	for _, line := range lines {
		maxWidth = max(maxWidth, ansi.StringWidth(line))
	}
	hasHorizontal := maxWidth > width
	contentHeight := height
	if hasHorizontal {
		contentHeight = max(1, height-1)
	}
	hasVertical := len(lines) > contentHeight
	contentWidth := width
	if hasVertical {
		contentWidth = max(1, width-1)
	}
	maxHorizontalOffset := max(0, maxWidth-contentWidth)
	horizontalOffset = min(max(0, horizontalOffset), maxHorizontalOffset)
	maxOffset := max(0, len(lines)-contentHeight)
	offset = min(max(0, offset), maxOffset)
	if !hasHorizontal && !hasVertical {
		return body
	}

	var b strings.Builder
	end := min(len(lines), offset+contentHeight)
	for row := 0; row < contentHeight; row++ {
		line := ""
		if offset+row < end {
			line = lines[offset+row]
			if horizontalOffset > 0 {
				line = cellSlice(ansi.Strip(line), horizontalOffset, horizontalOffset+contentWidth)
			} else {
				line = truncate(line, contentWidth)
			}
		}
		line += strings.Repeat(" ", max(0, contentWidth-ansi.StringWidth(line)))
		if hasVertical {
			thumbHeight := max(1, contentHeight*contentHeight/len(lines))
			thumbStart := 0
			if maxOffset > 0 {
				thumbStart = offset * (contentHeight - thumbHeight) / maxOffset
			}
			bar := dimStyle.Render("│")
			if row >= thumbStart && row < thumbStart+thumbHeight {
				bar = cyanStyle.Render("█")
			}
			line += bar
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	if hasHorizontal {
		barWidth := width
		thumbWidth := max(1, barWidth*contentWidth/maxWidth)
		thumbStart := 0
		if maxHorizontalOffset > 0 {
			thumbStart = horizontalOffset * (barWidth - thumbWidth) / maxHorizontalOffset
		}
		for column := 0; column < barWidth; column++ {
			if column >= thumbStart && column < thumbStart+thumbWidth {
				b.WriteString(cyanStyle.Render("█"))
			} else {
				b.WriteString(dimStyle.Render("─"))
			}
		}
	} else {
		return strings.TrimSuffix(b.String(), "\n")
	}
	return b.String()
}

func widestLineWidth(value string) int {
	width := 0
	for _, line := range strings.Split(value, "\n") {
		width = max(width, ansi.StringWidth(line))
	}
	return width
}

func renderViewportBody(body string, width, height, offset, total int) string {
	if total <= height || height <= 0 || width <= 0 {
		return body
	}
	lines := strings.Split(body, "\n")
	thumbHeight := max(1, height*height/total)
	maxOffset := max(1, total-height)
	thumbStart := offset * (height - thumbHeight) / maxOffset
	var b strings.Builder
	for row := 0; row < height; row++ {
		line := ""
		if row < len(lines) {
			line = truncate(lines[row], max(0, width-1))
		}
		line += strings.Repeat(" ", max(0, width-1-ansi.StringWidth(line)))
		bar := dimStyle.Render("│")
		if row >= thumbStart && row < thumbStart+thumbHeight {
			bar = cyanStyle.Render("█")
		}
		b.WriteString(line + bar)
		if row < height-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
