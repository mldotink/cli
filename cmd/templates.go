package cmd

import (
	"fmt"
	"strings"

	"github.com/mldotink/cli/internal/gql"
	"github.com/spf13/cobra"
)

var templateCmd = &cobra.Command{
	Use:     "template [search]",
	Aliases: []string{"templates"},
	Short:   "Search and list available templates",
	Long: `Lists available service templates (e.g. PostgreSQL, Redis, MySQL).
Pass an optional search term to filter by tag.`,
	Example: `# List all templates
ink template

# Search by tag
ink template Database`,
	Args: maxArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()

		var tag *string
		if len(args) == 1 {
			tag = &args[0]
		}

		result, err := gql.TemplateList(ctx(), client, tag)
		if err != nil {
			fatal(err.Error())
		}

		templates := result.TemplateList

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
		fmt.Println(styledTable([]string{"SLUG", "NAME", "DESCRIPTION", "TAGS"}, rows))
		tableFooter(len(templates), "template")
		fmt.Println()
		fmt.Println(dim.Render("  ink template deploy <slug> --name <name>    Deploy a template"))
		fmt.Println()
	},
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
