package cli

import (
	"context"
	"testing"
	"time"

	"github.com/yoanbernabeu/grepai/store"
)

type stubStatsStore struct {
	store.VectorStore
	files []store.FileStats
}

func (s stubStatsStore) ListFilesWithStats(ctx context.Context) ([]store.FileStats, error) {
	return s.files, nil
}
func (s stubStatsStore) GetStats(ctx context.Context) (*store.IndexStats, error) {
	return &store.IndexStats{IndexSize: 42}, nil
}

func TestCollectProjectStats(t *testing.T) {
	st := stubStatsStore{files: []store.FileStats{
		{Path: "ws/frontend/a.go", ChunkCount: 2, ModTime: time.Unix(10, 0)},
		{Path: "ws/api/b.go", ChunkCount: 5, ModTime: time.Unix(20, 0)},
	}}
	docs, chunks, last, size, err := collectProjectStats(context.Background(), st, "ws", "frontend")
	if err != nil {
		t.Fatal(err)
	}
	if docs != 1 || chunks != 2 || size != 42 || last.Unix() != 10 {
		t.Fatalf("docs=%d chunks=%d size=%d last=%v", docs, chunks, size, last)
	}
}
