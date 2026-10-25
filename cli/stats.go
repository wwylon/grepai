package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/yoanbernabeu/grepai/config"
	"github.com/yoanbernabeu/grepai/stats"
)

var (
	statsJSON    bool
	statsHistory bool
	statsLimit   int
	statsNoUI    bool
	statsRecent  int
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show token savings achieved by using grepai",
	Long: `Display a summary of token savings achieved by grepai compared to
a traditional grep-based workflow.

Search and trace commands append to ~/.grepai/stats.json (shared across
projects and workspaces). Each event stores who invoked grepai (cli/mcp + actor)
and the query. Use --recent to list those events.

Set GREPAI_STATS_ACTOR or pass MCP actor to label an agent session.`,
	RunE: runStats,
}

func init() {
	rootCmd.AddCommand(statsCmd)
	statsCmd.Flags().BoolVarP(&statsJSON, "json", "j", false, "Output results in JSON format")
	statsCmd.Flags().BoolVar(&statsHistory, "history", false, "Show per-day history breakdown")
	statsCmd.Flags().IntVarP(&statsLimit, "limit", "l", 30, "Max days shown with --history")
	statsCmd.Flags().BoolVar(&statsNoUI, "no-ui", false, "Print plain text instead of interactive UI")
	statsCmd.Flags().IntVar(&statsRecent, "recent", 0, "Show the N most recent events (who/query); 0 skips")
}

func runStats(cmd *cobra.Command, args []string) error {
	var entries []stats.Entry
	if userPath, err := stats.UserStatsPath(); err == nil {
		userEntries, readErr := stats.ReadAll(userPath)
		if readErr != nil {
			return fmt.Errorf("failed to read stats: %w", readErr)
		}
		entries = userEntries
	}

	projectRoot, _ := config.FindProjectRoot()
	provider := ""
	if projectRoot != "" {
		if cfg, err := config.Load(projectRoot); err == nil {
			provider = cfg.Embedder.Provider
		}
		if len(entries) == 0 {
			local, err := stats.ReadAll(stats.StatsPath(projectRoot))
			if err != nil {
				return fmt.Errorf("failed to read stats: %w", err)
			}
			entries = local
		}
	}

	if len(entries) == 0 {
		fmt.Println("No stats recorded yet.")
		fmt.Println("Run a search or trace command to start tracking token savings.")
		return nil
	}

	summary := stats.Summarize(entries, provider)

	if statsJSON {
		return outputStatsJSON(summary, entries)
	}

	if err := outputStatsHuman(summary, entries, provider); err != nil {
		return err
	}
	printRecentEvents(entries, statsRecent)
	return nil
}

