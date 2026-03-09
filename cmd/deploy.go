package cmd

import (
	"fmt"
	"strings"

	"github.com/mldotink/ink-cli/internal/api"
	"github.com/spf13/cobra"
)

func init() {
	f := deployCmd.Flags()
	f.StringP("repo", "r", "", "Repository name (defaults to service name)")
	f.IntP("port", "p", 0, "Application port")
	f.String("host", "", "Git host: ink or github")
	f.String("branch", "", "Git branch")
	f.String("memory", "", "Memory: 256Mi, 512Mi, 1Gi, 2Gi, 4Gi, 8Gi")
	f.String("vcpus", "", "vCPUs: 0.25, 0.5, 1, 2, 4")
	f.StringArray("env", nil, "Env var KEY=VALUE (repeatable)")
	f.String("build-command", "", "Custom build command")
	f.String("start-command", "", "Custom start command")
	f.String("root-dir", "", "Root directory (monorepo)")
	f.String("publish-dir", "", "Static publish directory (e.g. dist)")
	f.String("dockerfile", "", "Dockerfile path")
	f.String("buildpack", "", "Build strategy: railpack, dockerfile, static")
	f.String("region", "", "Deploy region (default: eu-central-1)")

	rootCmd.AddCommand(deployCmd)
}

var deployCmd = &cobra.Command{
	Use:   "deploy <name> [flags]",
	Short: "Create or update a service",
	Long:  "Creates a new service or updates an existing one. Detects automatically.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		client := newClient()

		existing, err := findService(client, name)
		if err != nil {
			fatal(err.Error())
		}

		if existing != nil {
			runUpdate(cmd, client, name)
		} else {
			runCreate(cmd, client, name)
		}
	},
}

func runCreate(cmd *cobra.Command, client *api.Client, name string) {
	repo, _ := cmd.Flags().GetString("repo")
	if repo == "" {
		repo = name
	}

	input := map[string]any{
		"name": name,
		"repo": repo,
	}
	addDefaults(input)
	addFlagInt(cmd, input, "port", "port")
	addFlagStr(cmd, input, "host", "host")
	addFlagStr(cmd, input, "branch", "branch")
	addFlagStr(cmd, input, "memory", "memory")
	addFlagStr(cmd, input, "vcpus", "vcpus")
	addFlagStr(cmd, input, "build-command", "buildCommand")
	addFlagStr(cmd, input, "start-command", "startCommand")
	addFlagStr(cmd, input, "root-dir", "rootDirectory")
	addFlagStr(cmd, input, "publish-dir", "publishDirectory")
	addFlagStr(cmd, input, "dockerfile", "dockerfilePath")
	addFlagStr(cmd, input, "buildpack", "buildPack")

	if cmd.Flags().Changed("region") {
		region, _ := cmd.Flags().GetString("region")
		input["regions"] = []string{region}
	}

	if envs, _ := cmd.Flags().GetStringArray("env"); len(envs) > 0 {
		input["envVars"] = parseEnvVars(envs)
	}

	var result struct {
		ServiceCreate struct {
			ServiceID   string `json:"serviceId"`
			Name        string `json:"name"`
			Status      string `json:"status"`
			InternalURL string `json:"internalUrl"`
		} `json:"serviceCreate"`
	}

	err := client.Do(`mutation($input: CreateServiceInput!) {
		serviceCreate(input: $input) { serviceId name status internalUrl }
	}`, map[string]any{"input": input}, &result)
	if err != nil {
		fatal(err.Error())
	}

	s := result.ServiceCreate
	if jsonOutput {
		printJSON(s)
		return
	}

	fmt.Println()
	success(fmt.Sprintf("Service created: %s", bold.Render(s.Name)))
	kv("Status", renderStatus(s.Status))
	kv("URL", accent.Render(fmt.Sprintf("https://%s.ml.ink", s.Name)))
	fmt.Println()
}

func runUpdate(cmd *cobra.Command, client *api.Client, name string) {
	input := map[string]any{
		"name": name,
	}
	addDefaults(input)
	addFlagStr(cmd, input, "repo", "repo")
	addFlagInt(cmd, input, "port", "port")
	addFlagStr(cmd, input, "host", "host")
	addFlagStr(cmd, input, "branch", "branch")
	addFlagStr(cmd, input, "memory", "memory")
	addFlagStr(cmd, input, "vcpus", "vcpus")
	addFlagStr(cmd, input, "build-command", "buildCommand")
	addFlagStr(cmd, input, "start-command", "startCommand")
	addFlagStr(cmd, input, "root-dir", "rootDirectory")
	addFlagStr(cmd, input, "publish-dir", "publishDirectory")
	addFlagStr(cmd, input, "dockerfile", "dockerfilePath")
	addFlagStr(cmd, input, "buildpack", "buildPack")

	if envs, _ := cmd.Flags().GetStringArray("env"); len(envs) > 0 {
		input["envVars"] = parseEnvVars(envs)
	}

	var result struct {
		ServiceUpdate struct {
			ServiceID string `json:"serviceId"`
			Name      string `json:"name"`
			Status    string `json:"status"`
		} `json:"serviceUpdate"`
	}

	err := client.Do(`mutation($input: UpdateServiceInput!) {
		serviceUpdate(input: $input) { serviceId name status }
	}`, map[string]any{"input": input}, &result)
	if err != nil {
		fatal(err.Error())
	}

	s := result.ServiceUpdate
	if jsonOutput {
		printJSON(s)
		return
	}

	fmt.Println()
	success(fmt.Sprintf("Service updated: %s", bold.Render(s.Name)))
	kv("Status", renderStatus(s.Status))
	fmt.Println()
}

func parseEnvVars(envs []string) []map[string]string {
	var vars []map[string]string
	for _, e := range envs {
		if k, v, ok := strings.Cut(e, "="); ok {
			vars = append(vars, map[string]string{"key": k, "value": v})
		}
	}
	return vars
}

func addFlagStr(cmd *cobra.Command, input map[string]any, flag, key string) {
	if cmd.Flags().Changed(flag) {
		val, _ := cmd.Flags().GetString(flag)
		input[key] = val
	}
}

func addFlagInt(cmd *cobra.Command, input map[string]any, flag, key string) {
	if cmd.Flags().Changed(flag) {
		val, _ := cmd.Flags().GetInt(flag)
		input[key] = val
	}
}
