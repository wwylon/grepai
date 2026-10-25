package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yoanbernabeu/grepai/config"
	"github.com/yoanbernabeu/grepai/search"
	"github.com/yoanbernabeu/grepai/store"
)

var (
	overlayName     string
	overlayUpstream string
)

var workspaceOverlayCmd = &cobra.Command{
	Use:   "overlay",
	Short: "Attach or detach a session overlay project",
	Long: `Temporarily attach one extra checkout (git worktree or feature clone)
to a workspace for search. Detach restores baseline-only membership.

Index the overlay with: grepai watch --workspace NAME --once --project OVERLAYNAME`,
}

var workspaceOverlayAttachCmd = &cobra.Command{
	Use:   "attach <workspace> <path>",
	Short: "Attach a session overlay project path",
	Args:  cobra.ExactArgs(2),
	RunE:  runOverlayAttach,
}

var workspaceOverlayDetachCmd = &cobra.Command{
	Use:   "detach <workspace>",
	Short: "Detach the session overlay for a workspace",
	Args:  cobra.ExactArgs(1),
	RunE:  runOverlayDetach,
}

var workspaceOverlayShowCmd = &cobra.Command{
	Use:   "show <workspace>",
	Short: "Show the session overlay for a workspace",
	Args:  cobra.ExactArgs(1),
	RunE:  runOverlayShow,
}

func init() {
	workspaceOverlayAttachCmd.Flags().StringVar(&overlayName, "name", "overlay", "Overlay project name")
	workspaceOverlayAttachCmd.Flags().StringVar(&overlayUpstream, "upstream", "", "Baseline project name for path-collision preference")
	workspaceOverlayCmd.AddCommand(workspaceOverlayAttachCmd)
	workspaceOverlayCmd.AddCommand(workspaceOverlayDetachCmd)
	workspaceOverlayCmd.AddCommand(workspaceOverlayShowCmd)
	workspaceCmd.AddCommand(workspaceOverlayCmd)
}

func runOverlayAttach(cmd *cobra.Command, args []string) error {
	wsName := args[0]
	abs, err := filepath.Abs(args[1])
	if err != nil {
		return err
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("overlay path must be an existing directory: %s", abs)
	}
	wsCfg, err := config.LoadWorkspaceConfig()
	if err != nil {
		return err
	}
	if _, err := wsCfg.GetWorkspace(wsName); err != nil {
		return err
	}
	cfg, err := config.LoadOverlayConfig()
	if err != nil {
		return err
	}
	meta := map[string]string{}
	if overlayUpstream != "" {
		meta["upstream"] = overlayUpstream
	}
	cfg.Set(wsName, config.OverlayEntry{Name: overlayName, Path: abs, Metadata: meta})
	if err := config.SaveOverlayConfig(cfg); err != nil {
		return err
	}
	fmt.Printf("Attached overlay %q (%s) on workspace %q\n", overlayName, abs, wsName)
	fmt.Printf("Index it with: grepai watch --workspace %s --once --project %s\n", wsName, overlayName)
	return nil
}

func runOverlayDetach(cmd *cobra.Command, args []string) error {
	wsName := args[0]
	cfg, err := config.LoadOverlayConfig()
	if err != nil {
		return err
	}
	cfg.Delete(wsName)
	if err := config.SaveOverlayConfig(cfg); err != nil {
		return err
	}
	fmt.Printf("Detached overlay from workspace %q\n", wsName)
	return nil
}

func runOverlayShow(cmd *cobra.Command, args []string) error {
	wsName := args[0]
	cfg, err := config.LoadOverlayConfig()
	if err != nil {
		return err
	}
	entry, ok := cfg.Get(wsName)
	if !ok {
		fmt.Printf("No overlay attached to workspace %q\n", wsName)
		return nil
	}
	fmt.Printf("Workspace: %s\n  name: %s\n  path: %s\n", wsName, entry.Name, entry.Path)
	if len(entry.Metadata) > 0 {
		fmt.Printf("  metadata: %v\n", entry.Metadata)
	}
	return nil
}

func mergeWorkspaceOverlay(ws *config.Workspace, workspaceName string) *config.Workspace {
	overlayCfg, err := config.LoadOverlayConfig()
	if err != nil {
		return ws
	}
	entry, ok := overlayCfg.Get(workspaceName)
	return config.MergeOverlay(ws, entry, ok)
}

func preferWorkspaceOverlay(results []store.SearchResult, ws *config.Workspace) []store.SearchResult {
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
