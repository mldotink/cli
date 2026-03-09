package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/mldotink/ink-cli/internal/api"
	"github.com/mldotink/ink-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	jsonOutput  bool
	apiKeyFlag  string
	wsFlag      string
	projectFlag string

	cfg *config.Resolved
)

// ── Color palette ──────────────────────────────────

var (
	clrPurple  = lipgloss.Color("#7C3AED")
	clrCyan    = lipgloss.Color("#06B6D4")
	clrEmerald = lipgloss.Color("#10B981")
	clrAmber   = lipgloss.Color("#F59E0B")
	clrRose    = lipgloss.Color("#F43F5E")
	clrSlate   = lipgloss.Color("#64748B")
	clrFaint   = lipgloss.Color("#475569")
)

// ── Styles ─────────────────────────────────────────

var (
	green      = lipgloss.NewStyle().Foreground(clrEmerald)
	red        = lipgloss.NewStyle().Foreground(clrRose)
	dim        = lipgloss.NewStyle().Foreground(clrSlate)
	bold       = lipgloss.NewStyle().Bold(true)
	yellow     = lipgloss.NewStyle().Foreground(clrAmber)
	accent     = lipgloss.NewStyle().Foreground(clrCyan)
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(clrPurple)
	cardStyle  = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(clrFaint).
			Padding(1, 2)
)

// ── Status rendering ───────────────────────────────

type statusDef struct {
	dot   string
	color lipgloss.Color
}

var statusDefs = map[string]statusDef{
	"active":       {"●", clrEmerald},
	"running":      {"●", clrEmerald},
	"building":     {"◐", clrAmber},
	"deploying":    {"◐", clrAmber},
	"queued":       {"○", clrCyan},
	"provisioning": {"◐", clrCyan},
	"failed":       {"●", clrRose},
	"crashed":      {"●", clrRose},
	"error":        {"●", clrRose},
	"removed":      {"○", clrSlate},
	"pending":      {"◐", clrAmber},
	"accepted":     {"●", clrEmerald},
	"declined":     {"○", clrRose},
	"revoked":      {"○", clrSlate},
}

func renderStatus(s string) string {
	if def, ok := statusDefs[strings.ToLower(s)]; ok {
		st := lipgloss.NewStyle().Foreground(def.color)
		return st.Render(def.dot + " " + s)
	}
	return s
}

// ── Table helper ───────────────────────────────────

func styledTable(headers []string, rows [][]string) string {
	t := table.New().
		Headers(headers...).
		Rows(rows...).
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(clrFaint)).
		StyleFunc(func(row, col int) lipgloss.Style {
			s := lipgloss.NewStyle().PaddingLeft(1).PaddingRight(2)
			if row == table.HeaderRow {
				return s.Bold(true).Foreground(clrSlate)
			}
			return s
		})
	return t.String()
}

// ── Detail card ────────────────────────────────────

type detail struct {
	title string
	rows  []string
}

func newDetail(title string) *detail {
	return &detail{title: title}
}

func (d *detail) kv(label, value string) {
	styled := dim.Render(fmt.Sprintf("%-14s", label))
	d.rows = append(d.rows, styled+"  "+value)
}

func (d *detail) section(title string) {
	d.rows = append(d.rows, "", bold.Render(title))
}

func (d *detail) blank() {
	d.rows = append(d.rows, "")
}

func (d *detail) line(s string) {
	d.rows = append(d.rows, s)
}

func (d *detail) String() string {
	content := titleStyle.Render(d.title) + "\n\n" + strings.Join(d.rows, "\n")
	return cardStyle.Render(content)
}

// ── Root command ───────────────────────────────────

var rootCmd = &cobra.Command{
	Use:     "ink",
	Short:   "Deploy and manage services on Ink (ml.ink)",
	Version: "0.1.0",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cfg = config.Resolve(apiKeyFlag, wsFlag, projectFlag)

		if !jsonOutput && cmd.Name() != "login" && cmd.Name() != "help" && cmd.Name() != "completion" {
			printConfigHints()
		}
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().StringVar(&apiKeyFlag, "api-key", "", "API key (overrides config)")
	rootCmd.PersistentFlags().StringVarP(&wsFlag, "workspace", "w", "", "Workspace slug (overrides config)")
	rootCmd.PersistentFlags().StringVar(&projectFlag, "project", "", "Project slug (overrides config)")
}

func Execute() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	if err := fang.Execute(
		context.Background(),
		rootCmd,
		fang.WithNotifySignal(os.Interrupt),
	); err != nil {
		os.Exit(1)
	}
}

func newClient() *api.Client {
	if cfg.APIKey == "" {
		fatal("Not authenticated. Run: ink login")
	}
	return api.New(cfg.APIKey)
}

// ── Output helpers ─────────────────────────────────

func fatal(msg string) {
	if jsonOutput {
		json.NewEncoder(os.Stderr).Encode(map[string]string{"error": msg})
	} else {
		fmt.Fprintln(os.Stderr, red.Render("  ✕ ")+msg)
	}
	os.Exit(1)
}

func success(msg string) {
	if !jsonOutput {
		fmt.Println(green.Render("  ✓ ") + msg)
	}
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func kv(label, value string) {
	styled := dim.Render(fmt.Sprintf("%-14s", label))
	fmt.Printf("  %s  %s\n", styled, value)
}

func tableFooter(count int, noun string) {
	if count != 1 {
		noun += "s"
	}
	fmt.Println(dim.Render(fmt.Sprintf("  %d %s", count, noun)))
}

// ── Config helpers ─────────────────────────────────

func printConfigHints() {
	var hints []string
	for _, f := range []struct{ name, value, source string }{
		{"workspace", cfg.Workspace, cfg.Sources["workspace"]},
		{"project", cfg.Project, cfg.Sources["project"]},
	} {
		if f.value != "" && f.source != "" && f.source != "flag" {
			hints = append(hints, fmt.Sprintf("%s=%s", f.name, f.value))
		}
	}
	if len(hints) > 0 {
		src := cfg.Sources["workspace"]
		if src == "" {
			src = cfg.Sources["project"]
		}
		fmt.Fprintf(os.Stderr, "  %s\n", dim.Render("using "+strings.Join(hints, ", ")+" ("+sourceLabel(src)+")"))
	}
}

func sourceLabel(src string) string {
	switch src {
	case "local":
		return ".ink"
	case "global":
		return "~/.config/ink/config"
	case "env":
		return "env"
	default:
		return src
	}
}

func addDefaults(input map[string]any) {
	if cfg.Workspace != "" {
		input["workspaceSlug"] = cfg.Workspace
	}
	if cfg.Project != "" {
		input["project"] = cfg.Project
	}
}

func defaultVars() map[string]any {
	vars := make(map[string]any)
	if cfg.Workspace != "" {
		vars["ws"] = cfg.Workspace
	}
	if cfg.Project != "" {
		vars["proj"] = cfg.Project
	}
	return vars
}

func mergeVars(extra map[string]any) map[string]any {
	vars := defaultVars()
	for k, v := range extra {
		vars[k] = v
	}
	return vars
}

func deref(s *string, fallback string) string {
	if s != nil {
		return *s
	}
	return fallback
}

// ── Arg validators (show help on error) ───────────

func exactArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != n {
			cmd.Help()
			fmt.Println()
			os.Exit(1)
		}
		return nil
	}
}

func rangeArgs(min, max int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) < min || len(args) > max {
			cmd.Help()
			fmt.Println()
			os.Exit(1)
		}
		return nil
	}
}

func maxArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) > n {
			cmd.Help()
			fmt.Println()
			os.Exit(1)
		}
		return nil
	}
}
