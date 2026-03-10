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
	Use:     "repos",
	Short:   "Manage repositories",
}

var reposCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a repository",
	Example: `# Create an Ink-hosted repo
ink repos create myapp

# Create and push your code
ink repos create myapp
git remote add ink <remote-url>
git push ink main`,
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
	Short: "Get a fresh push token for a repo",
	Example: `ink repos token myapp`,
	Args:    exactArgs(1),
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
		kv("Expires", dim.Render(r.ExpiresAt))
		fmt.Println()
	},
}
