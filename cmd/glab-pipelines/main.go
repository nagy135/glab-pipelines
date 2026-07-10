package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	m, err := initialModel(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initialModel(args []string) (model, error) {
	themeValue := os.Getenv("GLAB_TUI_THEME")
	if themeValue == "" {
		if savedTheme, ok := loadSavedThemeName(); ok {
			if _, err := applyTheme(savedTheme); err == nil {
				themeValue = savedTheme
			}
		}
	}
	themeName, err := applyTheme(themeValue)
	if err != nil {
		return model{}, err
	}
	borderName := applyActiveBorder("")
	if savedBorder, ok := loadSavedBorderName(); ok {
		borderName = applyActiveBorder(savedBorder)
	}

	m := model{
		status:      "active",
		limit:       envInt("GLAB_TUI_LIMIT", 10),
		refresh:     envDuration("GLAB_TUI_REFRESH", 20*time.Second),
		logRefresh:  envDuration("GLAB_TUI_LOG_REFRESH", 3*time.Second),
		mode:        modePipelines,
		activity:    newActivitySpinner(),
		loadingList: true,
		listRequest: 1,
		themeName:   themeName,
		themeCursor: themeIndex(themeName),
		borderName:  borderName,
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-R", "--repo":
			if i+1 >= len(args) {
				return m, errors.New("missing repo after -R/--repo")
			}
			m.repo = args[i+1]
			i++
		case "-h", "--help":
			printHelp()
			os.Exit(0)
		default:
			m.status = args[i]
		}
	}
	if cached, ok := loadPipelineCache(m.repo, m.status, m.limit); ok {
		m.list = cached
	}
	m.jobDurations = loadJobDurations(m.repo)
	return m, nil
}

func printHelp() {
	fmt.Println(`glab-pipelines - interactive TUI for GitLab pipelines via glab

Usage:
  glab-pipelines                 active pipelines, default
  glab-pipelines running         only running pipelines
  glab-pipelines manual          only manual pipelines
  glab-pipelines pending         any single status
  glab-pipelines all             newest pipelines regardless of status
  glab-pipelines -R group/proj active

Environment:
	GLAB_TUI_LIMIT=10              newest pipelines to show
	GLAB_TUI_REFRESH=20            detail refresh interval in seconds
	GLAB_TUI_LOG_REFRESH=3         log refresh interval in seconds
	GLAB_TUI_THEME=gruvbox-material color theme override; preferences save to ~/.local/share/glab-pipelines

Keys:
  Pipeline list: j/k or up/down move, enter details, s/v split, ctrl+hjkl focus, x close split, o only focused split, t theme, b border, r refresh, q close/quit, Q quit app
  Detail: j/k or up/down move jobs, s/v split, ctrl+hjkl focus, x close split, o only focused split, t theme, b border, l logs in focused split, L inline logs in focused split, S start/retry, c cancel, r refresh, q close, esc back
  Jobs: j/k or up/down move, s start/retry, c cancel, l logs, t theme, b border, r refresh, q back
  Logs: j/k or arrows scroll, pgup/pgdn page, g top, G bottom, s/v split, ctrl+hjkl focus, x close split, o only focused split, / search, t theme, b border, n next job or next match, N previous match, r reload, q close, esc back`)
}

func envInt(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func envDuration(name string, fallback time.Duration) time.Duration {
	seconds := envInt(name, int(fallback/time.Second))
	const maxSeconds = int64(^uint64(0)>>1) / int64(time.Second)
	if int64(seconds) > maxSeconds {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}
