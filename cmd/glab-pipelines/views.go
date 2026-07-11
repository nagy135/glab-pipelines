package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"
)

func (m model) View() string {
	switch m.mode {
	case modePipelines:
		return m.fillScreen(m.viewPipelines())
	case modeDetail:
		return m.fillScreen(m.viewDetail())
	case modeJobs:
		return m.fillScreen(m.viewJobs())
	case modeConfirm:
		return m.fillScreen(m.viewConfirm())
	case modeLogs:
		return m.fillScreen(m.viewLogs())
	case modeTheme:
		return m.viewTheme()
	}
	return ""
}

func (m model) headerLine(line string) string {
	if m.width <= 0 {
		return line
	}
	width := lipgloss.Width(line)
	if width >= m.width {
		return line
	}
	return line + strings.Repeat(" ", m.width-width)
}

func (m model) viewTheme() string {
	width := m.width
	if width <= 0 {
		width = 80
	}
	height := m.height
	if height <= 0 {
		height = 24
	}
	panelWidth := min(68, max(36, width-6))
	contentWidth := max(1, panelWidth-4)
	listRows := min(len(themeOptions), max(3, height-8))
	start := m.themeCursor - listRows/2
	if start < 0 {
		start = 0
	}
	if start+listRows > len(themeOptions) {
		start = max(0, len(themeOptions)-listRows)
	}
	end := min(len(themeOptions), start+listRows)

	var b strings.Builder
	b.WriteString(m.headerLine(boldStyle.Render("Theme")+" "+metaPill("current", m.themeName)) + "\n")
	b.WriteString(m.headerLine(hintBar(keyHint("j/k", "move"), keyHint("enter", "apply"), keyHint("q", "back"))) + "\n")
	if m.message != "" {
		b.WriteString(yellowStyle.Render(m.message) + "\n")
	}
	if start > 0 {
		b.WriteString(dimStyle.Render(truncate("...", contentWidth)) + "\n")
	}
	for i := start; i < end; i++ {
		theme := themeOptions[i]
		mark := " "
		if theme.Name == m.themeName {
			mark = "*"
		}
		line := fmt.Sprintf("%s %-18s %s", mark, theme.Name, theme.Description)
		line = truncate(line, contentWidth)
		if i == m.themeCursor {
			b.WriteString(selectedStyle.Render(line) + "\n")
		} else {
			b.WriteString(line + "\n")
		}
	}
	if end < len(themeOptions) {
		b.WriteString(dimStyle.Render(truncate("...", contentWidth)) + "\n")
	}

	panel := lipgloss.NewStyle().
		Width(contentWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(paneBorderActiveColor).
		Render(strings.TrimRight(b.String(), "\n"))
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, panel)
}

func (m model) viewPipelines() string {
	var b strings.Builder
	label := m.status
	if label == "active" {
		label = "active"
	}
	b.WriteString(m.headerLine(breadcrumbs("Pipelines")+" "+metaPill("status", label)+" "+metaPill("limit", fmt.Sprintf("%d", m.limit))) + "\n")
	if m.repo != "" {
		b.WriteString(m.headerLine(metaPill("repo", m.repo)) + "\n")
	}
	b.WriteString(m.headerLine(hintBar(keyHint("j/k", "move"), keyHint("ctrl+f/b", "page"), keyHint("enter", "details"), keyHint("s/v", "split"), keyHint("ctrl+hjkl", "focus"), keyHint("x", "close"), keyHint("o", "only"), keyHint("t", "theme"), keyHint("b", "border"), keyHint("r", "refresh"), keyHint("q", "close/quit"))) + "\n")
	if len(m.logPanes) > 1 {
		b.WriteString(m.viewLogSplits())
		return b.String()
	}
	var body strings.Builder
	if m.message != "" {
		body.WriteString(yellowStyle.Render(m.message) + "\n")
	}
	if m.loadingList && len(m.list) == 0 {
		body.WriteString(dimStyle.Render("loading pipelines...") + "\n")
		b.WriteString(m.renderSinglePane(body.String()))
		return b.String()
	}
	if len(m.list) == 0 {
		body.WriteString(yellowStyle.Render("no pipelines found") + "\n")
		b.WriteString(m.renderSinglePane(body.String()))
		return b.String()
	}
	body.WriteString(dimStyle.Render(fmt.Sprintf("%-10s %-16s %-24s %-10s %-14s %-19s %-11s %s", "ID", "STATUS", "REF", "SHA", "SOURCE", "UPDATED", "DURATION", "TITLE")) + "\n")
	for i, p := range m.list {
		sha := p.SHA
		if len(sha) > 8 {
			sha = sha[:8]
		}
		line := fmt.Sprintf("%-10s %-16s %-24s %-10s %-14s %-19s %-11s %s", fmt.Sprintf("#%d", p.ID), stripStatus(p.Status), truncate(p.Ref, 24), sha, truncate(p.Source, 14), shortTime(p.UpdatedOrCreated()), formatPipelineDuration(p.Duration), truncate(p.CommitTitle, 72))
		if i == m.listCursor {
			line = colorStatusInSelectedLine(line, p.Status)
			body.WriteString(selectedStyle.Render(line) + "\n")
		} else {
			line = colorStatusInLine(line, p.Status)
			body.WriteString(line + "\n")
		}
	}
	b.WriteString(m.renderSinglePane(body.String()))
	return b.String()
}

