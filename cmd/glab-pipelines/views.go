package main

import (
	"fmt"
	"strings"
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
	}
	return ""
}

func (m model) viewPipelines() string {
	var b strings.Builder
	label := m.status
	if label == "active" {
		label = "active"
	}
	b.WriteString(breadcrumbs("Pipelines") + " " + metaPill("status", label) + " " + metaPill("limit", fmt.Sprintf("%d", m.limit)) + "\n")
	if m.repo != "" {
		b.WriteString(metaPill("repo", m.repo) + "\n")
	}
	b.WriteString(hintBar(keyHint("j/k", "move"), keyHint("enter", "details"), keyHint("r", "refresh"), keyHint("q", "quit")) + "\n")
	if m.message != "" {
		b.WriteString(yellowStyle.Render(m.message) + "\n")
	}
	b.WriteString("\n")
	if m.loadingList && len(m.list) == 0 {
		b.WriteString(dimStyle.Render("loading pipelines...") + "\n")
		return b.String()
	}
	if len(m.list) == 0 {
		b.WriteString(yellowStyle.Render("no pipelines found") + "\n")
		return b.String()
	}
	b.WriteString(dimStyle.Render(fmt.Sprintf("%-10s %-16s %-24s %-10s %-14s %-16s %s", "ID", "STATUS", "REF", "SHA", "SOURCE", "UPDATED", "TITLE")) + "\n")
	for i, p := range m.list {
		sha := p.SHA
		if len(sha) > 8 {
			sha = sha[:8]
		}
		line := fmt.Sprintf("%-10s %-16s %-24s %-10s %-14s %-16s %s", fmt.Sprintf("#%d", p.ID), stripStatus(p.Status), truncate(p.Ref, 24), sha, truncate(p.Source, 14), shortTime(p.UpdatedOrCreated()), truncate(p.CommitTitle, 72))
		line = colorStatusInLine(line, p.Status)
		if i == m.listCursor {
			b.WriteString(selectedStyle.Render(line) + "\n")
		} else {
			b.WriteString(line + "\n")
		}
	}
	return b.String()
}

