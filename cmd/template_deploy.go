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
	templateDeployCmd.Flags().String("name", "", "Instance name (required)")
	templateDeployCmd.Flags().StringArray("var", nil, "Template variable as KEY=VALUE (repeatable)")
	templateCmd.AddCommand(templateDeployCmd)
}

var templateDeployCmd = &cobra.Command{
	Use:   "deploy <slug>",
	Short: "Deploy a template",
	Long: `Deploys a template (e.g. postgresql, redis) creating all services
and returning connection credentials.

Use "ink template info <slug>" to preview required variables before deploying.`,
	Example: `# Preview template variables
ink template info postgres

# Deploy PostgreSQL
ink template deploy postgres --name mydb

# Deploy with variables
ink template deploy postgres --name mydb --var db_name=myapp --var storage_gi=20`,
	Args: exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		slug := args[0]
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			fmt.Fprintln(os.Stderr, "Error: --name is required")
			fmt.Fprintln(os.Stderr, "Usage: ink template deploy", slug, "--name <instance-name> [--var KEY=VALUE ...]")
			fmt.Fprintln(os.Stderr, "\nRun \"ink template info "+slug+"\" to see available variables.")
			os.Exit(1)
		}

		client := newClient()

		templates, err := client.ListTemplates(ctx(), "")
		if err != nil {
			fatal(err.Error())
		}

		var tmpl *ink.Template
		for i := range templates {
			if templates[i].Slug == slug {
				tmpl = &templates[i]
				break
			}
		}
		if tmpl == nil {
			fatal(fmt.Sprintf("Template %q not found", slug))
		}

		varFlags, _ := cmd.Flags().GetStringArray("var")
		vars := make(map[string]string)
		for _, v := range varFlags {
			if k, val, ok := strings.Cut(v, "="); ok {
				vars[k] = val
			}
		}

		reader := bufio.NewReader(os.Stdin)
		for _, v := range tmpl.Variables {
			if _, ok := vars[v.Key]; ok {
				continue
			}
			if !v.Required && v.DefaultValue != "" {
				continue
			}
			if !v.Required {
				continue
			}
			prompt := fmt.Sprintf("  %s", v.Name)
			if v.Description != "" {
				prompt += fmt.Sprintf(" (%s)", v.Description)
			}
			if v.DefaultValue != "" {
				prompt += fmt.Sprintf(" [%s]", v.DefaultValue)
			}
			prompt += ": "
			fmt.Print(prompt)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input == "" && v.DefaultValue != "" {
				input = v.DefaultValue
			}
			if input != "" {
				vars[v.Key] = input
			}
		}

		var variables []ink.TemplateVariableValue
		for k, v := range vars {
			variables = append(variables, ink.TemplateVariableValue{Key: k, Value: v})
		}

		deploy, err := client.DeployTemplate(ctx(), ink.TemplateDeployInput{
			Template:      slug,
			Name:          name,
			Project:       cfg.Project,
			WorkspaceSlug: cfg.Workspace,
			Variables:     variables,
		})
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(deploy)
			return
		}

		fmt.Println()
		success(fmt.Sprintf("Template %s deployed: %s", bold.Render(slug), bold.Render(name)))

		if len(deploy.Services) > 0 {
			var rows [][]string
			for _, s := range deploy.Services {
				endpoint := dim.Render("—")
				for _, ep := range s.Endpoints {
					if ep.PublicEndpoint != "" {
						endpoint = accent.Render(ep.PublicEndpoint)
						break
					}
				}
				rows = append(rows, []string{s.Name, renderStatus(s.Status), endpoint})
			}
			fmt.Println()
			fmt.Println(styledTable([]string{"SERVICE", "STATUS", "ENDPOINT"}, rows))
		}

		if len(deploy.Outputs) > 0 {
			fmt.Println()
			fmt.Println(bold.Render("  Outputs"))
			for _, o := range deploy.Outputs {
				fmt.Printf("  %-20s %s\n", dim.Render(o.Label), o.Value)
			}
		}

		if len(deploy.Services) > 0 {
			svcName := deploy.Services[0].Name
			fmt.Println()
			fmt.Println(dim.Render(fmt.Sprintf("  ink service %s                     Check status and endpoints", svcName)))
			fmt.Println(dim.Render(fmt.Sprintf("  ink service %s --runtime-logs 20   View logs", svcName)))
		}

		fmt.Println()
	},
}