func (m model) viewDetail() string {
	var b strings.Builder
	title := breadcrumbs("Pipelines", fmt.Sprintf("Pipeline #%d", m.detailID)) + " " + metaPill("refresh", m.refresh.String())
	inlineHint := "show inline"
	if m.showInlineLogs {
		title += " " + metaPill("inline logs", "on")
		inlineHint = "hide inline"
	}
	b.WriteString(m.headerLine(title) + "\n")
	b.WriteString(m.headerLine(hintBar(keyHint("j/k", "jobs"), keyHint("ctrl+f/b", "page"), keyHint("s/v", "split"), keyHint("ctrl+hjkl", "focus"), keyHint("x", "close"), keyHint("o", "only"), keyHint("t", "theme"), keyHint("b", "border"), keyHint("l", "logs"), keyHint("L", inlineHint), keyHint("S", "start/retry"), keyHint("c", "cancel"), keyHint("r", "refresh"), keyHint("q", "close"), keyHint("esc", "back"))) + "\n")
	if len(m.logPanes) > 1 {
		b.WriteString(m.viewLogSplits())
		return b.String()
	}
	var body strings.Builder
	if m.message != "" {
		body.WriteString(yellowStyle.Render(m.message) + "\n")
	}
	if m.detail == nil {
		body.WriteString(dimStyle.Render(fmt.Sprintf("loading pipeline #%d...", m.detailID)) + "\n")
		b.WriteString(m.renderSinglePane(body.String()))
		return b.String()
	}
	p := m.detail.Pipeline
	fmt.Fprintf(&body, "%s  %s\n", boldStyle.Render(fmt.Sprintf("Pipeline #%d", p.ID)), statusStyle(p.Status).Render(p.Status))
	if p.CommitTitle != "" {
		fmt.Fprintf(&body, "%s %s\n", dimStyle.Render("title"), boldStyle.Render(truncate(p.CommitTitle, max(1, m.width-10))))
	}
	branchMeta := fmt.Sprintf("  %s %s   %s %s", dimStyle.Render("@"), shortSHA(p.SHA), dimStyle.Render("created"), shortTime(p.CreatedAt))
	body.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, dimStyle.Render("branch")+" ", branchBadge(p.Ref, max(5, m.width-42)), branchMeta) + "\n")
	body.WriteString(mutedStyle.Render(p.WebURL) + "\n\n")
	succeeded := 0
	for _, j := range m.detail.DisplayJobs {
		if strings.HasPrefix(combinedStatus(j), "success") {
			succeeded++
		}
	}
	fmt.Fprintf(&body, "%s %d/%d\n\n", dimStyle.Render("jobs succeeded:"), succeeded, len(m.detail.DisplayJobs))
	stages := orderedDisplayStages(m.detail.DisplayJobs)
	for _, stage := range stages {
		body.WriteString(blueStyle.Bold(true).Render(stage) + "\n")
		for i, row := range m.detail.DisplayJobs {
			j := row.Current
			if j.Stage != stage {
				continue
			}
			allow := ""
			if j.Status == "failed" && j.AllowFailure {
				allow = dimStyle.Render(" (allowed to fail)")
			}
			progress := renderJobProgress(j, m.jobDurations[j.Name].Average, time.Now(), max(1, m.width-10))
			if i == m.jobsCursor {
				name := lipgloss.NewStyle().Width(34).Render(selectedStyle.Render(truncate(j.Name, 32)))
				line := lipgloss.JoinHorizontal(lipgloss.Center, name, renderCombinedStatus(row), "  ", dimStyle.Render(formatDuration(displayDuration(row))), allow)
				if progress != "" {
					line += "\n" + truncate(progress, max(1, m.width-10))
				}
				body.WriteString(jobRowCard(line, max(8, m.width-4), true) + "\n")
			} else {
				name := lipgloss.NewStyle().Width(34).Render(cyanStyle.Bold(true).Render(truncate(j.Name, 32)))
				line := lipgloss.JoinHorizontal(lipgloss.Center, name, renderCombinedStatus(row), "  ", dimStyle.Render(formatDuration(displayDuration(row))), allow)
				if progress != "" {
					line += "\n" + truncate(progress, max(1, m.width-10))
				}
				body.WriteString(jobRowCard(line, max(8, m.width-4), false) + "\n")
			}
			body.WriteString(m.renderInlineLogLines(j, m.width-4, m.showInlineLogs))
		}
		body.WriteString("\n")
	}
	b.WriteString(m.renderSinglePane(body.String()))
	return b.String()
}

