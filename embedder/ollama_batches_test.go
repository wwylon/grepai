package embedder

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestOllamaEmbedder_EmbedBatches_ParallelismLimit(t *testing.T) {
	var inflight atomic.Int32
	var maxInflight atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cur := inflight.Add(1)
		for {
			prev := maxInflight.Load()
			if cur <= prev || maxInflight.CompareAndSwap(prev, cur) {
				break
			}
		}
		defer inflight.Add(-1)

		var req ollamaBatchEmbedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode: %v", err)
			http.Error(w, err.Error(), 400)
			return
		}
		resp := ollamaBatchEmbedResponse{Embeddings: make([][]float32, len(req.Input))}
		for i := range req.Input {
			resp.Embeddings[i] = []float32{0.1, 0.2}
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	e := NewOllamaEmbedder(
		WithOllamaEndpoint(server.URL),
		WithOllamaParallelism(2),
	)
	batches := []Batch{
		{Index: 0, Entries: []BatchEntry{{Content: "a"}, {Content: "b"}}},
		{Index: 1, Entries: []BatchEntry{{Content: "c"}, {Content: "d"}}},
		{Index: 2, Entries: []BatchEntry{{Content: "e"}}},
	}
	results, err := e.EmbedBatches(context.Background(), batches, nil)
	if err != nil {
		t.Fatalf("EmbedBatches: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("results=%d", len(results))
	}
	if maxInflight.Load() > 2 {
		t.Fatalf("max inflight %d > parallelism 2", maxInflight.Load())
	}
}
