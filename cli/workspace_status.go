package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yoanbernabeu/grepai/config"
	"github.com/yoanbernabeu/grepai/store"
)

type workspaceProjectStatusReport struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	PathExists  bool   `json:"path_exists"`
	Documents   int    `json:"documents"`
	Chunks      int    `json:"chunks"`
	IndexSize   int64  `json:"index_size"`
	LastUpdated string `json:"last_updated,omitempty"`
	Error       string `json:"error,omitempty"`
}

type workspaceStatusReport struct {
	Name     string                         `json:"name"`
	Backend  string                         `json:"backend"`
	Projects []workspaceProjectStatusReport `json:"projects"`
}

func collectProjectStats(ctx context.Context, st store.VectorStore, workspaceName, projectName string) (docs, chunks int, last time.Time, indexSize int64, err error) {
	prefix := workspaceName + "/" + projectName + "/"
	files, err := st.ListFilesWithStats(ctx)
	if err != nil {
		return 0, 0, time.Time{}, 0, err
	}
	for _, f := range files {
		if !strings.HasPrefix(f.Path, prefix) {
			continue
		}
		docs++
		chunks += f.ChunkCount
		if f.ModTime.After(last) {
			last = f.ModTime
		}
	}
	if stats, sErr := st.GetStats(ctx); sErr == nil && stats != nil {
		indexSize = stats.IndexSize
	}
	return docs, chunks, last, indexSize, nil
}

func buildWorkspaceStatusJSON(ctx context.Context, ws *config.Workspace) workspaceStatusReport {
	out := workspaceStatusReport{Name: ws.Name, Backend: ws.Store.Backend, Projects: []workspaceProjectStatusReport{}}
	st, err := initializeWorkspaceStore(ctx, ws)
	if err != nil {
		for _, p := range ws.Projects {
			_, statErr := os.Stat(p.Path)
			out.Projects = append(out.Projects, workspaceProjectStatusReport{
				Name:       p.Name,
				Path:       p.Path,
				PathExists: statErr == nil,
				Error:      err.Error(),
			})
		}
		return out
	}
	defer st.Close()
	for _, p := range ws.Projects {
		_, statErr := os.Stat(p.Path)
		row := workspaceProjectStatusReport{
			Name:       p.Name,
			Path:       p.Path,
			PathExists: statErr == nil,
		}
		docs, chunks, last, size, sErr := collectProjectStats(ctx, st, ws.Name, p.Name)
		if sErr != nil {
			row.Error = sErr.Error()
		} else {
			row.Documents = docs
			row.Chunks = chunks
			row.IndexSize = size
			if !last.IsZero() {
				row.LastUpdated = last.UTC().Format(time.RFC3339)
			}
		}
		out.Projects = append(out.Projects, row)
	}
	return out
}

func printWorkspaceStatusHuman(report workspaceStatusReport) {
	fmt.Printf("Workspace: %s\n", report.Name)
	fmt.Printf("  Backend: %s\n", report.Backend)
	fmt.Printf("  Projects: %d\n", len(report.Projects))
	for _, p := range report.Projects {
		exists := "✓"
		if !p.PathExists {
			exists = "✗ (path not found)"
		}
		fmt.Printf("    - %s: %s %s\n", p.Name, p.Path, exists)
		if p.Error != "" {
			fmt.Printf("        index: unavailable (%s)\n", p.Error)
			continue
		}
		fmt.Printf("        documents: %d  chunks: %d  index_size: %d\n", p.Documents, p.Chunks, p.IndexSize)
		if p.LastUpdated != "" {
			fmt.Printf("        last_updated: %s\n", p.LastUpdated)
		}
	}
}

func encodeWorkspaceStatusJSON(report workspaceStatusReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}
