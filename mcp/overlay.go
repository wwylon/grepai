package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/yoanbernabeu/grepai/config"
	"github.com/yoanbernabeu/grepai/search"
	"github.com/yoanbernabeu/grepai/stats"
	"github.com/yoanbernabeu/grepai/store"
)

func mergeMCPOverlay(ws *config.Workspace, workspaceName string) *config.Workspace {
	overlayCfg, err := config.LoadOverlayConfig()
	if err != nil {
		return ws
	}
	entry, ok := overlayCfg.Get(workspaceName)
	return config.MergeOverlay(ws, entry, ok)
}

func preferMCPOverlay(results []store.SearchResult, ws *config.Workspace) []store.SearchResult {
	overlayCfg, err := config.LoadOverlayConfig()
	if err != nil || ws == nil {
		return results
	}
	entry, ok := overlayCfg.Get(ws.Name)
	if !ok {
		return results
	}
	name := entry.Name
	if name == "" {
		name = "overlay"
	}
	upstream := ""
	if entry.Metadata != nil {
		upstream = entry.Metadata["upstream"]
	}
	return search.PreferOverlayResults(results, ws.Name, name, upstream)
}

func (s *Server) resolveWorkspaceArg(request mcp.CallToolRequest) string {
	name := request.GetString("workspace", "")
	if name == "" {
		return s.workspaceName
	}
	return name
}

func (s *Server) handleOverlayAttach(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	wsName := s.resolveWorkspaceArg(request)
	if wsName == "" {
		return mcp.NewToolResultError("workspace is required"), nil
	}
	path := request.GetString("path", "")
	if path == "" {
		return mcp.NewToolResultError("path is required"), nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return mcp.NewToolResultError(fmt.Sprintf("path must be an existing directory: %s", abs)), nil
	}
	wsCfg, err := config.LoadWorkspaceConfig()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if _, err := wsCfg.GetWorkspace(wsName); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	cfg, err := config.LoadOverlayConfig()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	name := request.GetString("name", "overlay")
	if name == "" {
		name = "overlay"
	}
	meta := map[string]string{}
	if up := request.GetString("upstream", ""); up != "" {
		meta["upstream"] = up
	}
	cfg.Set(wsName, config.OverlayEntry{Name: name, Path: abs, Metadata: meta})
	if err := config.SaveOverlayConfig(cfg); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("attached overlay %q at %s; index with grepai watch --workspace %s --once --project %s", name, abs, wsName, name)), nil
}

func (s *Server) handleOverlayDetach(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	wsName := s.resolveWorkspaceArg(request)
	if wsName == "" {
		return mcp.NewToolResultError("workspace is required"), nil
	}
	cfg, err := config.LoadOverlayConfig()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	cfg.Delete(wsName)
	if err := config.SaveOverlayConfig(cfg); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText("overlay detached"), nil
}

func (s *Server) handleGrep(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pattern, err := request.RequireString("pattern")
	if err != nil {
		return mcp.NewToolResultError("pattern is required"), nil
	}
	wsName := s.resolveWorkspaceArg(request)
	if wsName == "" {
		return mcp.NewToolResultError("workspace is required"), nil
	}
	wsCfg, err := config.LoadWorkspaceConfig()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	ws, err := wsCfg.GetWorkspace(wsName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	ws = mergeMCPOverlay(ws, wsName)
	projects := parseProjectNames(request.GetString("projects", ""))
	if len(projects) > 0 {
		selected := make([]config.ProjectEntry, 0, len(projects))
		byName := map[string]config.ProjectEntry{}
		for _, p := range ws.Projects {
			byName[p.Name] = p
		}
		for _, name := range projects {
			p, ok := byName[name]
			if !ok {
				return mcp.NewToolResultError(fmt.Sprintf("unknown project %q", name)), nil
			}
			selected = append(selected, p)
		}
		ws.Projects = selected
	}
	roots := make([]search.ExactRoot, 0, len(ws.Projects))
	for _, p := range ws.Projects {
		roots = append(roots, search.ExactRoot{Name: p.Name, Path: p.Path})
	}
	hits, err := search.GrepRoots(pattern, roots, request.GetBool("regex", false))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	format := request.GetString("format", "json")
	output, err := encodeOutput(hits, format)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	s.recordMCPStats(request, stats.Entry{
		CommandType: stats.Grep,
		OutputMode:  stats.Full,
		ResultCount: len(hits),
		Query:       pattern,
		Workspace:   wsName,
		Projects:    request.GetString("projects", ""),
	}, output)
	return mcp.NewToolResultText(output), nil
}
