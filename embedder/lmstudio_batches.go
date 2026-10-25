package embedder

import (
	"context"
	"fmt"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

const defaultLMStudioParallelism = 1

// EmbedBatches implements BatchEmbedder for LM Studio (OpenAI-compatible /v1/embeddings).
func (e *LMStudioEmbedder) EmbedBatches(ctx context.Context, batches []Batch, progress BatchProgress) ([]BatchResult, error) {
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
		limit = defaultLMStudioParallelism
	}

	results := make([]BatchResult, len(batches))
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(limit)

	for i := range batches {
		batch := batches[i]
		g.Go(func() error {
			embeddings, err := e.EmbedBatch(ctx, batch.Contents())
			if err != nil {
				return fmt.Errorf("lmstudio batch %d: %w", batch.Index, err)
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
