package cmd

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mldotink/cli/internal/gql"
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
			result *gql.AccountStatusResponse
			accErr error
			wsList *gql.ListWorkspacesResponse
			wsErr  error
			wg     sync.WaitGroup
		)

		wg.Add(2)
		go func() {
			defer wg.Done()
			result, accErr = gql.AccountStatus(ctx(), client)
		}()
		go func() {
			defer wg.Done()
			wsList, wsErr = gql.ListWorkspaces(ctx(), client)
		}()
		wg.Wait()

		if accErr != nil {
			fatal(accErr.Error())
		}

		a := result.AccountStatus
		if a == nil {
			fatal("Could not fetch account")
		}

		// Fetch billing for each workspace in parallel
		type wsBilling struct {
			slug string
			data *gql.UsageBillBreakdownResponse
		}
		var billings []wsBilling
		if wsErr == nil && len(wsList.WorkspaceList) > 0 {
			billings = make([]wsBilling, len(wsList.WorkspaceList))
			var bwg sync.WaitGroup
			for i, ws := range wsList.WorkspaceList {
				billings[i].slug = ws.Slug
				bwg.Add(1)
				go func(idx int, slug string) {
					defer bwg.Done()
					s := slug
					billings[idx].data, _ = gql.UsageBillBreakdown(ctx(), client, &s)
				}(i, ws.Slug)
			}
			bwg.Wait()
		}

		if jsonOutput {
			out := map[string]any{"account": a}
			if wsErr == nil {
				out["workspaces"] = wsList.WorkspaceList
			}
			if len(billings) > 0 {
				bmap := make(map[string]any)
				for _, b := range billings {
					if b.data != nil {
						bmap[b.slug] = b.data.UsageBillBreakdown
					}
				}
				out["billing"] = bmap
			}
			out["config"] = map[string]string{
				"workspace": cfg.Workspace,
				"project":   cfg.Project,
			}
			printJSON(out)
			return
		}

		d := newDetail("Account")
		if a.Email != nil {
			d.kv("Email", *a.Email)
		}

		tier := "free"
		if a.SubscriptionTier != nil {
			tier = *a.SubscriptionTier
		}
		d.kv("Plan", tier)

		// Config
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

		// Workspaces
		if wsErr == nil && len(wsList.WorkspaceList) > 0 {
			d.blank()
			d.section("Workspaces")

			// Build billing lookup
			billingMap := make(map[string]*gql.UsageBillBreakdownResponse)
			for _, b := range billings {
				if b.data != nil {
					billingMap[b.slug] = b.data
				}
			}

			for _, ws := range wsList.WorkspaceList {
				role := dim.Render(ws.Role)
				line := role
				if b, ok := billingMap[ws.Slug]; ok {
					u := b.UsageBillBreakdown
					period := shortDate(u.PeriodStart) + " – " + shortDate(u.PeriodEnd)
					line += "  " + formatCents(u.CurrentBillCents) + "  " + dim.Render(period)
				}
				d.kv(ws.Slug, line)
			}
		}

		// GitHub App — required for deploying from GitHub repos
		d.blank()
		if a.HasGitHubApp {
			d.kv("GitHub App", green.Render("installed")+"  "+dim.Render("deploy from GitHub repos"))
		} else {
			d.kv("GitHub App", dim.Render("not installed")+"  "+dim.Render("install at ml.ink to deploy GitHub repos"))
		}

		// GitHub OAuth — enables agents to create repos on your behalf
		if a.HasGitHubOAuth {
			name := ""
			if a.GithubUsername != nil {
				name = " (" + *a.GithubUsername + ")"
			}
			d.kv("GitHub OAuth", green.Render("connected")+name+"  "+dim.Render("agents can create GitHub repos"))
		} else {
			d.kv("GitHub OAuth", dim.Render("not connected")+"  "+dim.Render("connect to let agents create GitHub repos"))
		}

		// GitHub scopes
		if len(a.GithubScopes) > 0 {
			d.kv("OAuth Scopes", strings.Join(a.GithubScopes, ", "))
		}

		fmt.Println()
		fmt.Println(d.String())
		fmt.Println()
	},
}
