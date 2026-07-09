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
	m := model{
		status:      "active",
		limit:       envInt("GLAB_TUI_LIMIT", 10),
		refresh:     time.Duration(envInt("GLAB_TUI_REFRESH", 20)) * time.Second,
		logRefresh:  time.Duration(envInt("GLAB_TUI_LOG_REFRESH", 3)) * time.Second,
		mode:        modePipelines,
		loadingList: true,
		listRequest: 1,
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

Keys:
  Pipeline list: j/k or up/down move, enter details, r refresh, q quit
  Detail: j/k or up/down move jobs, p jobs list, s start/retry, c cancel, l logs, r refresh, q back, o open
  Jobs: j/k or up/down move, s start/retry, c cancel, l logs, r refresh, q back
  Logs: j/k or arrows scroll, pgup/pgdn page, g top, G bottom, n next job, r reload, q back`)
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
