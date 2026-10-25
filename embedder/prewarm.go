package embedder

import (
	"context"
	"fmt"
)

// Pinger is implemented by embedders that can check backend reachability.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Prewarm pings the embedder (when supported) and runs a tiny Embed so the
// model/server is hot before IndexAll or watch starts.
func Prewarm(ctx context.Context, emb Embedder) error {
	if p, ok := emb.(Pinger); ok {
		if err := p.Ping(ctx); err != nil {
			return fmt.Errorf("ping: %w", err)
		}
	}
	if _, err := emb.Embed(ctx, "grepai prewarm"); err != nil {
		return fmt.Errorf("embed: %w", err)
	}
	return nil
}
