package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(whoamiCmd)
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current account",
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()

		var result struct {
			AccountStatus *struct {
				ID               string  `json:"id"`
				Email            *string `json:"email"`
				GithubUsername   *string `json:"githubUsername"`
				HasGitHubOAuth   bool    `json:"hasGitHubOAuth"`
				HasGitHubApp     bool    `json:"hasGitHubApp"`
				DefaultWorkspace string  `json:"defaultWorkspace"`
				SubscriptionTier *string `json:"subscriptionTier"`
			} `json:"accountStatus"`
		}

		err := client.Do(`{
			accountStatus {
				id email githubUsername hasGitHubOAuth hasGitHubApp
				defaultWorkspace subscriptionTier
			}
		}`, nil, &result)
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

		gh := dim.Render("not connected")
		if a.HasGitHubOAuth {
			name := ""
			if a.GithubUsername != nil {
				name = " (" + *a.GithubUsername + ")"
			}
			if a.HasGitHubApp {
				gh = green.Render("connected") + name
			} else {
				gh = "OAuth only" + name + dim.Render(" — install GitHub App at ml.ink")
			}
		}
		d.kv("GitHub", gh)

		fmt.Println()
		fmt.Println(d.String())
		fmt.Println()
	},
}
