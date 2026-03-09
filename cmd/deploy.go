package cmd

import (
	"fmt"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/mldotink/cli/internal/gql"
	"github.com/spf13/cobra"
)

func init() {
	f := deployCmd.Flags()
	f.StringP("repo", "r", "", "Repository name (default: same as service name)")
	f.IntP("port", "p", 0, "Application port (default: auto-detected)")
	f.String("host", "ink", "Git host: ink, github")
	f.String("branch", "main", "Git branch to deploy")
	f.String("region", "eu-central-1", "Deploy region")
	addServiceFlags(deployCmd)

	addServiceFlags(redeployCmd)

	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(redeployCmd)
}

// addServiceFlags registers the flags shared between deploy and redeploy.
func addServiceFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	if f.Lookup("memory") == nil {
		f.String("memory", "512Mi", "Memory limit: 256Mi, 512Mi, 1Gi, 2Gi, 4Gi, 8Gi")
	}
	if f.Lookup("vcpu") == nil {
		f.String("vcpu", "0.5", "CPU cores: 0.25, 0.5, 1, 2, 4")
	}
	if f.Lookup("env") == nil {
		f.StringArray("env", nil, "Environment variable as KEY=VALUE (repeatable)")
	}
	if f.Lookup("build-command") == nil {
		f.String("build-command", "", "Custom build command")
	}
	if f.Lookup("start-command") == nil {
		f.String("start-command", "", "Custom start command")
	}
	if f.Lookup("root-dir") == nil {
		f.String("root-dir", "", "Root directory for monorepo projects")
	}
	if f.Lookup("publish-dir") == nil {
		f.String("publish-dir", "", "Publish directory for static sites (e.g. dist, build)")
	}
	if f.Lookup("dockerfile") == nil {
		f.String("dockerfile", "", "Path to Dockerfile")
	}
	if f.Lookup("buildpack") == nil {
		f.String("buildpack", "railpack", "Build strategy: railpack, dockerfile, static")
	}
}

var deployCmd = &cobra.Command{
	Use:   "deploy <name> [flags]",
	Short: "Create or update a service",
	Long:  "Creates a new service or updates an existing one. Detects automatically.",
	Example: `# Deploy a Node.js app
ink deploy myapp

# Deploy with environment variables
ink deploy myapi --env DATABASE_URL=postgres://... --env SECRET_KEY=abc123

# Deploy with custom resources
ink deploy myapi --memory 2Gi --vcpu 1

# Deploy from a GitHub repo on a specific branch
ink deploy myapi --host github --repo myorg/myrepo --branch develop

# Deploy a static site
ink deploy docs --buildpack static --publish-dir dist

# Deploy with a Dockerfile
ink deploy myapi --buildpack dockerfile --dockerfile Dockerfile.prod

# Update memory on an existing service
ink deploy myapi --memory 4Gi`,
	Args: exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		client := newClient()

		existing := findService(name)

		if existing != nil {
			runUpdate(cmd, client, name)
		} else {
			runCreate(cmd, client, name)
		}
	},
}

var redeployCmd = &cobra.Command{
	Use:   "redeploy <name>",
	Short: "Redeploy a service (pull latest code and rebuild)",
	Long:  "Triggers a rebuild and redeploy. Optionally update configuration at the same time.",
	Example: `# Redeploy with latest code
ink redeploy myapi

# Redeploy and update memory
ink redeploy myapi --memory 2Gi

# Redeploy with new env vars
ink redeploy myapi --env DATABASE_URL=postgres://... --env SECRET_KEY=abc123

# Redeploy with different build settings
ink redeploy myapi --buildpack dockerfile --dockerfile Dockerfile.prod`,
	Args: exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		client := newClient()

		existing := findService(name)
		if existing == nil {
			fatal(fmt.Sprintf("Service %q not found", name))
		}

		runUpdate(cmd, client, name)
	},
}

