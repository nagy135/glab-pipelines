package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type stylePalette struct {
	dim               lipgloss.Color
	muted             lipgloss.Color
	cyan              lipgloss.Color
	green             lipgloss.Color
	yellow            lipgloss.Color
	red               lipgloss.Color
	blue              lipgloss.Color
	selectedFG        lipgloss.Color
	selectedBG        lipgloss.Color
	crumbFG           lipgloss.Color
	crumbBG           lipgloss.Color
	activeCrumbFG     lipgloss.Color
	activeCrumbBG     lipgloss.Color
	crumbSepFG        lipgloss.Color
	crumbSepBG        lipgloss.Color
	hintFG            lipgloss.Color
	hintBG            lipgloss.Color
	keyFG             lipgloss.Color
	keyBG             lipgloss.Color
	metaFG            lipgloss.Color
	metaBG            lipgloss.Color
	searchHitFG       lipgloss.Color
	searchHitBG       lipgloss.Color
	statusBadgeFG     lipgloss.Color
	paneBorder        lipgloss.Color
	paneBorderLoading lipgloss.Color
	paneBorderActive  lipgloss.Color
}

type themeOption struct {
	Name        string
	Description string
}

var themeOptions = []themeOption{
	{Name: "default", Description: "built-in terminal palette"},
	{Name: "gruvbox-material", Description: "warm Gruvbox Material colors"},
	{Name: "tokyo-night", Description: "Tokyo Night colors"},
	{Name: "catppuccin", Description: "Catppuccin Mocha colors"},
	{Name: "gruvbox", Description: "Gruvbox Dark colors"},
	{Name: "nord", Description: "cool Nord colors"},
	{Name: "dracula", Description: "classic Dracula colors"},
	{Name: "kanagawa", Description: "muted Japanese-inspired colors"},
	{Name: "everforest", Description: "soft earthy greens"},
	{Name: "rose-pine", Description: "soft rose and pine colors"},
	{Name: "onedark", Description: "Atom One Dark colors"},
	{Name: "solarized-dark", Description: "Solarized dark colors"},
	{Name: "ayu", Description: "Ayu dark colors"},
	{Name: "material", Description: "Material dark colors"},
	{Name: "nightfox", Description: "deep blue Nightfox colors"},
	{Name: "sonokai", Description: "vivid Sonokai colors"},
	{Name: "moonfly", Description: "minimal dark Moonfly colors"},
	{Name: "oceanic-next", Description: "Oceanic Next colors"},
	{Name: "palenight", Description: "Material Palenight colors"},
	{Name: "monokai", Description: "classic Monokai colors"},
	{Name: "papercolor", Description: "light PaperColor colors"},
	{Name: "edge", Description: "Edge dark colors"},
}

