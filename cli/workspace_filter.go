package cli

import (
	"fmt"

	"github.com/yoanbernabeu/grepai/config"
)

func filterWorkspaceProjects(ws *config.Workspace, names []string) error {
	if ws == nil || len(names) == 0 {
		return nil
	}
	byName := make(map[string]config.ProjectEntry, len(ws.Projects))
	for _, p := range ws.Projects {
		byName[p.Name] = p
	}
	selected := make([]config.ProjectEntry, 0, len(names))
	seen := map[string]bool{}
	for _, name := range names {
		if seen[name] {
			continue
		}
		p, ok := byName[name]
		if !ok {
			return fmt.Errorf("unknown project %q in workspace %q", name, ws.Name)
		}
		selected = append(selected, p)
		seen[name] = true
	}
	ws.Projects = selected
	return nil
}
