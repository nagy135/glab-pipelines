package main

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) beginLogSearch() model {
	m.logSearchMode = true
	m.logSearchActive = false
	m.logSearchQuery = ""
	m.logSearchMatches = nil
	m.logSearchIndex = -1
	m.message = ""
	return m.configureLogViewport()
}

func (m model) clearLogSearch() model {
	m.logSearchMode = false
	m.logSearchActive = false
	m.logSearchQuery = ""
	m.logSearchMatches = nil
	m.logSearchIndex = -1
	m.message = ""
	if m.logs != "" {
		m.logsViewport.SetContent(m.logs)
	}
	return m.configureLogViewport()
}

func (m model) handleLogSearchKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyEsc:
		return m.clearLogSearch().saveActiveLogPane(), nil
	case tea.KeyEnter:
		m.logSearchMode = false
		m.logSearchActive = m.logSearchQuery != "" && len(m.logSearchMatches) > 0
		m.logsViewport.SetContent(m.renderLogContent())
		return m.configureLogViewport().saveActiveLogPane(), nil
	case tea.KeyBackspace, tea.KeyCtrlH:
		m.logSearchQuery = trimLastRune(m.logSearchQuery)
		return m.jumpToCurrentLogSearch().saveActiveLogPane(), nil
	case tea.KeySpace:
		m.logSearchQuery += " "
		return m.jumpToCurrentLogSearch().saveActiveLogPane(), nil
	case tea.KeyRunes:
		m.logSearchQuery += key.String()
		return m.jumpToCurrentLogSearch().saveActiveLogPane(), nil
	}
	return m, nil
}

func (m model) jumpToCurrentLogSearch() model {
	m = m.refreshLogSearchMatches()
	m.logsViewport.SetContent(m.renderLogContent())
	if m.logSearchQuery == "" || len(m.logSearchMatches) == 0 {
		return m
	}
	m.logSearchIndex = logSearchMatchNear(m.logSearchMatches, m.logsViewport.YOffset, 1)
	m.logsViewport.SetContent(m.renderLogContent())
	m.logsViewport.SetYOffset(m.logSearchMatches[m.logSearchIndex].Line)
	return m
}

func (m model) jumpLogSearchMatch(direction int) model {
	m = m.refreshLogSearchMatches()
	if m.logSearchQuery == "" {
		m.message = "no active search"
		return m
	}
	if len(m.logSearchMatches) == 0 {
		m.message = fmt.Sprintf("no matches for %q", m.logSearchQuery)
		return m
	}
	if m.logSearchIndex < 0 || m.logSearchIndex >= len(m.logSearchMatches) {
		m.logSearchIndex = logSearchMatchNear(m.logSearchMatches, m.logsViewport.YOffset, direction)
	} else {
		m.logSearchIndex = (m.logSearchIndex + direction + len(m.logSearchMatches)) % len(m.logSearchMatches)
	}
	m.logsViewport.SetContent(m.renderLogContent())
	m.logsViewport.SetYOffset(m.logSearchMatches[m.logSearchIndex].Line)
	m.message = ""
	return m
}

func (m model) refreshLogSearchMatches() model {
	m.logSearchMatches = findLogSearchMatches(m.logs, m.logSearchQuery)
	if len(m.logSearchMatches) == 0 {
		m.logSearchIndex = -1
		return m
	}
	if m.logSearchIndex >= len(m.logSearchMatches) {
		m.logSearchIndex = len(m.logSearchMatches) - 1
	}
	return m
}

func findLogSearchMatches(logs, query string) []logSearchMatch {
	query = strings.Map(unicode.ToLower, query)
	if query == "" {
		return nil
	}
	lines := strings.Split(logs, "\n")
	matches := make([]logSearchMatch, 0)
	for i, line := range lines {
		lineLower, offsets := foldCaseWithOffsets(line)
		for offset := 0; offset <= len(lineLower); {
			idx := strings.Index(lineLower[offset:], query)
			if idx < 0 {
				break
			}
			foldedStart := offset + idx
			foldedEnd := foldedStart + len(query)
			start, startOK := offsets[foldedStart]
			end, endOK := offsets[foldedEnd]
			if startOK && endOK {
				matches = append(matches, logSearchMatch{Line: i, Start: start, End: end})
			}
			offset = foldedEnd
		}
	}
	return matches
}

func foldCaseWithOffsets(value string) (string, map[int]int) {
	var folded strings.Builder
	offsets := map[int]int{0: 0}
	for sourceOffset := 0; sourceOffset < len(value); {
		r, size := utf8.DecodeRuneInString(value[sourceOffset:])
		foldedOffset := folded.Len()
		offsets[foldedOffset] = sourceOffset
		folded.WriteRune(unicode.ToLower(r))
		sourceOffset += size
		offsets[folded.Len()] = sourceOffset
	}
	return folded.String(), offsets
}

func logSearchMatchNear(matches []logSearchMatch, line int, direction int) int {
	if direction < 0 {
		for i := len(matches) - 1; i >= 0; i-- {
			if matches[i].Line <= line {
				return i
			}
		}
		return len(matches) - 1
	}
	for i, match := range matches {
		if match.Line >= line {
			return i
		}
	}
	return 0
}

func trimLastRune(value string) string {
	if value == "" {
		return ""
	}
	runes := []rune(value)
	return string(runes[:len(runes)-1])
}

func (m model) logSearchStatus() string {
	if !m.logSearchMode && m.logSearchQuery == "" {
		return ""
	}
	if m.logSearchQuery == "" {
		return cyanStyle.Render("/")
	}
	if len(m.logSearchMatches) == 0 {
		return cyanStyle.Render("/"+m.logSearchQuery) + " " + redStyle.Render("no matches")
	}
	current := max(1, m.logSearchIndex+1)
	return fmt.Sprintf("%s %s", cyanStyle.Render("/"+m.logSearchQuery), dimStyle.Render(fmt.Sprintf("%d/%d", current, len(m.logSearchMatches))))
}

func (m model) renderLogContent() string {
	return renderLogContentFor(m.logs, m.logSearchMatches, m.logSearchIndex)
}

func renderLogContentFor(logs string, matches []logSearchMatch, index int) string {
	if index < 0 || index >= len(matches) {
		return logs
	}
	match := matches[index]
	lines := strings.Split(logs, "\n")
	if match.Line < 0 || match.Line >= len(lines) {
		return logs
	}
	line := lines[match.Line]
	if match.Start < 0 || match.End > len(line) || match.Start >= match.End {
		return logs
	}
	lines[match.Line] = line[:match.Start] + searchHitStyle.Render(line[match.Start:match.End]) + line[match.End:]
	return strings.Join(lines, "\n")
}
