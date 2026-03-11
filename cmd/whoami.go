package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mldotink/cli/internal/gql"
	"github.com/spf13/cobra"
)

func formatCents(cents int) string {
	return fmt.Sprintf("$%.2f", float64(cents)/100)
}

func formatSubtotal(s string) string {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return "$" + s
	}
	return fmt.Sprintf("$%.2f", v)
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
			result   *gql.AccountStatusResponse
			accErr   error
			usage    *gql.UsageBillBreakdownResponse
			usageErr error
			wg       sync.WaitGroup
		)

		wg.Add(2)
		go func() {
			defer wg.Done()
			result, accErr = gql.AccountStatus(ctx(), client)
		}()
		go func() {
			defer wg.Done()
			usage, usageErr = gql.UsageBillBreakdown(ctx(), client, wsPtr())
		}()
		wg.Wait()

		if accErr != nil {
			fatal(accErr.Error())
		}

		a := result.AccountStatus
		if a == nil {
			fatal("Could not fetch account")
		}

		if jsonOutput {
			out := map[string]any{"account": a}
			if usageErr == nil {
				out["usage"] = usage.UsageBillBreakdown
			}
			printJSON(out)
			return
		}

		d := newDetail("Account")
		if a.Email != nil {
			d.kv("Email", *a.Email)
		}
		d.kv("Workspace", a.DefaultWorkspace)

		tier := "free"
		if a.SubscriptionTier != nil {
			tier = *a.SubscriptionTier
		}
		d.kv("Plan", tier)

		// Usage / Billing
		if usageErr == nil {
			u := usage.UsageBillBreakdown
			period := shortDate(u.PeriodStart) + " – " + shortDate(u.PeriodEnd)
			d.blank()
			d.section("Usage (" + period + ")")
			d.kv("Current Usage", formatSubtotal(u.Subtotal))
			d.kv("Current Bill", formatCents(u.CurrentBillCents))
			if u.IncludedUsageCents > 0 {
				d.kv("Included", formatCents(u.IncludedUsageCents))
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
