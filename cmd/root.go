package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/mldotink/cli/internal/api"
	"github.com/mldotink/cli/internal/config"
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

var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "ink",
	Short: "Deploy apps and services to the cloud in seconds (ml.ink)",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cfg = config.Resolve(apiKeyFlag, wsFlag, projectFlag)

		switch cmd.Name() {
		case "login", "help", "completion", "workspace", "workspaces", "whoami", "account", "config", "update":
			// These commands don't operate within a workspace/project scope.
		default:
			if !jsonOutput {
				printConfigHints()
			}
		}
	},
}

func init() {
	cobra.EnableCommandSorting = false

	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().StringVar(&apiKeyFlag, "api-key", "", "API key (overrides config)")
	rootCmd.PersistentFlags().StringVarP(&wsFlag, "workspace", "w", "", "Workspace slug (overrides config)")
	rootCmd.PersistentFlags().StringVar(&projectFlag, "project", "", "Project slug (overrides config)")

}

func Execute() {
	registerCommands()
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	opts := []fang.Option{
		fang.WithNotifySignal(os.Interrupt),
	}
	if Version != "dev" {
		opts = append(opts, fang.WithVersion(Version))
	}

	if err := fang.Execute(
		context.Background(),
		rootCmd,
		opts...,
	); err != nil {
		os.Exit(1)
	}
}

func newClient() graphql.Client {
	if cfg.APIKey == "" {
		fatal("Not authenticated. Run: ink login")
	}
	return api.NewClient(cfg.APIKey)
}

func ctx() context.Context {
	return context.Background()
}

// wsPtr returns a pointer to workspace slug, or nil if unset.
func wsPtr() *string {
	if cfg.Workspace == "" {
		return nil
	}
	return &cfg.Workspace
}

// projPtr returns a pointer to project slug, or nil if unset.
func projPtr() *string {
	if cfg.Project == "" {
		return nil
	}
	return &cfg.Project
}

func ptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func deref(s *string, fallback string) string {
	if s != nil {
		return *s
	}
	return fallback
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

// ── Subcommand footer ─────────────────────────────

func printSubcommands(cmd *cobra.Command) {
	fmt.Println()
	fmt.Println(bold.Render("  Commands"))
	for _, sub := range cmd.Commands() {
		if !sub.Hidden {
			fmt.Printf("    %-20s %s\n", accent.Render(sub.Name()), dim.Render(sub.Short))
		}
	}
	if cmd.HasExample() {
		fmt.Println()
		fmt.Println(bold.Render("  Examples"))
		for _, line := range strings.Split(cmd.Example, "\n") {
			if strings.HasPrefix(line, "#") {
				fmt.Println("    " + dim.Render(line))
			} else {
				fmt.Println("    " + line)
			}
		}
	}
	fmt.Println()
	fmt.Println(dim.Render(fmt.Sprintf("  Use \"ink %s <command> --help\" for more information.", cmd.Name())))
	fmt.Println()
}

// ── Time formatting ───────────────────────────────

func fmtTime(raw string) string {
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return raw
	}
	return t.Local().Format("2006-01-02 15:04:05")
}

// ── Config helpers ─────────────────────────────────

func printConfigHints() {
	ws := cfg.Workspace
	if ws == "" {
		ws = "default"
	}

	line := fmt.Sprintf("  %s workspace=%s", dim.Render("▸"), ws)
	if cfg.Project != "" {
		line += fmt.Sprintf("  project=%s", cfg.Project)
	}

	src := cfg.Sources["workspace"]
	if src == "" {
		src = cfg.Sources["project"]
	}
	if src != "" {
		line += "  " + dim.Render("("+sourceLabel(src)+")")
	}

	fmt.Fprintln(os.Stderr, line)
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
