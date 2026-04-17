package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	ink "github.com/mldotink/sdk-go"
	"github.com/spf13/cobra"
)

func init() {
	projectsDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
	projectsCmd.AddCommand(projectsCreateCmd)
	projectsCmd.AddCommand(projectsDeleteCmd)
}

var projectsCmd = &cobra.Command{
	Use:     "project",
	Aliases: []string{"projects", "proj"},
	Short:   "List and manage projects. Services are grouped into projects.",
	Example: `ink project
ink project delete my-project`,
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()

		projects, err := client.ListProjects(ctx(), cfg.Workspace)
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(projects)
			return
		}

		if len(projects) == 0 {
			fmt.Println(dim.Render("  No projects"))
		} else {
			var rows [][]string
			for _, p := range projects {
				rows = append(rows, []string{p.Name, p.Slug})
			}

			fmt.Println()
			fmt.Println(styledTable([]string{"NAME", "SLUG"}, rows))
			tableFooter(len(projects), "project")
		}

		printSubcommands(cmd)
	},
}

var projectsCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new project",
	Args:  exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		client := newClient()

		p, err := client.CreateProject(ctx(), ink.CreateProjectInput{
			Name:          name,
			WorkspaceSlug: cfg.Workspace,
		})
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(p)
			return
		}

		success(fmt.Sprintf("Project created: %s (%s)", bold.Render(p.Name), p.Slug))
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

		if err := client.DeleteProject(ctx(), slug, cfg.Workspace); err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(map[string]any{"deleted": true, "slug": slug})
			return
		}

		success(fmt.Sprintf("Project %s deleted", bold.Render(slug)))
	},
}