func (m model) viewDetail() string {
	var b strings.Builder
	b.WriteString(breadcrumbs("Pipelines", fmt.Sprintf("Pipeline #%d", m.detailID)) + " " + metaPill("refresh", m.refresh.String()) + "\n")
	b.WriteString(hintBar(keyHint("j/k", "jobs"), keyHint("p", "jobs list"), keyHint("s", "start/retry"), keyHint("c", "cancel"), keyHint("l", "logs"), keyHint("r", "refresh"), keyHint("q", "back"), keyHint("o", "open")) + "\n")
	if m.message != "" {
		b.WriteString(yellowStyle.Render(m.message) + "\n")
	}
	b.WriteString("\n")
	if m.detail == nil {
		b.WriteString(dimStyle.Render(fmt.Sprintf("loading pipeline #%d...", m.detailID)) + "\n")
		return b.String()
	}
	p := m.detail.Pipeline
	fmt.Fprintf(&b, "%s  %s\n", boldStyle.Render(fmt.Sprintf("Pipeline #%d", p.ID)), statusStyle(p.Status).Render(p.Status))
	fmt.Fprintf(&b, "%s %s %s %s   %s %s\n", dimStyle.Render("ref"), cyanStyle.Render(p.Ref), dimStyle.Render("@"), shortSHA(p.SHA), dimStyle.Render("created"), shortTime(p.CreatedAt))
	b.WriteString(mutedStyle.Render(p.WebURL) + "\n\n")
	succeeded := 0
	for _, j := range m.detail.DisplayJobs {
		if strings.HasPrefix(combinedStatus(j), "success") {
			succeeded++
		}
	}
	fmt.Fprintf(&b, "%s %d/%d\n\n", dimStyle.Render("jobs succeeded:"), succeeded, len(m.detail.DisplayJobs))
	stages := orderedDisplayStages(m.detail.DisplayJobs)
	for _, stage := range stages {
		b.WriteString(blueStyle.Bold(true).Render(stage) + "\n")
		for i, row := range m.detail.DisplayJobs {
			j := row.Current
			if j.Stage != stage {
				continue
			}
			allow := ""
			if j.Status == "failed" && j.AllowFailure {
				allow = dimStyle.Render(" (allowed to fail)")
			}
			line := fmt.Sprintf("  %-32s %-26s %s%s", truncate(j.Name, 32), renderCombinedStatus(row), dimStyle.Render(formatDuration(displayDuration(row))), allow)
			if i == m.jobsCursor {
				b.WriteString(selectedStyle.Render(line) + "\n")
			} else {
				b.WriteString(line + "\n")
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (m model) viewJobs() string {
	var b strings.Builder
	b.WriteString(breadcrumbs("Pipelines", fmt.Sprintf("Pipeline #%d", m.detailID), "Jobs") + "\n")
	b.WriteString(hintBar(keyHint("j/k", "move"), keyHint("s", "start/retry"), keyHint("c", "cancel"), keyHint("l", "logs"), keyHint("r", "refresh"), keyHint("q", "back")) + "\n")
	if m.message != "" {
		b.WriteString(yellowStyle.Render(m.message) + "\n")
	}
	b.WriteString("\n")
	if m.detail == nil || len(m.detail.DisplayJobs) == 0 {
		b.WriteString(dimStyle.Render("no jobs loaded") + "\n")
		return b.String()
	}
	for i, row := range m.detail.DisplayJobs {
		j := row.Current
		keys := availableKeys(j)
		if logTarget(row).ID != j.ID {
			if keys == "-" {
				keys = ""
			}
			keys = strings.TrimSpace(keys + " l:prev-log")
		}
		line := fmt.Sprintf("%-26s %-24s %-18s %s", combinedStatusText(row), keys, truncate(j.Stage, 18), j.Name)
		line = colorCombinedStatusInLine(line, row)
		if i == m.jobsCursor {
			b.WriteString(selectedStyle.Render(line) + "\n")
		} else {
			b.WriteString(line + "\n")
		}
	}
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
	b.WriteString(breadcrumbs("Pipelines", fmt.Sprintf("Pipeline #%d", m.detailID), "Jobs", "Logs") + " " + metaPill("job", truncate(name, 28)) + " " + metaPill("status", status) + " " + metaPill("live", m.logRefresh.String()) + "\n")
	nHint := keyHint("n", "next job")
	if m.logSearchActive {
		nHint = keyHint("n/N", "match")
	}
	b.WriteString(hintBar(keyHint("j/k", "scroll"), keyHint("pgup/pgdn", "page"), keyHint("g/G", "top/bottom"), keyHint("/", "search"), nHint, keyHint("r", "reload"), keyHint("q", "back")) + "\n")
	if m.message != "" {
		b.WriteString(yellowStyle.Render(m.message) + "\n")
	}
	if m.logsLoading {
		b.WriteString(dimStyle.Render("loading...") + "\n")
	}
	if status := m.logSearchStatus(); status != "" {
		b.WriteString(status + "\n")
	}
	b.WriteString("\n")
	b.WriteString(m.logsViewport.View())
	return b.String()
}

func (m model) viewConfirm() string {
	if m.pending == nil {
		return ""
	}
	var b strings.Builder
	a := *m.pending
	b.WriteString(breadcrumbs("Pipelines", fmt.Sprintf("Pipeline #%d", m.detailID), "Confirm") + " " + metaPill("action", a.Verb) + " " + metaPill("job", fmt.Sprintf("#%d", a.Job.ID)) + "\n")
	b.WriteString(cyanStyle.Render(a.Job.Name) + "\n\n")
	prodPlay := strings.Contains(strings.ToLower(a.Job.Name), "prod") && a.Endpoint == "play"
	if prodPlay {
		b.WriteString(redStyle.Bold(true).Render("Production deploy protection") + "\n")
		b.WriteString(fmt.Sprintf("Type exact job name to confirm: %s\n", boldStyle.Render(a.Job.Name)))
		b.WriteString(cyanStyle.Render(m.confirmText) + "\n")
		b.WriteString(dimStyle.Render("enter confirm  esc cancel") + "\n")
		if m.message != "" {
			b.WriteString(yellowStyle.Render(m.message) + "\n")
		}
		return b.String()
	}
	b.WriteString(fmt.Sprintf("Confirm %s? %s\n", a.Verb, dimStyle.Render("y/n")))
	return b.String()
}