var themePalettes = map[string]stylePalette{
	"tokyo-night":    schemeStylePalette("#1a1b26", "#c0caf5", "#f7768e", "#e0af68", "#9ece6a", "#7dcfff", "#7aa2f7", "#bb9af7"),
	"catppuccin":     schemeStylePalette("#1e1e2e", "#cdd6f4", "#f38ba8", "#f9e2af", "#a6e3a1", "#94e2d5", "#89b4fa", "#cba6f7"),
	"gruvbox":        schemeStylePalette("#282828", "#ebdbb2", "#cc241d", "#d79921", "#98971a", "#689d6a", "#458588", "#b16286"),
	"nord":           schemeStylePalette("#2e3440", "#eceff4", "#bf616a", "#ebcb8b", "#a3be8c", "#88c0d0", "#5e81ac", "#b48ead"),
	"dracula":        schemeStylePalette("#282a36", "#f8f8f2", "#ff5555", "#f1fa8c", "#50fa7b", "#8be9fd", "#bd93f9", "#ff79c6"),
	"kanagawa":       schemeStylePalette("#1f1f28", "#dcd7ba", "#c34043", "#c0a36e", "#76946a", "#6a9589", "#7e9cd8", "#957fb8"),
	"rose-pine":      schemeStylePalette("#191724", "#e0def4", "#eb6f92", "#f6c177", "#31748f", "#9ccfd8", "#31748f", "#c4a7e7"),
	"everforest":     schemeStylePalette("#2d353b", "#d3c6aa", "#e67e80", "#dbbc7f", "#a7c080", "#83c092", "#7fbbb3", "#d699b6"),
	"onedark":        schemeStylePalette("#282c34", "#abb2bf", "#e06c75", "#e5c07b", "#98c379", "#56b6c2", "#61afef", "#c678dd"),
	"solarized-dark": compactStylePalette("#002b36", "#839496", "#268bd2", "#859900", "#dc322f"),
	"ayu":            compactStylePalette("#0a0e14", "#b3b1ad", "#39bae6", "#ffb454", "#f07178"),
	"material":       compactStylePalette("#263238", "#eeffff", "#82aaff", "#c3e88d", "#ff5370"),
	"nightfox":       compactStylePalette("#192330", "#cdcecf", "#719cd6", "#81b29a", "#c94f6d"),
	"sonokai":        compactStylePalette("#2c2e34", "#e2e2e3", "#76cce0", "#9ed072", "#fc5d7c"),
	"moonfly":        compactStylePalette("#080808", "#b2b2b2", "#80a0ff", "#8cc85f", "#ff5189"),
	"oceanic-next":   compactStylePalette("#1b2b34", "#c0c5ce", "#6699cc", "#99c794", "#ec5f67"),
	"palenight":      compactStylePalette("#292d3e", "#a6accd", "#82aaff", "#c3e88d", "#ff5370"),
	"monokai":        compactStylePalette("#272822", "#f8f8f2", "#66d9ef", "#a6e22e", "#f92672"),
	"papercolor":     compactStylePalette("#eeeeee", "#444444", "#0087af", "#008700", "#af0000"),
	"edge":           compactStylePalette("#2c2e34", "#c5cdd9", "#7fbbb3", "#a7c080", "#e67e80"),
}

var gruvboxMaterialColors = [16]string{
	"#665c54",
	"#ea6962",
	"#a9b665",
	"#e78a4e",
	"#7daea3",
	"#d3869b",
	"#89b482",
	"#d4be98",
	"#928374",
	"#ea6962",
	"#a9b665",
	"#d8a657",
	"#7daea3",
	"#d3869b",
	"#89b482",
	"#d4be98",
}

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
	searchHitStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("220")).Bold(true)

	statusBadgeFGColor = lipgloss.Color("16")
	statusGreenColor   = lipgloss.Color("42")
	statusYellowColor  = lipgloss.Color("220")
	statusRedColor     = lipgloss.Color("196")
	statusBlueColor    = lipgloss.Color("69")
	statusCyanColor    = lipgloss.Color("39")
	statusMutedColor   = lipgloss.Color("245")

	paneBorderColor        = lipgloss.Color("238")
	paneBorderLoadingColor = lipgloss.Color("220")
	paneBorderActiveColor  = lipgloss.Color("45")
)

func applyTheme(name string) (string, error) {
	normalized := canonicalThemeName(name)
	switch normalized {
	case "", "default":
		applyStylePalette(defaultStylePalette())
	case "gruvbox-material":
		applyStylePalette(terminalStylePalette(gruvboxMaterialColors))
	default:
		palette, ok := themePalettes[normalized]
		if !ok {
			return "", fmt.Errorf("unknown GLAB_TUI_THEME %q (available: %s)", name, strings.Join(themeNames(), ", "))
		}
		applyStylePalette(palette)
	}
	if normalized == "" {
		normalized = "default"
	}
	return normalized, nil
}

func normalizeThemeName(name string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), " ", "-")

}

func canonicalThemeName(name string) string {
	normalized := normalizeThemeName(name)
	switch normalized {
	case "gruvbox-material-dark":
		return "gruvbox-material"
	case "tokyonight":
		return "tokyo-night"
	case "catppuccin-mocha", "mocha":
		return "catppuccin"
	case "one-dark":
		return "onedark"
	case "rosé-pine", "rosepine":
		return "rose-pine"
	case "solarized":
		return "solarized-dark"
	}
	return normalized
}