func (m model) renderInlineLogLines(j job, width int, show bool) string {
	if !show || !supportsInlineLogs(j) {
		return ""
	}
	if width <= 0 {
		width = 120
	}
	const prefix = "      | "
	snippet, loaded := m.inlineLogs[j.ID]
	if !loaded || snippet.Status != j.Status {
		if m.inlineLogsLoading[j.ID] {
			return dimStyle.Render(truncate(prefix+"loading latest log lines...", width)) + "\n"
		}
		return ""
	}
	if len(snippet.Lines) == 0 {
		return dimStyle.Render(truncate(prefix+"(empty log)", width)) + "\n"
	}
	var b strings.Builder
	for _, line := range snippet.Lines {
		line = strings.ReplaceAll(line, "\t", "    ")
		b.WriteString(dimStyle.Render(truncate(prefix+line, width)) + "\n")
	}
	return b.String()
}

func (m model) viewJobs() string {
	var b strings.Builder
	b.WriteString(m.headerLine(breadcrumbs("Pipelines", fmt.Sprintf("Pipeline #%d", m.detailID), "Jobs")) + "\n")
	b.WriteString(m.headerLine(hintBar(keyHint("j/k", "move"), keyHint("ctrl+f/b", "page"), keyHint("s", "start/retry"), keyHint("c", "cancel"), keyHint("l", "logs"), keyHint("t", "theme"), keyHint("b", "border"), keyHint("r", "refresh"), keyHint("q", "back"))) + "\n")
	var body strings.Builder
	if m.message != "" {
		body.WriteString(yellowStyle.Render(m.message) + "\n")
	}
	if m.detail == nil || len(m.detail.DisplayJobs) == 0 {
		body.WriteString(dimStyle.Render("no jobs loaded") + "\n")
		b.WriteString(m.renderSinglePane(body.String()))
		return b.String()
	}
	for i, row := range m.detail.DisplayJobs {
		j := row.Current
		progress := renderJobProgress(j, m.jobDurations[j.Name].Average, time.Now(), max(1, m.width-4))
		keys := availableKeys(j)
		if logTarget(row).ID != j.ID {
			if keys == "-" {
				keys = ""
			}
			keys = strings.TrimSpace(keys + " l:prev-log")
		}
		if i == m.jobsCursor {
			line := fmt.Sprintf("%-26s %-24s %-18s %s", combinedStatusText(row), keys, truncate(j.Stage, 18), j.Name)
			if progress != "" {
				line += "\n  " + truncate(progress, max(1, m.width-4))
			}
			line = colorCombinedStatusInSelectedLine(line, row)
			body.WriteString(selectedStyle.Render(line) + "\n")
		} else {
			line := fmt.Sprintf("%-26s %-24s %-18s %s", combinedStatusText(row), keys, truncate(j.Stage, 18), j.Name)
			if progress != "" {
				line += "\n  " + truncate(progress, max(1, m.width-4))
			}
			line = colorCombinedStatusInLine(line, row)
			body.WriteString(line + "\n")
		}
	}
	b.WriteString(m.renderSinglePane(body.String()))
	return b.String()
}

func (m model) viewLogs() string {
	var b strings.Builder
	name := "unknown job"
	status := ""
	if m.logJob != nil {
		name = m.logJob.Name
		status = m.logJob.Status
	}
	b.WriteString(m.headerLine(breadcrumbs("Pipelines", fmt.Sprintf("Pipeline #%d", m.detailID), "Jobs", "Logs")+" "+metaPill("job", truncate(name, 28))+" "+metaPill("status", status)+" "+metaPill("live", m.logRefresh.String())) + "\n")
	nHint := keyHint("n", "next job")
	if m.logSearchActive {
		nHint = keyHint("n/N", "match")
	}
	b.WriteString(m.headerLine(hintBar(keyHint("j/k", "scroll"), keyHint("ctrl+f/b", "page"), keyHint("s/v", "split"), keyHint("ctrl+hjkl", "focus"), keyHint("x", "close"), keyHint("o", "only"), keyHint("/", "search"), keyHint("t", "theme"), keyHint("b", "border"), nHint, keyHint("r", "reload"), keyHint("q", "close"), keyHint("esc", "back"))) + "\n")
	if len(m.logPanes) > 1 {
		b.WriteString(m.viewLogSplits())
		return b.String()
	}
	var body strings.Builder
	if m.message != "" {
		body.WriteString(yellowStyle.Render(m.message) + "\n")
	}
	if m.logsLoading {
		body.WriteString(dimStyle.Render("loading...") + "\n")
	}
	if status := m.logSearchStatus(); status != "" {
		body.WriteString(status + "\n")
	}
	body.WriteString("\n")
	body.WriteString(renderViewportBody(m.logsViewport.View(), m.logsViewport.Width, m.logsViewport.Height, m.logsViewport.YOffset, m.logsViewport.TotalLineCount()))
	b.WriteString(m.renderSinglePane(body.String()))
	return b.String()
}

