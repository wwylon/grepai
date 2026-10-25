package embedder

import (
	"context"
	"fmt"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

const defaultOllamaParallelism = 1

// EmbedBatches implements BatchEmbedder for Ollama.
// Each FormBatches batch is embedded via EmbedBatch (which already chunks to
// ollamaBatchSize). Parallelism controls how many batch HTTP calls run at once;
// default is 1 for single-GPU hosts (see OLLAMA_NUM_PARALLEL).
func (e *OllamaEmbedder) EmbedBatches(ctx context.Context, batches []Batch, progress BatchProgress) ([]BatchResult, error) {
	if len(batches) == 0 {
		return nil, nil
	}

	totalChunks := 0
	for _, batch := range batches {
		totalChunks += batch.Size()
	}
	var completedChunks atomic.Int64

	limit := e.parallelism
	if limit < 1 {
		limit = defaultOllamaParallelism
	}

	results := make([]BatchResult, len(batches))
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(limit)

	for i := range batches {
		batch := batches[i]
		g.Go(func() error {
			embeddings, err := e.EmbedBatch(ctx, batch.Contents())
			if err != nil {
				return fmt.Errorf("ollama batch %d: %w", batch.Index, err)
			}
			results[batch.Index] = BatchResult{
				BatchIndex: batch.Index,
				Embeddings: embeddings,
			}
			done := int(completedChunks.Add(int64(batch.Size())))
			if progress != nil {
				progress(batch.Index, len(batches), done, totalChunks, false, 0, 0)
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}
