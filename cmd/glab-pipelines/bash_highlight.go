package main

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
)

type bashTokenKind int

const (
	bashPlain bashTokenKind = iota
	bashComment
	bashString
	bashVariable
	bashKeyword
	bashOperator
	bashCommand
	bashFlag
)

type bashToken struct {
	start int
	end   int
	kind  bashTokenKind
}

var bashKeywords = map[string]bool{
	"case": true, "coproc": true, "do": true, "done": true, "elif": true,
	"else": true, "esac": true, "fi": true, "for": true, "function": true,
	"if": true, "in": true, "select": true, "then": true, "time": true,
	"until": true, "while": true,
}

func renderBashContentFor(code string, matches []logSearchMatch, index int) string {
	lines := strings.Split(code, "\n")
	var activeMatch *logSearchMatch
	if index >= 0 && index < len(matches) {
		activeMatch = &matches[index]
	}
	for lineIndex, line := range lines {
		var lineMatch *logSearchMatch
		if activeMatch != nil && activeMatch.Line == lineIndex {
			lineMatch = activeMatch
		}
		lines[lineIndex] = renderBashLine(line, lineMatch)
	}
	return strings.Join(lines, "\n")
}

func renderBashLine(line string, match *logSearchMatch) string {
	tokens := tokenizeBashLine(line)
	var b strings.Builder
	for _, token := range tokens {
		style, styled := bashTokenStyle(token.kind)
		writeHighlightedToken(&b, line, token.start, token.end, style, styled, match)
	}
	return b.String()
}

func tokenizeBashLine(line string) []bashToken {
	if line == "" {
		return nil
	}
	tokens := make([]bashToken, 0, 8)
	expectCommand := true
	for i := 0; i < len(line); {
		start := i
		switch {
		case isShellSpace(line[i]):
			for i < len(line) && isShellSpace(line[i]) {
				i++
			}
			tokens = append(tokens, bashToken{start: start, end: i, kind: bashPlain})
		case line[i] == '#' && shellTokenStart(line, i):
			tokens = append(tokens, bashToken{start: i, end: len(line), kind: bashComment})
			i = len(line)
		case line[i] == '\'' || line[i] == '"':
			i = shellQuoteEnd(line, i)
			tokens = append(tokens, bashToken{start: start, end: i, kind: bashString})
			expectCommand = false
		case line[i] == '$':
			i = shellVariableEnd(line, i)
			tokens = append(tokens, bashToken{start: start, end: i, kind: bashVariable})
			expectCommand = false
		case isShellOperator(line[i]):
			for i < len(line) && isShellOperator(line[i]) {
				i++
			}
			tokens = append(tokens, bashToken{start: start, end: i, kind: bashOperator})
			operator := line[start:i]
			if operator == ";" || operator == "|" || operator == "||" || operator == "&" || operator == "&&" || strings.Contains(operator, "(") {
				expectCommand = true
			}
		default:
			for i < len(line) && !isShellSpace(line[i]) && !isShellOperator(line[i]) && line[i] != '\'' && line[i] != '"' && line[i] != '$' && !(line[i] == '#' && shellTokenStart(line, i)) {
				i++
			}
			word := line[start:i]
			kind := bashPlain
			switch {
			case bashKeywords[word]:
				kind = bashKeyword
				expectCommand = word == "if" || word == "then" || word == "do" || word == "else" || word == "elif" || word == "while" || word == "until"
			case expectCommand && isShellAssignment(word):
				kind = bashVariable
			case expectCommand:
				kind = bashCommand
				expectCommand = false
			case strings.HasPrefix(word, "-"):
				kind = bashFlag
			}
			tokens = append(tokens, bashToken{start: start, end: i, kind: kind})
		}
	}
	return tokens
}

func bashTokenStyle(kind bashTokenKind) (lipgloss.Style, bool) {
	switch kind {
	case bashComment:
		return dimStyle, true
	case bashString:
		return yellowStyle, true
	case bashVariable:
		return cyanStyle, true
	case bashKeyword:
		return blueStyle.Bold(true), true
	case bashOperator:
		return yellowStyle.Bold(true), true
	case bashCommand:
		return greenStyle.Bold(true), true
	case bashFlag:
		return blueStyle, true
	default:
		return lipgloss.NewStyle(), false
	}
}

func writeHighlightedToken(b *strings.Builder, line string, start, end int, style lipgloss.Style, styled bool, match *logSearchMatch) {
	if match == nil || match.End <= start || match.Start >= end {
		writeBashStyled(b, line[start:end], style, styled)
		return
	}
	overlapStart := max(start, match.Start)
	overlapEnd := min(end, match.End)
	writeBashStyled(b, line[start:overlapStart], style, styled)
	b.WriteString(searchHitStyle.Render(line[overlapStart:overlapEnd]))
	writeBashStyled(b, line[overlapEnd:end], style, styled)
}

func writeBashStyled(b *strings.Builder, value string, style lipgloss.Style, styled bool) {
	if value == "" {
		return
	}
	if styled {
		b.WriteString(style.Render(value))
		return
	}
	b.WriteString(value)
}

func shellQuoteEnd(line string, start int) int {
	quote := line[start]
	for i := start + 1; i < len(line); i++ {
		if quote == '"' && line[i] == '\\' {
			i++
			continue
		}
		if line[i] == quote {
			return i + 1
		}
	}
	return len(line)
}

func shellVariableEnd(line string, start int) int {
	if start+1 >= len(line) {
		return start + 1
	}
	if line[start+1] == '{' {
		if end := strings.IndexByte(line[start+2:], '}'); end >= 0 {
			return start + 2 + end + 1
		}
		return len(line)
	}
	if line[start+1] == '(' {
		depth := 1
		for i := start + 2; i < len(line); i++ {
			switch line[i] {
			case '(':
				depth++
			case ')':
				depth--
				if depth == 0 {
					return i + 1
				}
			}
		}
		return len(line)
	}
	i := start + 1
	for i < len(line) {
		r := rune(line[i])
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || line[i] == '_' || strings.ContainsRune("?@#*-$!", r)) {
			break
		}
		i++
	}
	return i
}

func shellTokenStart(line string, index int) bool {
	return index == 0 || isShellSpace(line[index-1]) || isShellOperator(line[index-1])
}

func isShellSpace(value byte) bool {
	return value == ' ' || value == '\t'
}

func isShellOperator(value byte) bool {
	return strings.ContainsRune(";|&()<>", rune(value))
}

func isShellAssignment(word string) bool {
	name, _, ok := strings.Cut(word, "=")
	if !ok || name == "" {
		return false
	}
	for i, r := range name {
		if !(unicode.IsLetter(r) || r == '_' || (i > 0 && unicode.IsDigit(r))) {
			return false
		}
	}
	return true
}
