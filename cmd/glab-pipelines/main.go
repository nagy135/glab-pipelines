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
	themeValue := firstEnv("CI_TUI_THEME", "GLAB_TUI_THEME")
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
		status:         "active",
		limit:          envIntAny([]string{"CI_TUI_LIMIT", "GLAB_TUI_LIMIT"}, 10),
		refresh:        envDurationAny([]string{"CI_TUI_REFRESH", "GLAB_TUI_REFRESH"}, 20*time.Second),
		logRefresh:     envDurationAny([]string{"CI_TUI_LOG_REFRESH", "GLAB_TUI_LOG_REFRESH"}, 3*time.Second),
		mode:           modePipelines,
		activity:       newActivitySpinner(),
		loadingList:    true,
		listRequest:    1,
		showInlineLogs: true,
		themeName:      themeName,
		themeCursor:    themeIndex(themeName),
		borderName:     borderName,
	}
	providerValue := firstEnv("CI_TUI_PROVIDER")
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-R", "--repo":
			if i+1 >= len(args) {
				return m, errors.New("missing repo after -R/--repo")
			}
			m.repo = args[i+1]
			i++
		case "-p", "--provider":
			if i+1 >= len(args) {
				return m, errors.New("missing provider after -p/--provider")
			}
			providerValue = args[i+1]
			i++
		case "--github":
			providerValue = "github"
		case "--gitlab":
			providerValue = "gitlab"
		case "-h", "--help":
			printHelp()
			os.Exit(0)
		default:
			m.status = args[i]
		}
	}
	if providerValue == "" || strings.EqualFold(providerValue, "auto") {
		m.provider = detectProvider(m.repo)
	} else {
		m.provider, err = parseProvider(providerValue)
		if err != nil {
			return m, err
		}
	}
	scope := providerScope(m.provider, m.repo)
	if cached, ok := loadPipelineCache(scope, m.status, m.limit); ok {
		m.list = cached
	}
	m.jobDurations = loadJobDurations(scope)
	return m, nil
}

func printHelp() {
	fmt.Println(`glab-pipelines - interactive TUI for GitLab CI and GitHub Actions

Usage:
  glab-pipelines                 active pipelines/runs; provider auto-detected from origin
  glab-pipelines running         only running pipelines
  glab-pipelines manual          only manual pipelines
  glab-pipelines pending         any single status
  glab-pipelines all             newest pipelines regardless of status
  glab-pipelines -R group/proj active
  glab-pipelines --provider github -R owner/repo all
  glab-pipelines --github        force GitHub Actions via gh
  glab-pipelines --gitlab        force GitLab CI via glab

Environment:
  CI_TUI_PROVIDER=auto            auto, github, or gitlab
  CI_TUI_LIMIT=10                 newest pipelines to show
  CI_TUI_REFRESH=20               detail refresh interval in seconds
  CI_TUI_LOG_REFRESH=3            job log refresh interval in seconds
  CI_TUI_THEME=gruvbox-material   color theme override

  Legacy GLAB_TUI_* settings remain supported.
  GLAB_TUI_LIMIT=10              newest pipelines to show
  GLAB_TUI_REFRESH=20            detail refresh interval in seconds
  GLAB_TUI_LOG_REFRESH=3         log refresh interval in seconds
  GLAB_TUI_THEME=gruvbox-material color theme override; preferences save to ~/.local/share/glab-pipelines

Keys:
  Pipeline list: j/k or up/down move, ctrl+n/p/f/b scroll down/up/right/left, c cancel pipeline, s/v split, ctrl+hjkl focus, x close split, o only focused split, t theme, b border, r refresh, q close/quit, Q quit app
  Detail: j/k or up/down move jobs, ctrl+n/p/f/b scroll down/up/right/left, s/v split, ctrl+hjkl focus, x close split, o only focused split, t theme, b border, l logs in focused split, C code in focused split, L inline logs in focused split, S play/retry/rerun, c cancel, r refresh, q close, esc back
  Jobs: j/k or up/down move, s play/retry/rerun, c cancel, l logs, C code, t theme, b border, r refresh, q back
  Logs/code: j/k or ctrl+n/p scroll vertically, left/right or ctrl+f/b scroll horizontally, pgup/pgdn page, g top, G bottom, s/v split, ctrl+hjkl focus, x close split, o only focused split, / search, t theme, b border, n/N search matches, r reload, q close, esc back`)
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

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}

func envIntAny(names []string, fallback int) int {
	for _, name := range names {
		if strings.TrimSpace(os.Getenv(name)) != "" {
			return envInt(name, fallback)
		}
	}
	return fallback
}

func envDuration(name string, fallback time.Duration) time.Duration {
	seconds := envInt(name, int(fallback/time.Second))
	const maxSeconds = int64(^uint64(0)>>1) / int64(time.Second)
	if int64(seconds) > maxSeconds {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}

func envDurationAny(names []string, fallback time.Duration) time.Duration {
	for _, name := range names {
		if strings.TrimSpace(os.Getenv(name)) != "" {
			return envDuration(name, fallback)
		}
	}
	return fallback
}
