package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var templateCmd = &cobra.Command{
	Use:     "template [search]",
	Aliases: []string{"templates"},
	Short:   "Deploy pre-configured stacks with a single command",
	Long:    `Deploy pre-configured stacks with a single command.`,
	Example: `# List all templates
ink template

# Search templates
ink template postgres
ink template database`,
	Args: maxArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()

		search := ""
		if len(args) == 1 {
			search = args[0]
		}

		templates, err := client.ListTemplates(ctx(), search)
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(templates)
			return
		}

		if len(templates) == 0 {
			fmt.Println(dim.Render("  No templates found"))
			return
		}

		var rows [][]string
		for _, t := range templates {
			rows = append(rows, []string{
				t.Slug,
				t.Name,
				truncate(t.Description, 50),
				strings.Join(t.Tags, ", "),
			})
		}

		fmt.Println()
		fmt.Println(dim.Render("  Deploy pre-configured stacks with a single command."))
		fmt.Println()
		fmt.Println(styledTable([]string{"SLUG", "NAME", "DESCRIPTION", "TAGS"}, rows))
		tableFooter(len(templates), "template")
		fmt.Println()
		fmt.Println(dim.Render("  ink template info <slug>        View template details and variables"))
		fmt.Println(dim.Render("  ink template deploy <slug>      Deploy a template"))
		fmt.Println()
	},
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
