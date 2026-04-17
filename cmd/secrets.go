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
	secretsImportCmd.Flags().String("file", "", "Read env vars from file (default: stdin)")
	secretsImportCmd.Flags().Bool("replace", false, "Replace all vars (remove unspecified)")

	secretsSetCmd.Flags().Bool("replace", false, "Replace all vars (remove unspecified)")

	secretsCmd.AddCommand(secretsImportCmd)
	secretsCmd.AddCommand(secretsSetCmd)
	secretsCmd.AddCommand(secretsListCmd)
	secretsCmd.AddCommand(secretsDeleteCmd)
	secretsCmd.AddCommand(secretsUnsetCmd)
}

var secretsCmd = &cobra.Command{
	Use:     "secret",
	Aliases: []string{"secrets", "env"},
	Short:   "Set, list, and remove environment variables and secrets on a service",
	Example: `ink secret import myapi --file .env
cat .env | ink secret import myapi
ink secret set myapi KEY1=value1 KEY2=value2
ink secret list myapi
ink secret unset myapi DATABASE_URL
ink secret delete myapi KEY1 KEY2`,
}

var secretsImportCmd = &cobra.Command{
	Use:   "import <service>",
	Short: "Import env vars from file or stdin",
	Long:  "Reads KEY=VALUE pairs and merges them with existing vars. Triggers a redeploy.",
	Example: `# From file
ink secret import myapi --file .env

# From stdin
cat .env | ink secret import myapi

# Replace all vars (removes unspecified)
ink secret import myapi --file .env --replace`,
	Args: exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		client := newClient()

		file, _ := cmd.Flags().GetString("file")
		replace, _ := cmd.Flags().GetBool("replace")

		var envVars []ink.EnvVar
		var err error
		if file != "" {
			envVars, err = parseEnvFile(file)
			if err != nil {
				fatal(err.Error())
			}
		} else {
			envVars, err = parseEnvReader(os.Stdin)
			if err != nil {
				fatal(err.Error())
			}
		}

		if len(envVars) == 0 {
			fatal("No variables found in input")
		}

		err = client.SetSecrets(ctx(), ink.SetSecretsInput{
			Name:          name,
			WorkspaceSlug: cfg.Workspace,
			Project:       cfg.Project,
			EnvVars:       envVars,
			Replace:       replace,
		})
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(map[string]any{"imported": len(envVars), "service": name})
			return
		}

		fmt.Println()
		success(fmt.Sprintf("Imported %d variable(s) into %s", len(envVars), bold.Render(name)))
		fmt.Println()
	},
}

var secretsSetCmd = &cobra.Command{
	Use:   "set <service> KEY=VALUE [KEY=VALUE ...]",
	Short: "Set one or more env vars",
	Long:  "Sets variables and triggers a redeploy. Merges with existing vars.",
	Args:  rangeArgs(2, 100),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		client := newClient()
		replace, _ := cmd.Flags().GetBool("replace")

		var envVars []ink.EnvVar
		for _, arg := range args[1:] {
			k, v, ok := strings.Cut(arg, "=")
			if !ok {
				fatal(fmt.Sprintf("Invalid format %q — use KEY=VALUE", arg))
			}
			envVars = append(envVars, ink.EnvVar{Key: k, Value: v})
		}

		err := client.SetSecrets(ctx(), ink.SetSecretsInput{
			Name:          name,
			WorkspaceSlug: cfg.Workspace,
			Project:       cfg.Project,
			EnvVars:       envVars,
			Replace:       replace,
		})
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(map[string]any{"set": len(envVars), "service": name})
			return
		}

		fmt.Println()
		success(fmt.Sprintf("Set %d variable(s) on %s", len(envVars), bold.Render(name)))
		fmt.Println()
	},
}

var secretsListCmd = &cobra.Command{
	Use:   "list <service>",
	Short: "List env vars for a service",
	Args:  exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		svc := findService(name)
		if svc == nil {
			fatal(fmt.Sprintf("Service %q not found", name))
		}

		if jsonOutput {
			printJSON(svc.EnvVars)
			return
		}

		if len(svc.EnvVars) == 0 {
			fmt.Println(dim.Render("  No environment variables"))
			return
		}

		fmt.Println()
		for _, e := range svc.EnvVars {
			fmt.Printf("  %s=%s\n", bold.Render(e.Key), e.Value)
		}
		tableFooter(len(svc.EnvVars), "variable")
		fmt.Println()
	},
}

var secretsDeleteCmd = &cobra.Command{
	Use:   "delete <service> KEY [KEY ...]",
	Short: "Remove env vars from a service",
	Long:  "Removes variables and triggers a redeploy.",
	Args:  rangeArgs(2, 100),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		client := newClient()

		err := client.DeleteSecrets(ctx(), ink.DeleteSecretsInput{
			Name:          name,
			WorkspaceSlug: cfg.Workspace,
			Project:       cfg.Project,
			Keys:          args[1:],
		})
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(map[string]any{"removed": len(args) - 1, "service": name})
			return
		}

		fmt.Println()
		success(fmt.Sprintf("Removed %d variable(s) from %s", len(args)-1, bold.Render(name)))
		fmt.Println()
	},
}

var secretsUnsetCmd = &cobra.Command{
	Use:   "unset <service> <key>",
	Short: "Remove a single env var",
	Args:  exactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		key := args[1]
		client := newClient()

		err := client.DeleteSecrets(ctx(), ink.DeleteSecretsInput{
			Name:          name,
			WorkspaceSlug: cfg.Workspace,
			Project:       cfg.Project,
			Keys:          []string{key},
		})
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(map[string]any{"removed": key, "service": name})
			return
		}

		fmt.Println()
		success(fmt.Sprintf("Removed %s from %s", bold.Render(key), bold.Render(name)))
		fmt.Println()
	},
}

func parseEnvReader(r *os.File) ([]ink.EnvVar, error) {
	var vars []ink.EnvVar
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if len(v) >= 2 && ((v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'')) {
			v = v[1 : len(v)-1]
		}
		vars = append(vars, ink.EnvVar{Key: k, Value: v})
	}
	return vars, scanner.Err()
}
