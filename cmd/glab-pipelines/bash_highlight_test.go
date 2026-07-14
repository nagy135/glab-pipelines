package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestTokenizeBashLineClassifiesShellSyntax(t *testing.T) {
	line := `if DEBUG=1 test "$HOME" = yes; then echo --verbose '# literal'; fi # comment`
	tokens := tokenizeBashLine(line)
	want := map[string]bashTokenKind{
		"if":          bashKeyword,
		"DEBUG=1":     bashVariable,
		"test":        bashCommand,
		`"$HOME"`:     bashString,
		"then":        bashKeyword,
		"echo":        bashCommand,
		"--verbose":   bashFlag,
		`'# literal'`: bashString,
		"fi":          bashKeyword,
		"# comment":   bashComment,
	}
	for text, kind := range want {
		found := false
		for _, token := range tokens {
			if line[token.start:token.end] == text && token.kind == kind {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("token %q was not classified as %d: %#v", text, kind, tokens)
		}
	}
}

func TestRenderBashContentPreservesSource(t *testing.T) {
	code := "echo \"hello\" | grep -q hello\nprintf '%s\\n' \"$HOME\" # output"
	rendered := renderBashContentFor(code, nil, -1)
	if got := ansi.Strip(rendered); got != code {
		t.Fatalf("highlighted source = %q, want %q", got, code)
	}
}

func TestRenderBashContentPreservesSearchHighlight(t *testing.T) {
	code := "echo hello"
	matches := findLogSearchMatches(code, "echo")
	rendered := renderBashContentFor(code, matches, 0)
	if !strings.Contains(rendered, searchHitStyle.Render("echo")) {
		t.Fatalf("search match was not highlighted: %q", rendered)
	}
	if got := ansi.Strip(rendered); got != code {
		t.Fatalf("highlighted source = %q, want %q", got, code)
	}
}
