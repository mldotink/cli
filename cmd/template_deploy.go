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
	templateDeployCmd.Flags().String("name", "", "Instance name (required)")
	templateDeployCmd.Flags().StringArray("var", nil, "Template variable as KEY=VALUE (repeatable)")
	templateCmd.AddCommand(templateDeployCmd)
}

var templateDeployCmd = &cobra.Command{
	Use:   "deploy <slug>",
	Short: "Deploy a template",
	Long: `Deploys a template (e.g. postgresql, redis) creating all services
and returning connection credentials.`,
	Example: `# Deploy PostgreSQL
ink template deploy postgresql --name mydb

# Deploy with variables
ink template deploy postgresql --name mydb --var db_name=myapp --var storage_gi=20`,
	Args: exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		slug := args[0]
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			fatal("--name is required")
		}

		client := newClient()

		// Fetch template to know required variables
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

		// Collect variables from --var flags
		varFlags, _ := cmd.Flags().GetStringArray("var")
		vars := make(map[string]string)
		for _, v := range varFlags {
			if k, val, ok := strings.Cut(v, "="); ok {
				vars[k] = val
			}
		}

		// Prompt for missing required variables
		reader := bufio.NewReader(os.Stdin)
		for _, v := range tmpl.Variables {
			if _, ok := vars[v.Key]; ok {
				continue
			}
			if !v.Required && v.DefaultValue != nil {
				continue
			}
			if !v.Required {
				continue
			}
			prompt := fmt.Sprintf("  %s", v.Name)
			if v.Description != "" {
				prompt += fmt.Sprintf(" (%s)", v.Description)
			}
			if v.DefaultValue != nil {
				prompt += fmt.Sprintf(" [%s]", *v.DefaultValue)
			}
			prompt += ": "
			fmt.Print(prompt)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input == "" && v.DefaultValue != nil {
				input = *v.DefaultValue
			}
			if input != "" {
				vars[v.Key] = input
			}
		}

		// Build variable inputs
		var variables []gql.TemplateVariableValueInput
		for k, v := range vars {
			variables = append(variables, gql.TemplateVariableValueInput{Key: k, Value: v})
		}

		input := gql.TemplateDeployInput{
			Template:      slug,
			Name:          name,
			Project:       projPtr(),
			WorkspaceSlug: wsPtr(),
			Variables:     variables,
		}

		deployResult, err := gql.TemplateDeploy(ctx(), client, input)
		if err != nil {
			fatal(err.Error())
		}

		deploy := deployResult.TemplateDeploy

		if jsonOutput {
			printJSON(deploy)
			return
		}

		fmt.Println()
		success(fmt.Sprintf("Template %s deployed: %s", bold.Render(slug), bold.Render(name)))

		// Services table
		if len(deploy.Services) > 0 {
			var rows [][]string
			for _, s := range deploy.Services {
				endpoint := dim.Render("—")
				for _, ep := range s.Endpoints {
					if ep.PublicEndpoint != nil {
						endpoint = accent.Render(*ep.PublicEndpoint)
						break
					}
				}
				rows = append(rows, []string{s.Name, renderStatus(s.Status), endpoint})
			}
			fmt.Println()
			fmt.Println(styledTable([]string{"SERVICE", "STATUS", "ENDPOINT"}, rows))
		}

		// Outputs
		if len(deploy.Outputs) > 0 {
			fmt.Println()
			fmt.Println(bold.Render("  Outputs"))
			for _, o := range deploy.Outputs {
				val := o.Value
				fmt.Printf("  %-20s %s\n", dim.Render(o.Label), val)
			}
		}

		fmt.Println()
	},
}
