package cmd

import (
	"fmt"

	"github.com/mldotink/cli/internal/gql"
	"github.com/spf13/cobra"
)

func init() {
	reposCreateCmd.Flags().String("host", "ink", "Git host: ink, github")
	reposCreateCmd.Flags().String("description", "", "Repository description")
	reposTokenCmd.Flags().String("host", "ink", "Git host: ink, github")
	reposCmd.AddCommand(reposCreateCmd)
	reposCmd.AddCommand(reposTokenCmd)
}

var reposCmd = &cobra.Command{
	Use:     "repo",
	Aliases: []string{"repos"},
	Short:   "Create repos and get git remotes for pushing code",
	Long: `Manage git repositories in your workspace. There are three ways to deploy code:

  1. Ink-managed repo (no GitHub needed)
     Create a repo on Ink's git server in your workspace, push code, deploy.

  2. GitHub App (you manage the GitHub repo)
     Install the Ink GitHub App on your GitHub account, create a repo yourself
     (e.g. 'gh repo create'), push code, then deploy with --host github.
     The App lets Ink pull your code — no OAuth needed.

  3. GitHub App + OAuth (agents create repos for you)
     With both GitHub App and OAuth connected, AI agents can create GitHub
     repos on your behalf via 'ink repo create --host github'.`,
	Example: `# Ink-managed repo — no GitHub needed
ink repo create myapp
git remote add ink <remote-url-from-output>
git push ink main
ink deploy myapp

# GitHub App — you create the repo, Ink pulls from it
gh repo create myorg/myapi --private
git push origin main
ink deploy myapi --host github --repo myorg/myapi

# GitHub App + OAuth — agent creates the GitHub repo
ink repo create myapi --host github
ink deploy myapi --host github --repo myorg/myapi

# Get a fresh push token for an Ink-managed repo
ink repo token myapp`,
}

var reposCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a git repository in your workspace",
	Long: `Creates a git repository in your workspace. With --host ink (default), creates
a private repo on Ink's git server (git.ml.ink) and returns a remote URL with
a push token. With --host github, creates a GitHub repo (requires OAuth).

For GitHub repos without OAuth, use 'gh repo create' yourself and deploy
with 'ink deploy --host github --repo user/repo' (only requires GitHub App).`,
	Example: `# Create an Ink-managed repo in your workspace
ink repo create myapp

# Push your code using the remote URL from output
git remote add ink <remote-url>
git push ink main

# Create a GitHub repo (requires OAuth)
ink repo create myapi --host github`,
	Args: exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		host, _ := cmd.Flags().GetString("host")
		client := newClient()

		desc, _ := cmd.Flags().GetString("description")

		input := gql.RepoCreateInput{
			Name:          name,
			Host:          ptr(host),
			WorkspaceSlug: wsPtr(),
			Project:       projPtr(),
		}
		if desc != "" {
			input.Description = ptr(desc)
		}

		result, err := gql.CreateRepo(ctx(), client, input)
		if err != nil {
			fatal(err.Error())
		}

		r := result.RepoCreate
		if jsonOutput {
			printJSON(r)
			return
		}

		fmt.Println()
		success(fmt.Sprintf("Repository created: %s", bold.Render(r.Name)))
		kv("Remote", accent.Render(r.GitRemote))
		fmt.Println()
		fmt.Println(dim.Render("  Push your code:"))
		fmt.Printf("  git remote add ink %s\n", r.GitRemote)
		fmt.Println("  git push ink main")
		fmt.Println()
	},
}

var reposTokenCmd = &cobra.Command{
	Use:   "token <name>",
	Short: "Get a fresh push token for an Ink-managed repo",
	Long: `Returns a fresh remote URL with a new push token. Tokens expire periodically,
so use this when a previous push token has expired.

The returned URL can be used directly with 'git remote set-url' or 'git push'.`,
	Example: `# Get a fresh remote URL
ink repo token myapp

# Update your existing remote
git remote set-url ink $(ink repo token myapp --json | jq -r .gitRemote)`,
	Args: exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		host, _ := cmd.Flags().GetString("host")
		client := newClient()

		input := gql.RepoGetTokenInput{
			Name:          name,
			Host:          ptr(host),
			WorkspaceSlug: wsPtr(),
		}

		result, err := gql.GetRepoToken(ctx(), client, input)
		if err != nil {
			fatal(err.Error())
		}

		r := result.RepoGetToken
		if jsonOutput {
			printJSON(r)
			return
		}

		fmt.Println()
		kv("Remote", accent.Render(r.GitRemote))
		kv("Expires", dim.Render(fmtTime(r.ExpiresAt)))
		fmt.Println()
	},
}
