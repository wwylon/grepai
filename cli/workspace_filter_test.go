package cli

import (
	"testing"

	"github.com/yoanbernabeu/grepai/config"
)

func TestFilterWorkspaceProjects(t *testing.T) {
	ws := &config.Workspace{
		Name: "demo",
		Projects: []config.ProjectEntry{
			{Name: "frontend", Path: "/a"},
			{Name: "api", Path: "/b"},
		},
	}
	if err := filterWorkspaceProjects(ws, []string{"api"}); err != nil {
		t.Fatal(err)
	}
	if len(ws.Projects) != 1 || ws.Projects[0].Name != "api" {
		t.Fatalf("%+v", ws.Projects)
	}
	ws.Projects = []config.ProjectEntry{{Name: "frontend", Path: "/a"}, {Name: "api", Path: "/b"}}
	if err := filterWorkspaceProjects(ws, []string{"missing"}); err == nil {
		t.Fatal("expected unknown project error")
	}
}
