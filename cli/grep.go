package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yoanbernabeu/grepai/config"
	"github.com/yoanbernabeu/grepai/search"
)

var (
	grepWorkspace string
	grepProjects  []string
	grepRegex     bool
	grepJSON      bool
)

var grepCmd = &cobra.Command{
	Use:   "grep <pattern>",
	Short: "Exact or regex search across workspace project roots",
	Long: `Search the working trees of workspace members for a literal or regex pattern.

This complements semantic 'grepai search': it does not call the embedder.
Results reflect files on disk (prefer ripgrep when installed).

  grepai grep AuthToken --workspace my-fullstack
  grepai grep 'func Handle' --workspace my-fullstack --regex --project api`,
	Args: cobra.ExactArgs(1),
	RunE: runGrep,
}

func init() {
	grepCmd.Flags().StringVar(&grepWorkspace, "workspace", "", "Workspace name (required)")
	grepCmd.Flags().StringArrayVar(&grepProjects, "project", nil, "Limit to these workspace projects (repeatable)")
	grepCmd.Flags().BoolVar(&grepRegex, "regex", false, "Treat pattern as a regular expression")
	grepCmd.Flags().BoolVar(&grepJSON, "json", false, "JSON output")
	_ = grepCmd.MarkFlagRequired("workspace")
	rootCmd.AddCommand(grepCmd)
}

func runGrep(cmd *cobra.Command, args []string) error {
	wsCfg, err := config.LoadWorkspaceConfig()
	if err != nil {
		return err
	}
	ws, err := wsCfg.GetWorkspace(grepWorkspace)
	if err != nil {
		return err
	}
	if err := filterGrepProjects(ws, grepProjects); err != nil {
		return err
	}
	roots := make([]search.ExactRoot, 0, len(ws.Projects))
	for _, p := range ws.Projects {
		roots = append(roots, search.ExactRoot{Name: p.Name, Path: p.Path})
	}
	hits, err := search.GrepRoots(args[0], roots, grepRegex)
	if err != nil {
		return err
	}
	if grepJSON {
		data, err := json.MarshalIndent(hits, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}
	for _, h := range hits {
		fmt.Printf("%s:%s:%d:%s\n", h.Project, h.Path, h.Line, h.Text)
	}
	return nil
}

func filterGrepProjects(ws *config.Workspace, names []string) error {
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
