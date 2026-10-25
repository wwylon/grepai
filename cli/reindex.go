package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/yoanbernabeu/grepai/config"
	"github.com/yoanbernabeu/grepai/embedder"
	"github.com/yoanbernabeu/grepai/indexer"
)

var reindexPaths []string

var reindexCmd = &cobra.Command{
	Use:   "reindex",
	Short: "Re-index specific paths in the current project",
	Long: `Re-index one or more files or directories without restarting watch.

Useful after an agent edit batch:

  grepai reindex --paths src/a.go --paths pkg/

Uses the same IndexFile core as the watcher (ContentHash reuse when possible).`,
	RunE: runReindex,
}

func init() {
	reindexCmd.Flags().StringArrayVar(&reindexPaths, "paths", nil, "File or directory paths to re-index (repeatable)")
	rootCmd.AddCommand(reindexCmd)
}

func runReindex(cmd *cobra.Command, args []string) error {
	if len(reindexPaths) == 0 {
		return fmt.Errorf("at least one --paths value is required")
	}

	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx := context.Background()
	emb, err := initializeEmbedder(ctx, cfg)
	if err != nil {
		return err
	}
	defer emb.Close()
	if err := embedder.Prewarm(ctx, emb); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: embedder prewarm: %v\n", err)
	}

	st, err := initializeStore(ctx, cfg, projectRoot)
	if err != nil {
		return err
	}
	defer st.Close()

	ignoreMatcher, err := indexer.NewIgnoreMatcher(projectRoot, cfg.Ignore, cfg.ExternalGitignore)
	if err != nil {
		return err
	}
	scanner := indexer.NewScanner(projectRoot, ignoreMatcher)
	chunker := indexer.NewFileChunker(cfg.Chunking.Strategy, cfg.Chunking.Size, cfg.Chunking.Overlap)
	idx := indexer.NewIndexer(projectRoot, st, emb, chunker, scanner, cfg.Watch.LastIndexTime)

	start := time.Now()
	var totalChunks, files int

	for _, p := range reindexPaths {
		abs := p
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(projectRoot, p)
		}
		info, err := os.Stat(abs)
		if err != nil {
			return fmt.Errorf("stat %s: %w", p, err)
		}
		if info.IsDir() {
			err := filepath.Walk(abs, func(path string, fi os.FileInfo, walkErr error) error {
				if walkErr != nil || fi.IsDir() {
					return walkErr
				}
				rel, err := filepath.Rel(projectRoot, path)
				if err != nil {
					return err
				}
				rel = filepath.ToSlash(rel)
				if ignoreMatcher.ShouldIgnore(rel) {
					return nil
				}
				n, err := reindexOne(ctx, idx, scanner, rel)
				if err != nil {
					return err
				}
				if n > 0 {
					totalChunks += n
					files++
				}
				return nil
			})
			if err != nil {
				return err
			}
			continue
		}
		rel, err := filepath.Rel(projectRoot, abs)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		n, err := reindexOne(ctx, idx, scanner, rel)
		if err != nil {
			return err
		}
		if n > 0 {
			totalChunks += n
			files++
		}
	}

	if err := st.Persist(ctx); err != nil {
		return fmt.Errorf("persist: %w", err)
	}
	fmt.Printf("Reindexed %d files (%d chunks) in %s\n", files, totalChunks, time.Since(start).Round(time.Millisecond))
	return nil
}

func reindexOne(ctx context.Context, idx *indexer.Indexer, scanner *indexer.Scanner, rel string) (int, error) {
	fi, err := scanner.ScanFile(rel)
	if err != nil {
		return 0, err
	}
	if fi == nil {
		return 0, nil
	}
	return idx.IndexFile(ctx, *fi)
}
