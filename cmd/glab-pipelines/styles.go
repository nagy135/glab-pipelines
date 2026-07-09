package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	boldStyle        = lipgloss.NewStyle().Bold(true)
	dimStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	mutedStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	cyanStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	greenStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	yellowStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	redStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	blueStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	selectedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(lipgloss.Color("24")).Bold(true)
	crumbStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Background(lipgloss.Color("236")).Padding(0, 1).Bold(true)
	activeCrumbStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("45")).Padding(0, 1).Bold(true)
	crumbSepStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Background(lipgloss.Color("234")).Padding(0, 1)
	hintStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Background(lipgloss.Color("236")).Padding(0, 1)
	keyStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("220")).Padding(0, 1).Bold(true)
	metaStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("235")).Padding(0, 1)
)

func statusStyle(status string) lipgloss.Style {
	switch status {
	case "success", "ran":
		return greenStyle
	case "failed":
		return redStyle
	case "running", "pending", "waiting_for_resource":
		return yellowStyle
	case "manual", "scheduled":
		return blueStyle
	case "preparing":
		return cyanStyle
	default:
		return mutedStyle
	}
}

func stripStatus(status string) string { return "[" + status + "]" }

func colorStatusInLine(line, status string) string {
	needle := stripStatus(status)
	return strings.Replace(line, needle, statusStyle(status).Render(needle), 1)
}

func breadcrumbs(labels ...string) string {
	if len(labels) == 0 {
		return ""
	}
	parts := make([]string, 0, len(labels)*2-1)
	for i, label := range labels {
		style := crumbStyle
		if i == len(labels)-1 {
			style = activeCrumbStyle
		}
		parts = append(parts, style.Render(label))
		if i < len(labels)-1 {
			parts = append(parts, crumbSepStyle.Render("›"))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func keyHint(key, text string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top, keyStyle.Render(key), hintStyle.Render(" "+text))
}

func hintBar(items ...string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top, items...)
}

func metaPill(label, value string) string {
	return metaStyle.Render(label + " " + value)
}
