package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yoanbernabeu/grepai/config"
	"github.com/yoanbernabeu/grepai/embedder"
	"github.com/yoanbernabeu/grepai/search"
	"github.com/yoanbernabeu/grepai/stats"
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
	overlayCfg, overlayErr := config.LoadOverlayConfig()
	if overlayErr == nil {
		entry, ok := overlayCfg.Get(grepWorkspace)
		ws = config.MergeOverlay(ws, entry, ok)
	}
	if err := filterWorkspaceProjects(ws, grepProjects); err != nil {
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
	var outputStr string
	if grepJSON {
		data, err := json.MarshalIndent(hits, "", "  ")
		if err != nil {
			return err
		}
		outputStr = string(data)
		fmt.Println(outputStr)
	} else {
		var b strings.Builder
		for _, h := range hits {
			fmt.Fprintf(&b, "%s:%s:%d:%s\n", h.Project, h.Path, h.Line, h.Text)
		}
		outputStr = b.String()
		fmt.Print(outputStr)
	}
	e := stats.Entry{
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		CommandType: stats.Grep,
		OutputMode:  stats.Full,
		ResultCount: len(hits),
		Query:       args[0],
		Workspace:   grepWorkspace,
		Projects:    strings.Join(grepProjects, ","),
	}
	e.OutputTokens = embedder.EstimateTokens(outputStr)
	e.GrepTokens = stats.GrepEquivalentTokens(len(hits))
	stats.FillIdentity(&e, stats.SourceCLI)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = stats.NewUserRecorder("").Record(ctx, e)
	return nil
}
