package main

import "strings"

func resolveAction(key string, j job) (pendingAction, bool) {
	if key == "s" {
		switch j.Status {
		case "manual":
			return pendingAction{Job: j, Endpoint: "play", Verb: "Play"}, true
		case "failed", "success", "canceled", "cancelled":
			return pendingAction{Job: j, Endpoint: "retry", Verb: "Retry"}, true
		}
	}
	if key == "c" {
		switch j.Status {
		case "running", "pending", "created", "preparing", "waiting_for_resource":
			return pendingAction{Job: j, Endpoint: "cancel", Verb: "Cancel"}, true
		}
	}
	return pendingAction{}, false
}

func availableKeys(j job) string {
	var keys []string
	switch j.Status {
	case "manual":
		keys = append(keys, "s:play")
	case "failed", "success", "canceled", "cancelled":
		keys = append(keys, "s:retry")
	}
	switch j.Status {
	case "running", "pending", "created", "preparing", "waiting_for_resource":
		keys = append(keys, "c:cancel")
	}
	if len(keys) == 0 {
		return "-"
	}
	return strings.Join(keys, " ")
}

func buildDisplayJobs(jobs []job) []uiJob {
	byKey := map[string]int{}
	var rows []uiJob
	for _, j := range jobs {
		key := j.Stage + "\x00" + j.Name
		idx, ok := byKey[key]
		if !ok {
			byKey[key] = len(rows)
			rows = append(rows, uiJob{Current: j})
			continue
		}

		row := rows[idx]
		switch {
		case j.Status == "manual":
			row.Previous = betterPrevious(row.Previous, row.Current)
			row.Current = j
		case row.Current.Status == "manual":
			row.Previous = betterPrevious(row.Previous, j)
		case !j.Retried && (row.Current.Retried || j.ID > row.Current.ID):
			row.Previous = betterPrevious(row.Previous, row.Current)
			row.Current = j
		default:
			row.Previous = betterPrevious(row.Previous, j)
		}
		rows[idx] = row
	}
	return rows
}

func betterPrevious(existing *job, candidate job) *job {
	if (candidate.Status == "manual" && !jobHasRun(candidate)) || candidate.Status == "created" {
		return existing
	}
	if existing == nil || candidate.ID > existing.ID {
		c := candidate
		return &c
	}
	return existing
}

func jobHasRun(j job) bool {
	return j.StartedAt != "" || j.FinishedAt != "" || j.Duration != nil
}

func shouldAutoRefreshLogs(j *job) bool {
	return j != nil && j.Status == "running"
}

func augmentPreviousRuns(rows []uiJob, history []job, p pipeline) {
	for i := range rows {
		if rows[i].Current.Status != "manual" {
			continue
		}
		for _, candidate := range history {
			if candidate.ID == rows[i].Current.ID || candidate.Name != rows[i].Current.Name || candidate.Stage != rows[i].Current.Stage || !samePipelineJob(candidate, p) {
				continue
			}
			rows[i].Previous = betterPrevious(rows[i].Previous, candidate)
		}
	}
}

func logTarget(row uiJob) job {
	if row.Current.Status == "manual" && row.Previous != nil {
		return *row.Previous
	}
	return row.Current
}

func displayDuration(row uiJob) *float64 {
	if row.Current.Status == "manual" && row.Previous != nil {
		return row.Previous.Duration
	}
	return row.Current.Duration
}

func combinedStatus(row uiJob) string {
	if row.Current.Status == "manual" && row.Previous != nil {
		return previousStatus(*row.Previous) + " + manual"
	}
	return row.Current.Status
}

func previousStatus(j job) string {
	if j.Status == "manual" && jobHasRun(j) {
		return "ran"
	}
	return j.Status
}

func combinedStatusText(row uiJob) string {
	status := combinedStatus(row)
	if strings.Contains(status, " + ") {
		parts := strings.Split(status, " + ")
		for i, part := range parts {
			parts[i] = stripStatus(part)
		}
		return strings.Join(parts, "+")
	}
	return stripStatus(status)
}

func renderCombinedStatus(row uiJob) string {
	if row.Current.Status == "manual" && row.Previous != nil {
		prev := previousStatus(*row.Previous)
		return statusStyle(prev).Render(prev) + dimStyle.Render(" + ") + statusStyle("manual").Render("manual")
	}
	return statusStyle(row.Current.Status).Render(row.Current.Status)
}

func renderSelectedCombinedStatus(row uiJob) string {
	if row.Current.Status == "manual" && row.Previous != nil {
		prev := previousStatus(*row.Previous)
		return selectedStatusStyle(prev).Render(prev) + dimStyle.Render(" + ") + selectedStatusStyle("manual").Render("manual")
	}
	return selectedStatusStyle(row.Current.Status).Render(row.Current.Status)
}

func colorCombinedStatusInLine(line string, row uiJob) string {
	if row.Current.Status == "manual" && row.Previous != nil {
		prev := previousStatus(*row.Previous)
		needle := stripStatus(prev) + "+" + stripStatus("manual")
		repl := statusStyle(prev).Render(stripStatus(prev)) + dimStyle.Render("+") + statusStyle("manual").Render(stripStatus("manual"))
		return strings.Replace(line, needle, repl, 1)
	}
	return colorStatusInLine(line, row.Current.Status)
}

func colorCombinedStatusInSelectedLine(line string, row uiJob) string {
	if row.Current.Status == "manual" && row.Previous != nil {
		prev := previousStatus(*row.Previous)
		needle := stripStatus(prev) + "+" + stripStatus("manual")
		repl := selectedStatusStyle(prev).Render(stripStatus(prev)) + dimStyle.Render("+") + selectedStatusStyle("manual").Render(stripStatus("manual"))
		return strings.Replace(line, needle, repl, 1)
	}
	return colorStatusInSelectedLine(line, row.Current.Status)
}

func orderedDisplayStages(jobs []uiJob) []string {
	seen := map[string]bool{}
	var stages []string
	for _, j := range jobs {
		if seen[j.Current.Stage] {
			continue
		}
		seen[j.Current.Stage] = true
		stages = append(stages, j.Current.Stage)
	}
	return stages
}

func detailJobOrder(jobs []uiJob) []int {
	stages := orderedDisplayStages(jobs)
	order := make([]int, 0, len(jobs))
	for _, stage := range stages {
		for i, job := range jobs {
			if job.Current.Stage == stage {
				order = append(order, i)
			}
		}
	}
	return order
}

func moveDetailJobCursor(cursor int, jobs []uiJob, delta int) int {
	order := detailJobOrder(jobs)
	if len(order) == 0 {
		return 0
	}
	pos := 0
	for i, idx := range order {
		if idx == cursor {
			pos = i
			break
		}
	}
	pos = (pos + delta) % len(order)
	if pos < 0 {
		pos += len(order)
	}
	return order[pos]
}
