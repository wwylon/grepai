package search

import (
	"testing"

	"github.com/yoanbernabeu/grepai/store"
)

func TestPreferOverlayResults(t *testing.T) {
	results := []store.SearchResult{
		{Chunk: store.Chunk{FilePath: "ws/frontend/src/app.ts"}, Score: 0.9},
		{Chunk: store.Chunk{FilePath: "ws/feature/src/app.ts"}, Score: 0.8},
		{Chunk: store.Chunk{FilePath: "ws/api/src/ok.ts"}, Score: 0.7},
	}
	got := PreferOverlayResults(results, "ws", "feature", "frontend")
	if len(got) != 2 {
		t.Fatalf("len=%d %+v", len(got), got)
	}
	if got[0].Chunk.FilePath != "ws/feature/src/app.ts" {
		t.Fatalf("want overlay first remaining, got %s", got[0].Chunk.FilePath)
	}
}
