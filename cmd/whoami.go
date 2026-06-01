package cmd

import (
	"fmt"
	"strings"
	"sync"
	"time"

	ink "github.com/mldotink/sdk-go"
	"github.com/spf13/cobra"
)

func formatCents(cents int) string {
	return fmt.Sprintf("$%.2f", float64(cents)/100)
}

func shortDate(raw string) string {
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return raw
	}
	return t.Local().Format("Jan 2")
}

var whoamiCmd = &cobra.Command{
	Use:     "whoami",
	Aliases: []string{"account"},
	Short:   "Show account info, plan, usage, and GitHub connection status",
	Example: `ink whoami
ink whoami --json`,
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()

		var (
			account    *ink.AccountStatus
			accErr     error
			workspaces []ink.Workspace
			wsErr      error
			wg         sync.WaitGroup
		)

		wg.Add(2)
		go func() {
			defer wg.Done()
			account, accErr = client.GetAccountStatus(ctx())
		}()
		go func() {
			defer wg.Done()
			workspaces, wsErr = client.ListWorkspaces(ctx())
		}()
		wg.Wait()

		if accErr != nil {
			fatal(accErr.Error())
		}

		if account == nil {
			fatal("Could not fetch account")
		}

		type wsBilling struct {
			slug string
			data *ink.UsageBillBreakdown
		}
		var billings []wsBilling
		if wsErr == nil && len(workspaces) > 0 {
			billings = make([]wsBilling, len(workspaces))
			var bwg sync.WaitGroup
			for i, ws := range workspaces {
				billings[i].slug = ws.Slug
				bwg.Add(1)
				go func(idx int, slug string) {
					defer bwg.Done()
					billings[idx].data, _ = client.GetUsageBillBreakdown(ctx(), slug)
				}(i, ws.Slug)
			}
			bwg.Wait()
		}

		if jsonOutput {
			validation := configuredContextValidation(client)
			out := map[string]any{"account": account}
			if wsErr == nil {
				out["workspaces"] = workspaces
			}
			if len(billings) > 0 {
				bmap := make(map[string]any)
				for _, b := range billings {
					if b.data != nil {
						bmap[b.slug] = b.data
					}
				}
				out["billing"] = bmap
			}
			out["config"] = map[string]any{
				"workspace":  cfg.Workspace,
				"project":    cfg.Project,
				"api_url":    cfg.APIURL,
				"oauth_url":  cfg.OAuthURL,
				"web_url":    cfg.WebURL,
				"sources":    cfg.Sources,
				"validation": validation,
			}
			printJSON(out)
			return
		}

		d := newDetail("Account")
		if account.Email != "" {
			d.kv("Email", account.Email)
		}

		tier := "free"
		if account.SubscriptionTier != "" {
			tier = account.SubscriptionTier
		}
		d.kv("Plan", tier)

		d.blank()
		d.section("Config")
		cfgWs := cfg.Workspace
		if cfgWs == "" {
			cfgWs = dim.Render("(default)")
		}
		d.kv("Workspace", cfgWs)
		cfgProj := cfg.Project
		if cfgProj == "" {
			cfgProj = dim.Render("(default)")
		}
		d.kv("Project", cfgProj)
		if cfg.APIURL != "" {
			d.kv("API URL", cfg.APIURL)
		}
		if cfg.OAuthURL != "" {
			d.kv("OAuth URL", cfg.OAuthURL)
		}
		if cfg.WebURL != "" {
			d.kv("Web URL", cfg.WebURL)
		}
		validation := configuredContextValidation(client)
		if validation.Valid {
			d.kv("Config Valid", green.Render("yes"))
		} else {
			d.kv("Config Valid", red.Render("no")+"  "+dim.Render(validation.Error))
		}

		if wsErr == nil && len(workspaces) > 0 {
			d.blank()
			d.section("Workspaces")

			billingMap := make(map[string]*ink.UsageBillBreakdown)
			for _, b := range billings {
				if b.data != nil {
					billingMap[b.slug] = b.data
				}
			}

			maxSlug := 0
			for _, ws := range workspaces {
				if len(ws.Slug) > maxSlug {
					maxSlug = len(ws.Slug)
				}
			}

			for _, ws := range workspaces {
				slug := ws.Slug
				if ws.IsDefault {
					slug += dim.Render("*")
				}
				d.line(fmt.Sprintf("  %-*s  %s", maxSlug, slug, dim.Render(ws.Role)))
				if b, ok := billingMap[ws.Slug]; ok {
					period := dim.Render(shortDate(b.PeriodStart) + " – " + shortDate(b.PeriodEnd))
					d.line(fmt.Sprintf("    CPU %s  Mem %s  Egress %s",
						formatCents(b.CPU.TotalCents),
						formatCents(b.Memory.TotalCents),
						formatCents(b.Egress.TotalCents)))
					d.line(fmt.Sprintf("    Bill %s  %s", formatCents(b.CurrentBillCents), period))
				}
			}
		}

		d.blank()
		if account.HasGitHubApp {
			d.kv("GitHub App", green.Render("installed")+"  "+dim.Render("deploy from GitHub repos"))
		} else {
			d.kv("GitHub App", dim.Render("not installed")+"  "+dim.Render("install at ml.ink to deploy GitHub repos"))
		}

		if account.HasGitHubOAuth {
			name := ""
			if account.GitHubUsername != "" {
				name = " (" + account.GitHubUsername + ")"
			}
			d.kv("GitHub OAuth", green.Render("connected")+name+"  "+dim.Render("agents can create GitHub repos"))
		} else {
			d.kv("GitHub OAuth", dim.Render("not connected")+"  "+dim.Render("connect to let agents create GitHub repos"))
		}

		if len(account.GitHubScopes) > 0 {
			d.kv("OAuth Scopes", strings.Join(account.GitHubScopes, ", "))
		}

		fmt.Println()
		fmt.Println(d.String())
		fmt.Println()
	},
}

type configuredContextValidationResult struct {
	Workspace string `json:"workspace"`
	Project   string `json:"project,omitempty"`
	Valid     bool   `json:"valid"`
	Error     string `json:"error,omitempty"`
}

func configuredContextValidation(client *ink.Client) configuredContextValidationResult {
	workspace := configuredWorkspaceLabel()
	projects, err := client.ListProjects(ctx(), cfg.Workspace)
	if err := validateProjectSelection(workspace, cfg.Project, projects, err); err != nil {
		return configuredContextValidationResult{
			Workspace: workspace,
			Project:   cfg.Project,
			Valid:     false,
			Error:     err.Error(),
		}
	}
	return configuredContextValidationResult{
		Workspace: workspace,
		Project:   cfg.Project,
		Valid:     true,
	}
}