func themeIndex(name string) int {
	normalized := canonicalThemeName(name)
	for i, theme := range themeOptions {
		if theme.Name == normalized {
			return i
		}
	}
	return 0
}

func themeNames() []string {
	names := make([]string, 0, len(themeOptions))
	for _, theme := range themeOptions {
		names = append(names, theme.Name)
	}
	return names
}

func defaultStylePalette() stylePalette {
	return stylePalette{
		dim:               lipgloss.Color("240"),
		muted:             lipgloss.Color("245"),
		cyan:              lipgloss.Color("39"),
		green:             lipgloss.Color("42"),
		yellow:            lipgloss.Color("220"),
		red:               lipgloss.Color("196"),
		blue:              lipgloss.Color("69"),
		selectedFG:        lipgloss.Color("15"),
		selectedBG:        lipgloss.Color("24"),
		crumbFG:           lipgloss.Color("250"),
		crumbBG:           lipgloss.Color("236"),
		activeCrumbFG:     lipgloss.Color("16"),
		activeCrumbBG:     lipgloss.Color("45"),
		crumbSepFG:        lipgloss.Color("238"),
		crumbSepBG:        lipgloss.Color("234"),
		hintFG:            lipgloss.Color("250"),
		hintBG:            lipgloss.Color("236"),
		keyFG:             lipgloss.Color("16"),
		keyBG:             lipgloss.Color("220"),
		metaFG:            lipgloss.Color("252"),
		metaBG:            lipgloss.Color("235"),
		searchHitFG:       lipgloss.Color("16"),
		searchHitBG:       lipgloss.Color("220"),
		statusBadgeFG:     lipgloss.Color("16"),
		paneBorder:        lipgloss.Color("238"),
		paneBorderLoading: lipgloss.Color("220"),
		paneBorderActive:  lipgloss.Color("45"),
	}
}

func terminalStylePalette(colors [16]string) stylePalette {
	c := func(i int) lipgloss.Color { return lipgloss.Color(colors[i]) }
	return stylePalette{
		dim:               c(8),
		muted:             c(7),
		cyan:              c(6),
		green:             c(2),
		yellow:            c(3),
		red:               c(1),
		blue:              c(4),
		selectedFG:        c(0),
		selectedBG:        c(4),
		crumbFG:           c(7),
		crumbBG:           c(0),
		activeCrumbFG:     c(0),
		activeCrumbBG:     c(6),
		crumbSepFG:        c(8),
		crumbSepBG:        c(0),
		hintFG:            c(7),
		hintBG:            c(0),
		keyFG:             c(0),
		keyBG:             c(3),
		metaFG:            c(7),
		metaBG:            c(0),
		searchHitFG:       c(0),
		searchHitBG:       c(3),
		statusBadgeFG:     c(0),
		paneBorder:        c(8),
		paneBorderLoading: c(3),
		paneBorderActive:  c(6),
	}
}

func compactStylePalette(bg, fg, blue, green, red string) stylePalette {
	return schemeStylePalette(bg, fg, red, blue, green, blue, blue, blue)
}

func schemeStylePalette(bg, fg, red, yellow, green, cyan, blue, magenta string) stylePalette {
	background := lipgloss.Color(bg)
	foreground := lipgloss.Color(fg)
	warning := lipgloss.Color(yellow)
	success := lipgloss.Color(green)
	danger := lipgloss.Color(red)
	info := lipgloss.Color(cyan)
	accent := lipgloss.Color(blue)
	secondary := lipgloss.Color(magenta)
	return stylePalette{
		dim:               foreground,
		muted:             foreground,
		cyan:              info,
		green:             success,
		yellow:            warning,
		red:               danger,
		blue:              accent,
		selectedFG:        background,
		selectedBG:        accent,
		crumbFG:           foreground,
		crumbBG:           background,
		activeCrumbFG:     background,
		activeCrumbBG:     info,
		crumbSepFG:        foreground,
		crumbSepBG:        background,
		hintFG:            foreground,
		hintBG:            background,
		keyFG:             background,
		keyBG:             warning,
		metaFG:            foreground,
		metaBG:            background,
		searchHitFG:       background,
		searchHitBG:       secondary,
		statusBadgeFG:     background,
		paneBorder:        foreground,
		paneBorderLoading: warning,
		paneBorderActive:  info,
	}
}

