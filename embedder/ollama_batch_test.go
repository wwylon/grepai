package embedder

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestOllamaEmbedder_EmbedBatches_Parallelism(t *testing.T) {
	var current, maxConcurrent, requests int32
	gate := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := atomic.AddInt32(&current, 1)
		defer atomic.AddInt32(&current, -1)
		for {
			m := atomic.LoadInt32(&maxConcurrent)
			if c <= m || atomic.CompareAndSwapInt32(&maxConcurrent, m, c) {
				break
			}
		}
		atomic.AddInt32(&requests, 1)
		if c >= 2 {
			select {
			case <-gate:
			default:
				close(gate)
			}
		}
		select {
		case <-gate:
		case <-time.After(2 * time.Second):
		}
		_ = json.NewEncoder(w).Encode(ollamaEmbedResponse{Embedding: []float32{0.1, 0.2, 0.3}})
	}))
	defer server.Close()

	e := NewOllamaEmbedder(WithOllamaEndpoint(server.URL), WithOllamaParallelism(2), WithOllamaDimensions(3))
	batches := []Batch{
		{Index: 0, Entries: []BatchEntry{{Content: "a"}}},
		{Index: 1, Entries: []BatchEntry{{Content: "b"}}},
		{Index: 2, Entries: []BatchEntry{{Content: "c"}}},
	}
	out, err := e.EmbedBatches(context.Background(), batches, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 {
		t.Fatalf("results=%d", len(out))
	}
	if atomic.LoadInt32(&maxConcurrent) < 2 {
		t.Fatalf("max concurrent=%d, want >=2", maxConcurrent)
	}
}

func TestOllamaEmbedder_EmbedBatches_Parallelism1(t *testing.T) {
	var current, maxConcurrent int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := atomic.AddInt32(&current, 1)
		defer atomic.AddInt32(&current, -1)
		for {
			m := atomic.LoadInt32(&maxConcurrent)
			if c <= m || atomic.CompareAndSwapInt32(&maxConcurrent, m, c) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(ollamaEmbedResponse{Embedding: []float32{1, 2, 3}})
	}))
	defer server.Close()

	e := NewOllamaEmbedder(WithOllamaEndpoint(server.URL), WithOllamaParallelism(1), WithOllamaDimensions(3))
	batches := []Batch{
		{Index: 0, Entries: []BatchEntry{{Content: "a"}}},
		{Index: 1, Entries: []BatchEntry{{Content: "b"}}},
	}
	if _, err := e.EmbedBatches(context.Background(), batches, nil); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&maxConcurrent) != 1 {
		t.Fatalf("parallelism 1 should be sequential, max=%d", maxConcurrent)
	}
}
