package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	reposCreateCmd.Flags().String("host", "ink", "Git host: ink or github")
	reposCreateCmd.Flags().String("description", "", "Repository description")
	reposTokenCmd.Flags().String("host", "ink", "Git host: ink or github")
	reposCmd.AddCommand(reposCreateCmd)
	reposCmd.AddCommand(reposTokenCmd)
	rootCmd.AddCommand(reposCmd)
}

var reposCmd = &cobra.Command{
	Use:   "repos",
	Short: "Manage repositories",
}

var reposCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a repository",
	Args:  exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		host, _ := cmd.Flags().GetString("host")
		client := newClient()

		desc, _ := cmd.Flags().GetString("description")

		input := map[string]any{
			"name": name,
			"host": host,
		}
		if desc != "" {
			input["description"] = desc
		}
		addDefaults(input)

		var result struct {
			RepoCreate struct {
				Name      string `json:"name"`
				GitRemote string `json:"gitRemote"`
				ExpiresAt string `json:"expiresAt"`
				Message   string `json:"message"`
			} `json:"repoCreate"`
		}

		err := client.Do(`mutation($input: RepoCreateInput!) {
			repoCreate(input: $input) { name gitRemote expiresAt message }
		}`, map[string]any{"input": input}, &result)
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
	Args:  exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		host, _ := cmd.Flags().GetString("host")
		client := newClient()

		input := map[string]any{"name": name, "host": host}
		addDefaults(input)

		var result struct {
			RepoGetToken struct {
				GitRemote string `json:"gitRemote"`
				ExpiresAt string `json:"expiresAt"`
			} `json:"repoGetToken"`
		}

		err := client.Do(`mutation($input: RepoGetTokenInput!) {
			repoGetToken(input: $input) { gitRemote expiresAt }
		}`, map[string]any{"input": input}, &result)
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
