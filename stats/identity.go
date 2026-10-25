package stats

import (
	"os"
	"strings"
	"unicode/utf8"
)

const (
	SourceCLI = "cli"
	SourceMCP = "mcp"
)

// MaxQueryRunes caps stored query text so stats files stay small.
const MaxQueryRunes = 400

// TruncateQuery shortens a query for storage.
func TruncateQuery(q string) string {
	q = strings.TrimSpace(q)
	if utf8.RuneCountInString(q) <= MaxQueryRunes {
		return q
	}
	r := []rune(q)
	return string(r[:MaxQueryRunes]) + "…"
}

// FillIdentity sets source, actor, and cwd when missing.
// Actor: explicit value, then GREPAI_STATS_ACTOR, then OS username.
func FillIdentity(e *Entry, source string) {
	if e.Source == "" {
		e.Source = source
	}
	if e.Actor == "" {
		e.Actor = os.Getenv("GREPAI_STATS_ACTOR")
	}
	if e.Actor == "" {
		e.Actor = os.Getenv("USERNAME")
		if e.Actor == "" {
			e.Actor = os.Getenv("USER")
		}
	}
	if e.Cwd == "" {
		if cwd, err := os.Getwd(); err == nil {
			e.Cwd = cwd
		}
	}
	e.Query = TruncateQuery(e.Query)
}
