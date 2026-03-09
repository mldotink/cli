package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	projectsDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
	projectsCmd.AddCommand(projectsDeleteCmd)
	rootCmd.AddCommand(projectsCmd)
}

var projectsCmd = &cobra.Command{
	Use:     "projects",
	Aliases: []string{"proj"},
	Short:   "List projects",
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()

		var result struct {
			ProjectList struct {
				Nodes []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
					Slug string `json:"slug"`
				} `json:"nodes"`
				TotalCount int `json:"totalCount"`
			} `json:"projectList"`
		}

		err := client.Do(`query($ws: String) {
			projectList(workspaceSlug: $ws) {
				nodes { id name slug }
				totalCount
			}
		}`, defaultVars(), &result)
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(result.ProjectList)
			return
		}

		nodes := result.ProjectList.Nodes
		if len(nodes) == 0 {
			fmt.Println(dim.Render("  No projects"))
			return
		}

		var rows [][]string
		for _, p := range nodes {
			rows = append(rows, []string{p.Name, p.Slug})
		}

		fmt.Println()
		fmt.Println(styledTable([]string{"NAME", "SLUG"}, rows))
		tableFooter(len(nodes), "project")
		fmt.Println()
	},
}

var projectsDeleteCmd = &cobra.Command{
	Use:   "delete <slug>",
	Short: "Delete a project and all its services",
	Args:  exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		slug := args[0]
		yes, _ := cmd.Flags().GetBool("yes")

		if !yes && !jsonOutput {
			fmt.Printf("Delete project %s and all its services? [y/N] ", bold.Render(slug))
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				if strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" {
					fmt.Println(dim.Render("  Cancelled"))
					return
				}
			}
		}

		client := newClient()

		var result struct {
			ProjectDelete bool `json:"projectDelete"`
		}

		err := client.Do(`mutation($slug: String!, $ws: String) {
			projectDelete(slug: $slug, workspaceSlug: $ws)
		}`, mergeVars(map[string]any{"slug": slug}), &result)
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(map[string]any{"deleted": result.ProjectDelete, "slug": slug})
			return
		}

		success(fmt.Sprintf("Project %s deleted", bold.Render(slug)))
	},
}
