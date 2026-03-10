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
	Use:     "secrets",
	Aliases: []string{"env"},
	Short:   "Manage service environment variables and secrets",
	Example: `# Import secrets from file
ink secrets import myapi --file .env

# Import from stdin
cat .env | ink secrets import myapi

# Set individual vars
ink secrets set myapi KEY1=value1 KEY2=value2

# List all vars
ink secrets list myapi

# Remove a single var
ink secrets unset myapi DATABASE_URL

# Remove multiple vars
ink secrets delete myapi KEY1 KEY2`,
}

var secretsImportCmd = &cobra.Command{
	Use:   "import <service>",
	Short: "Import env vars from file or stdin",
	Long:  "Reads KEY=VALUE pairs and merges them with existing vars. Triggers a redeploy.",
	Example: `# From file
ink secrets import myapi --file .env

# From stdin
cat .env | ink secrets import myapi

# Replace all vars (removes unspecified)
ink secrets import myapi --file .env --replace`,
	Args: exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		client := newClient()

		file, _ := cmd.Flags().GetString("file")
		replace, _ := cmd.Flags().GetBool("replace")

		var envVars []gql.EnvVarInput
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

		result, err := gql.SetSecrets(ctx(), client, gql.SetSecretsInput{
			Name:          name,
			WorkspaceSlug: wsPtr(),
			Project:       projPtr(),
			EnvVars:       envVars,
			Replace:       &replace,
		})
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(result.ServiceSetSecrets)
			return
		}

		fmt.Println()
		success(fmt.Sprintf("Imported %d variable(s) into %s", len(envVars), bold.Render(name)))
		kv("Status", renderStatus(result.ServiceSetSecrets.Status))
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

		var envVars []gql.EnvVarInput
		for _, arg := range args[1:] {
			k, v, ok := strings.Cut(arg, "=")
			if !ok {
				fatal(fmt.Sprintf("Invalid format %q — use KEY=VALUE", arg))
			}
			envVars = append(envVars, gql.EnvVarInput{Key: k, Value: v})
		}

		result, err := gql.SetSecrets(ctx(), client, gql.SetSecretsInput{
			Name:          name,
			WorkspaceSlug: wsPtr(),
			Project:       projPtr(),
			EnvVars:       envVars,
			Replace:       &replace,
		})
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(result.ServiceSetSecrets)
			return
		}

		fmt.Println()
		success(fmt.Sprintf("Set %d variable(s) on %s", len(envVars), bold.Render(name)))
		kv("Status", renderStatus(result.ServiceSetSecrets.Status))
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

		result, err := gql.DeleteSecrets(ctx(), client, gql.DeleteSecretsInput{
			Name:          name,
			WorkspaceSlug: wsPtr(),
			Project:       projPtr(),
			Keys:          args[1:],
		})
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(result.ServiceDeleteSecrets)
			return
		}

		fmt.Println()
		success(fmt.Sprintf("Removed %d variable(s) from %s", len(args)-1, bold.Render(name)))
		kv("Status", renderStatus(result.ServiceDeleteSecrets.Status))
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

		result, err := gql.UnsetSecret(ctx(), client, name, key, projPtr(), wsPtr())
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(result.ServiceUnsetSecret)
			return
		}

		fmt.Println()
		success(fmt.Sprintf("Removed %s from %s", bold.Render(key), bold.Render(name)))
		kv("Status", renderStatus(result.ServiceUnsetSecret.Status))
		fmt.Println()
	},
}

func parseEnvReader(r *os.File) ([]gql.EnvVarInput, error) {
	var vars []gql.EnvVarInput
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
		vars = append(vars, gql.EnvVarInput{Key: k, Value: v})
	}
	return vars, scanner.Err()
}
