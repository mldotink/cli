package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/mldotink/cli/internal/gql"
	"github.com/spf13/cobra"
)

func init() {
	projectsDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
	projectsCmd.AddCommand(projectsDeleteCmd)
}

var projectsCmd = &cobra.Command{
	GroupID: "manage",
	Use:     "projects",
	Aliases: []string{"proj"},
	Short:   "List projects",
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()

		result, err := gql.ListProjects(ctx(), client, wsPtr())
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
		} else {
			var rows [][]string
			for _, p := range nodes {
				rows = append(rows, []string{p.Name, p.Slug})
			}

			fmt.Println()
			fmt.Println(styledTable([]string{"NAME", "SLUG"}, rows))
			tableFooter(len(nodes), "project")
		}

		fmt.Println()
		fmt.Println(dim.Render("Available Commands:"))
		for _, sub := range cmd.Commands() {
			if !sub.Hidden {
				fmt.Printf("  %-20s %s\n", sub.Name(), sub.Short)
			}
		}
		fmt.Println()
		fmt.Println(dim.Render(fmt.Sprintf("Use \"ink projects <command> --help\" for more information about a command.")))
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

		result, err := gql.DeleteProject(ctx(), client, slug, wsPtr())
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