// outputStatsJSON renders the summary (and optional history) as JSON.
func outputStatsJSON(summary stats.Summary, entries []stats.Entry) error {
	out := struct {
		Summary stats.Summary      `json:"summary"`
		History []stats.DaySummary `json:"history,omitempty"`
		Recent  []stats.Entry      `json:"recent,omitempty"`
	}{
		Summary: summary,
	}
	if statsHistory {
		days := stats.HistoryByDay(entries)
		if statsLimit > 0 && len(days) > statsLimit {
			days = days[:statsLimit]
		}
		out.History = days
	}
	n := statsRecent
	if n == 0 && (statsHistory || statsJSON) {
		n = 50
	}
	if statsRecent > 0 {
		n = statsRecent
	}
	if n > 0 {
		out.Recent = stats.RecentEntries(entries, n)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func printRecentEvents(entries []stats.Entry, n int) {
	if n <= 0 {
		return
	}
	recent := stats.RecentEntries(entries, n)
	fmt.Println("\nRecent queries:")
	for _, e := range recent {
		who := e.Actor
		if who == "" {
			who = "?"
		}
		src := e.Source
		if src == "" {
			src = "?"
		}
		q := e.Query
		if q == "" {
			q = "(no query recorded)"
		}
		fmt.Printf("  %s  %s/%s  %s  %q  hits=%d  ws=%s  projects=%s\n",
			e.Timestamp, src, who, e.CommandType, q, e.ResultCount, e.Workspace, e.Projects)
	}
}

// outputStatsHuman renders the summary using lipgloss styles.
func outputStatsHuman(summary stats.Summary, entries []stats.Entry, provider string) error {
	// Styles
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Width(22)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2)

	content := headerStyle.Render("grepai stats — Token Savings Report") + "\n\n"

	content += labelStyle.Render("Total queries") + valueStyle.Render(fmt.Sprintf("%d", summary.TotalQueries)) + "\n"
	content += labelStyle.Render("Tokens (grepai)") + valueStyle.Render(formatInt(summary.OutputTokens)) + "\n"
	content += labelStyle.Render("Tokens (grep est.)") + valueStyle.Render(formatInt(summary.GrepTokens)) + "\n"
	content += labelStyle.Render("Tokens saved") +
		valueStyle.Render(fmt.Sprintf("%s  ▲ %.1f%%", formatInt(summary.TokensSaved), summary.SavingsPct)) + "\n"

	if summary.CostSavedUSD != nil {
		content += labelStyle.Render("Est. cost saved") +
			valueStyle.Render(fmt.Sprintf("$%.4f", *summary.CostSavedUSD)) +
			dimStyle.Render("  (cloud provider)") + "\n"
	}

	// Command breakdown
	content += "\n"
	cmdLine := "By command:  "
	for _, k := range []string{"search", "trace-callers", "trace-callees", "trace-graph"} {
		if v := summary.ByCommandType[k]; v > 0 {
			cmdLine += fmt.Sprintf("%s %d · ", k, v)
		}
	}
	content += dimStyle.Render(strings.TrimSuffix(cmdLine, " · ")) + "\n"

	modeLine := "By mode:     "
	for _, k := range []string{"full", "compact", "toon"} {
		if v := summary.ByOutputMode[k]; v > 0 {
			modeLine += fmt.Sprintf("%s %d · ", k, v)
		}
	}
	content += dimStyle.Render(strings.TrimSuffix(modeLine, " · ")) + "\n"

	fmt.Println(boxStyle.Render(content))

	if statsHistory {
		printHistoryTable(entries, dimStyle, valueStyle)
	}

	return nil
}

func printHistoryTable(entries []stats.Entry, dimStyle, valueStyle lipgloss.Style) {
	days := stats.HistoryByDay(entries)
	if statsLimit > 0 && len(days) > statsLimit {
		days = days[:statsLimit]
	}

	colDate := lipgloss.NewStyle().Width(14)
	colNum := lipgloss.NewStyle().Width(10)
	colSaved := lipgloss.NewStyle().Width(16)
	colPct := lipgloss.NewStyle().Width(10)

	header := dimStyle.Render(
		colDate.Render("Date") +
			colNum.Render("Queries") +
			colSaved.Render("Tokens saved") +
			colPct.Render("Savings"),
	)
	sep := dimStyle.Render(fmt.Sprintf("%-14s %-10s %-16s %-10s", "──────────────", "─────────", "───────────────", "────────"))
	fmt.Println(header)
	fmt.Println(sep)

	for _, d := range days {
		pct := 0.0
		if d.GrepTokens > 0 {
			pct = float64(d.TokensSaved) / float64(d.GrepTokens) * 100
		}
		row := colDate.Render(d.Date) +
			colNum.Render(fmt.Sprintf("%d", d.QueryCount)) +
			colSaved.Render(formatInt(d.TokensSaved)) +
			colPct.Render(fmt.Sprintf("%.1f%%", pct))
		fmt.Println(valueStyle.Render(row))
	}
}

func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	s := fmt.Sprintf("%d", n)
	result := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += ","
		}
		result += string(c)
	}
	return result
}
