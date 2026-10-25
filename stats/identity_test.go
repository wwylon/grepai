package stats

import (
	"strings"
	"testing"
)

func TestTruncateQuery(t *testing.T) {
	if TruncateQuery("  hi  ") != "hi" {
		t.Fatal("trim")
	}
	long := strings.Repeat("q", MaxQueryRunes+10)
	got := TruncateQuery(long)
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("suffix %q", got)
	}
}

func TestFillIdentity(t *testing.T) {
	t.Setenv("GREPAI_STATS_ACTOR", "agent-htn")
	t.Setenv("USERNAME", "osuser")
	e := Entry{Query: " opening lead "}
	FillIdentity(&e, SourceMCP)
	if e.Source != SourceMCP || e.Actor != "agent-htn" || e.Query != "opening lead" {
		t.Fatalf("%+v", e)
	}
	e2 := Entry{}
	t.Setenv("GREPAI_STATS_ACTOR", "")
	FillIdentity(&e2, SourceCLI)
	if e2.Actor == "" {
		t.Fatal("expected OS user fallback")
	}
}
