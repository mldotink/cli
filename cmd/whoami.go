package cmd

import (
	"fmt"
	"strings"

	"github.com/mldotink/cli/internal/gql"
	"github.com/spf13/cobra"
)


var whoamiCmd = &cobra.Command{
	Use:     "whoami",
	Aliases: []string{"account"},
	Short:   "Show account info, plan, and GitHub App/OAuth connection status",
	Example: `ink whoami
ink whoami --json`,
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()

		result, err := gql.AccountStatus(ctx(), client)
		if err != nil {
			fatal(err.Error())
		}

		a := result.AccountStatus
		if a == nil {
			fatal("Could not fetch account")
		}

		if jsonOutput {
			printJSON(a)
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

		// GitHub App — required for deploying from GitHub repos
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