func runCreate(cmd *cobra.Command, client graphql.Client, name string) {
	repo, _ := cmd.Flags().GetString("repo")
	if repo == "" {
		repo = name
	}

	input := gql.CreateServiceInput{
		Name:          name,
		Repo:          repo,
		WorkspaceSlug: wsPtr(),
		Project:       projPtr(),
	}

	if cmd.Flags().Changed("host") {
		v, _ := cmd.Flags().GetString("host")
		input.Host = ptr(v)
	}
	if cmd.Flags().Changed("branch") {
		v, _ := cmd.Flags().GetString("branch")
		input.Branch = ptr(v)
	}
	if cmd.Flags().Changed("memory") {
		v, _ := cmd.Flags().GetString("memory")
		input.Memory = ptr(v)
	}
	if cmd.Flags().Changed("vcpu") {
		v, _ := cmd.Flags().GetString("vcpu")
		input.Vcpus = ptr(v)
	}
	if cmd.Flags().Changed("build-command") {
		v, _ := cmd.Flags().GetString("build-command")
		input.BuildCommand = ptr(v)
	}
	if cmd.Flags().Changed("start-command") {
		v, _ := cmd.Flags().GetString("start-command")
		input.StartCommand = ptr(v)
	}
	if cmd.Flags().Changed("root-dir") {
		v, _ := cmd.Flags().GetString("root-dir")
		input.RootDirectory = ptr(v)
	}
	if cmd.Flags().Changed("publish-dir") {
		v, _ := cmd.Flags().GetString("publish-dir")
		input.PublishDirectory = ptr(v)
	}
	if cmd.Flags().Changed("dockerfile") {
		v, _ := cmd.Flags().GetString("dockerfile")
		input.DockerfilePath = ptr(v)
	}
	if cmd.Flags().Changed("buildpack") {
		v, _ := cmd.Flags().GetString("buildpack")
		input.BuildPack = ptr(v)
	}
	if cmd.Flags().Changed("port") {
		v, _ := cmd.Flags().GetInt("port")
		input.Port = &v
	}
	if cmd.Flags().Changed("region") {
		region, _ := cmd.Flags().GetString("region")
		input.Regions = []string{region}
	}

	if envs, _ := cmd.Flags().GetStringArray("env"); len(envs) > 0 {
		input.EnvVars = parseEnvVars(envs)
	}

	result, err := gql.CreateService(ctx(), client, input)
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

func runUpdate(cmd *cobra.Command, client graphql.Client, name string) {
	input := gql.UpdateServiceInput{
		Name:          name,
		WorkspaceSlug: wsPtr(),
		Project:       projPtr(),
	}

	if cmd.Flags().Changed("repo") {
		v, _ := cmd.Flags().GetString("repo")
		input.Repo = ptr(v)
	}
	if cmd.Flags().Changed("host") {
		v, _ := cmd.Flags().GetString("host")
		input.Host = ptr(v)
	}
	if cmd.Flags().Changed("branch") {
		v, _ := cmd.Flags().GetString("branch")
		input.Branch = ptr(v)
	}
	if cmd.Flags().Changed("memory") {
		v, _ := cmd.Flags().GetString("memory")
		input.Memory = ptr(v)
	}
	if cmd.Flags().Changed("vcpu") {
		v, _ := cmd.Flags().GetString("vcpu")
		input.Vcpus = ptr(v)
	}
	if cmd.Flags().Changed("build-command") {
		v, _ := cmd.Flags().GetString("build-command")
		input.BuildCommand = ptr(v)
	}
	if cmd.Flags().Changed("start-command") {
		v, _ := cmd.Flags().GetString("start-command")
		input.StartCommand = ptr(v)
	}
	if cmd.Flags().Changed("root-dir") {
		v, _ := cmd.Flags().GetString("root-dir")
		input.RootDirectory = ptr(v)
	}
	if cmd.Flags().Changed("publish-dir") {
		v, _ := cmd.Flags().GetString("publish-dir")
		input.PublishDirectory = ptr(v)
	}
	if cmd.Flags().Changed("dockerfile") {
		v, _ := cmd.Flags().GetString("dockerfile")
		input.DockerfilePath = ptr(v)
	}
	if cmd.Flags().Changed("buildpack") {
		v, _ := cmd.Flags().GetString("buildpack")
		input.BuildPack = ptr(v)
	}
	if cmd.Flags().Changed("port") {
		v, _ := cmd.Flags().GetInt("port")
		input.Port = &v
	}

	if envs, _ := cmd.Flags().GetStringArray("env"); len(envs) > 0 {
		input.EnvVars = parseEnvVars(envs)
	}

	result, err := gql.UpdateService(ctx(), client, input)
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

func parseEnvVars(envs []string) []gql.EnvVarInput {
	var vars []gql.EnvVarInput
	for _, e := range envs {
		if k, v, ok := strings.Cut(e, "="); ok {
			vars = append(vars, gql.EnvVarInput{Key: k, Value: v})
		}
	}
	return vars
}