func (m model) viewConfirm() string {
	if m.pending == nil {
		return ""
	}
	backgroundModel := m
	backgroundModel.mode = m.confirmBackMode
	var background string
	if m.confirmBackMode == modeJobs {
		background = backgroundModel.viewJobs()
	} else {
		background = backgroundModel.viewDetail()
	}

	var b strings.Builder
	a := *m.pending
	b.WriteString(boldStyle.Render(a.Verb+" job?") + "\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("#%d", a.Job.ID)) + "  " + cyanStyle.Render(a.Job.Name) + "\n\n")
	prodPlay := strings.Contains(strings.ToLower(a.Job.Name), "prod") && a.Endpoint == "play"
	if prodPlay {
		b.WriteString(redStyle.Bold(true).Render("Production deploy protection") + "\n")
		b.WriteString("Type the exact job name to confirm:\n")
		b.WriteString(boldStyle.Render(a.Job.Name) + "\n")
		b.WriteString(cyanStyle.Render(m.confirmText+"_") + "\n\n")
		b.WriteString(hintBar(keyHint("enter", "confirm"), keyHint("esc", "cancel")))
		if m.message != "" {
			b.WriteString("\n" + yellowStyle.Render(m.message))
		}
	} else {
		b.WriteString("Send this action to GitLab?\n\n")
		b.WriteString(hintBar(keyHint("y", "confirm"), keyHint("n", "cancel")))
	}
	panelWidth := 62
	if m.width > 0 {
		panelWidth = min(panelWidth, m.width)
	}
	padding := 1
	if panelWidth < 8 {
		padding = 0
	}
	contentWidth := max(1, panelWidth-2-(2*padding))
	panel := lipgloss.NewStyle().
		Width(contentWidth).
		Padding(padding).
		Border(activePaneBorder()).
		BorderForeground(paneBorderActiveColor).
		Render(b.String())
	return overlayCentered(background, panel, m.width, m.height)
}

func overlayCentered(background, foreground string, width, height int) string {
	if width <= 0 || height <= 0 {
		return foreground
	}
	foregroundWidth := min(width, lipgloss.Width(foreground))
	foregroundLines := strings.Split(foreground, "\n")
	foregroundHeight := min(height, len(foregroundLines))
	x := max(0, (width-foregroundWidth)/2)
	y := max(0, (height-foregroundHeight)/2)

	backgroundLines := strings.Split(ansi.Strip(background), "\n")
	lines := make([]string, height)
	for row := range lines {
		base := ""
		if row < len(backgroundLines) {
			base = backgroundLines[row]
		}
		base = cellSlice(base, 0, width)
		if row < y || row >= y+foregroundHeight {
			lines[row] = base
			continue
		}
		modalLine := foregroundLines[row-y]
		modalLine = truncate(modalLine, foregroundWidth)
		modalLine += strings.Repeat(" ", max(0, foregroundWidth-ansi.StringWidth(modalLine)))
		lines[row] = cellSlice(base, 0, x) + modalLine + cellSlice(base, x+foregroundWidth, width)
	}
	return strings.Join(lines, "\n")
}

func cellSlice(value string, start, end int) string {
	if end <= start {
		return ""
	}
	var b strings.Builder
	position := 0
	graphemes := uniseg.NewGraphemes(value)
	for graphemes.Next() {
		width := graphemes.Width()
		next := position + width
		if next <= start {
			position = next
			continue
		}
		if position >= end {
			break
		}
		if position < start || next > end {
			visible := min(next, end) - max(position, start)
			b.WriteString(strings.Repeat(" ", max(0, visible)))
		} else {
			b.WriteString(graphemes.Str())
		}
		position = next
	}
	b.WriteString(strings.Repeat(" ", max(0, end-start-uniseg.StringWidth(b.String()))))
	return b.String()
}