func applyStylePalette(p stylePalette) {
	dimStyle = lipgloss.NewStyle().Foreground(p.dim)
	mutedStyle = lipgloss.NewStyle().Foreground(p.muted)
	cyanStyle = lipgloss.NewStyle().Foreground(p.cyan)
	greenStyle = lipgloss.NewStyle().Foreground(p.green)
	yellowStyle = lipgloss.NewStyle().Foreground(p.yellow)
	redStyle = lipgloss.NewStyle().Foreground(p.red)
	blueStyle = lipgloss.NewStyle().Foreground(p.blue)
	selectedStyle = lipgloss.NewStyle().Foreground(p.selectedFG).Background(p.selectedBG).Bold(true)
	crumbStyle = lipgloss.NewStyle().Foreground(p.crumbFG).Background(p.crumbBG).Padding(0, 1).Bold(true)
	activeCrumbStyle = lipgloss.NewStyle().Foreground(p.activeCrumbFG).Background(p.activeCrumbBG).Padding(0, 1).Bold(true)
	crumbSepStyle = lipgloss.NewStyle().Foreground(p.crumbSepFG).Background(p.crumbSepBG).Padding(0, 1)
	hintStyle = lipgloss.NewStyle().Foreground(p.hintFG).Background(p.hintBG).Padding(0, 1)
	keyStyle = lipgloss.NewStyle().Foreground(p.keyFG).Background(p.keyBG).Padding(0, 1).Bold(true)
	metaStyle = lipgloss.NewStyle().Foreground(p.metaFG).Background(p.metaBG).Padding(0, 1)
	searchHitStyle = lipgloss.NewStyle().Foreground(p.searchHitFG).Background(p.searchHitBG).Bold(true)
	statusBadgeFGColor = p.statusBadgeFG
	statusGreenColor = p.green
	statusYellowColor = p.yellow
	statusRedColor = p.red
	statusBlueColor = p.blue
	statusCyanColor = p.cyan
	statusMutedColor = p.muted
	paneBorderColor = p.paneBorder
	paneBorderLoadingColor = p.paneBorderLoading
	paneBorderActiveColor = p.paneBorderActive
}

func statusStyle(status string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(statusColor(status))
}

func selectedStatusStyle(status string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(statusBadgeFGColor).Background(statusColor(status)).Bold(true)
}

func statusColor(status string) lipgloss.Color {
	switch status {
	case "success", "ran":
		return statusGreenColor
	case "failed":
		return statusRedColor
	case "canceled", "cancelled":
		return statusRedColor
	case "running", "pending", "waiting_for_resource":
		return statusYellowColor
	case "manual", "scheduled":
		return statusBlueColor
	case "neutral", "skipped":
		return statusBlueColor
	case "preparing":
		return statusCyanColor
	default:
		return statusMutedColor
	}
}

func stripStatus(status string) string { return "[" + status + "]" }

func colorStatusInLine(line, status string) string {
	needle := stripStatus(status)
	return strings.Replace(line, needle, statusStyle(status).Render(needle), 1)
}

func colorStatusInSelectedLine(line, status string) string {
	needle := stripStatus(status)
	return strings.Replace(line, needle, selectedStatusStyle(status).Render(needle), 1)
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
	return strings.Join(items, " ")
}

func metaPill(label, value string) string {
	return metaStyle.Render(label + " " + value)
}

func branchBadge(ref string, width int) string {
	contentWidth := max(1, width-4)
	return cyanStyle.
		Bold(true).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		Render(truncate(ref, contentWidth))
}

func jobRowCard(content string, width int, selected bool) string {
	borderColor := paneBorderColor
	style := lipgloss.NewStyle().
		Width(max(1, width-4)).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder())
	if selected {
		borderColor = paneBorderActiveColor
	}
	return style.BorderForeground(borderColor).Render(content)
}
