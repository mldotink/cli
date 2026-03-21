package cmd

import (
	"fmt"
	"strings"

	"github.com/mldotink/cli/internal/gql"
	"github.com/spf13/cobra"
)

func init() {
	templateCmd.AddCommand(templateInfoCmd)
}

var templateInfoCmd = &cobra.Command{
	Use:   "info <slug>",
	Short: "View template details and variables",
	Long:  `Shows full details for a template including variables, services, and example deploy commands.`,
	Example: `ink template info postgres
ink template info redis`,
	Args: exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		slug := args[0]
		client := newClient()

		result, err := gql.TemplateList(ctx(), client, nil)
		if err != nil {
			fatal(err.Error())
		}

		var tmpl *gql.TemplateListTemplateListServiceTemplate
		for i := range result.TemplateList {
			if result.TemplateList[i].Slug == slug {
				tmpl = &result.TemplateList[i]
				break
			}
		}
		if tmpl == nil {
			fatal(fmt.Sprintf("Template %q not found", slug))
		}

		if jsonOutput {
			printJSON(tmpl)
			return
		}

		// Detail card
		d := newDetail(tmpl.Name)
		d.kv("Slug", tmpl.Slug)
		d.kv("Description", tmpl.Description)
		d.kv("Tags", strings.Join(tmpl.Tags, ", "))

		// Variables
		if len(tmpl.Variables) > 0 {
			d.section("Variables")
			for _, v := range tmpl.Variables {
				label := v.Key
				meta := v.Type
				if v.Required {
					meta += ", required"
				}
				if v.Name != "" {
					meta += " — " + v.Name
				}
				d.line(fmt.Sprintf("  %-18s %s", accent.Render(label), dim.Render(meta)))
				if v.DefaultValue != nil {
					d.line(fmt.Sprintf("  %-18s default: %s", "", *v.DefaultValue))
				}
			}
		}

		// Services
		if len(tmpl.Services) > 0 {
			d.section("Services")
			for _, s := range tmpl.Services {
				d.line(fmt.Sprintf("  %-18s %s  %s  %s vCPU",
					accent.Render(s.Key),
					s.Image,
					dim.Render(s.Memory),
					dim.Render(s.Vcpus),
				))
			}
		}

		fmt.Println()
		fmt.Println(d.String())

		// Example deploy commands
		fmt.Println()
		fmt.Println(dim.Render(fmt.Sprintf("  ink template deploy %s --name my%s", tmpl.Slug, tmpl.Slug)))

		// Show example with a variable if there are any
		if len(tmpl.Variables) > 0 {
			v := tmpl.Variables[0]
			example := "value"
			if v.DefaultValue != nil {
				example = *v.DefaultValue
			}
			fmt.Println(dim.Render(fmt.Sprintf("  ink template deploy %s --name my%s --var %s=%s", tmpl.Slug, tmpl.Slug, v.Key, example)))
		}
		fmt.Println()
	},
}
