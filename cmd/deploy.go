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
	f := deployCmd.Flags()
	f.StringP("repo", "r", "", "Repository name (default: same as service name)")
	f.String("image", "", "Docker image to deploy (e.g. nginx:latest)")
	f.IntP("port", "p", 0, "Application port (default: auto-detected)")
	f.String("host", "ink", "Git host: ink, github")
	f.String("branch", "main", "Git branch to deploy")
	f.String("region", "eu-central-1", "Deploy region")
	addServiceFlags(deployCmd)

	redeployCmd.Flags().String("image", "", "Docker image to deploy (e.g. nginx:latest)")
	addServiceFlags(redeployCmd)

}

func addServiceFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	if f.Lookup("memory") == nil {
		f.String("memory", "256Mi", "Memory limit: 128Mi, 256Mi, 512Mi, 1024Mi, 2048Mi, 4096Mi")
	}
	if f.Lookup("vcpu") == nil {
		f.String("vcpu", "0.25", "CPU cores: 0.1, 0.2, 0.25, 0.3, 0.4, 0.5, 1, 2, 3, 4")
	}
	if f.Lookup("env") == nil {
		f.StringArray("env", nil, "Environment variable as KEY=VALUE (repeatable)")
	}
	if f.Lookup("env-file") == nil {
		f.StringArray("env-file", nil, "Read env vars from file (repeatable, e.g. .env)")
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
	if f.Lookup("destroy-timeout-seconds") == nil {
		f.Int("destroy-timeout-seconds", 0, "Auto-destroy after N seconds from deploy completion (0=persistent, max 86400)")
	}
}

var deployCmd = &cobra.Command{
	Use:   "deploy <name> [flags]",
	Short: "Deploy a service for the first time",
	Long: `Creates a new service from a git repo or Docker image. For git repos, the repo
must exist first — create one with 'ink repo create' (Ink-managed) or use a GitHub
repo with the GitHub App installed. The service will be live at {name}.ml.ink.

To update or redeploy an existing service, use 'ink redeploy'.`,
	Example: `# Ink-managed repo
ink repo create myrepo
git remote add ink <remote-url-from-output>
git push ink main
ink deploy myapi --repo myrepo

# GitHub repo (requires GitHub App installed)
ink deploy myapi --host github --repo myorg/myrepo

# Docker image (skips build)
ink deploy mynginx --image nginx:latest --port 80

# With options
ink deploy myapi --repo myrepo --memory 512Mi --vcpu 0.5 --env-file .env
ink deploy docs --repo myrepo --buildpack static --publish-dir dist`,
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
	Short: "Rebuild and redeploy a service, optionally updating its config",
	Long: `Pulls the latest code from the repo, rebuilds, and redeploys. Use this to pick
up new commits or to change configuration (memory, CPU, env vars, buildpack).`,
	Example: `# Redeploy with latest code
ink redeploy myapi

# Update memory and CPU
ink redeploy myapi --memory 2Gi --vcpu 1

# Update env vars
ink redeploy myapi --env-file .env
ink redeploy myapi --env NODE_ENV=production

# Switch to Dockerfile build
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

func runCreate(cmd *cobra.Command, client *ink.Client, name string) {
	image, _ := cmd.Flags().GetString("image")
	isImage := image != ""

	input := ink.CreateServiceInput{
		Name:          name,
		WorkspaceSlug: cfg.Workspace,
		Project:       cfg.Project,
	}

	if isImage {
		input.Source = "image"
		input.Image = image
	} else {
		repo, _ := cmd.Flags().GetString("repo")
		if repo == "" {
			repo = name
		}
		input.Source = "repo"
		input.Repo = repo
	}

	if cmd.Flags().Changed("host") && !isImage {
		v, _ := cmd.Flags().GetString("host")
		input.Host = v
	}
	if cmd.Flags().Changed("branch") && !isImage {
		v, _ := cmd.Flags().GetString("branch")
		input.Branch = v
	}
	if cmd.Flags().Changed("memory") {
		v, _ := cmd.Flags().GetString("memory")
		input.Memory = v
	}
	if cmd.Flags().Changed("vcpu") {
		v, _ := cmd.Flags().GetString("vcpu")
		input.VCPUs = v
	}
	if cmd.Flags().Changed("build-command") {
		v, _ := cmd.Flags().GetString("build-command")
		input.BuildCommand = v
	}
	if cmd.Flags().Changed("start-command") {
		v, _ := cmd.Flags().GetString("start-command")
		input.StartCommand = v
	}
	if cmd.Flags().Changed("root-dir") {
		v, _ := cmd.Flags().GetString("root-dir")
		input.RootDirectory = v
	}
	if cmd.Flags().Changed("publish-dir") {
		v, _ := cmd.Flags().GetString("publish-dir")
		input.PublishDirectory = v
	}
	if cmd.Flags().Changed("dockerfile") {
		v, _ := cmd.Flags().GetString("dockerfile")
		input.DockerfilePath = v
	}
	if cmd.Flags().Changed("buildpack") {
		v, _ := cmd.Flags().GetString("buildpack")
		input.BuildPack = v
	}
	if cmd.Flags().Changed("port") {
		v, _ := cmd.Flags().GetInt("port")
		input.Ports = singlePublicHTTPPort(v)
	}
	if cmd.Flags().Changed("region") {
		region, _ := cmd.Flags().GetString("region")
		input.Regions = []string{region}
	}
	if cmd.Flags().Changed("destroy-timeout-seconds") {
		v, _ := cmd.Flags().GetInt("destroy-timeout-seconds")
		input.DestroyTimeoutSeconds = v
	}

	input.EnvVars = collectEnvVars(cmd)

	result, err := client.CreateService(ctx(), input)
	if err != nil {
		fatal(err.Error())
	}

	if jsonOutput {
		printJSON(result)
		return
	}

	fmt.Println()
	success(fmt.Sprintf("Service created: %s", bold.Render(result.Name)))
	kv("Status", renderStatus(result.Status))
	if endpoint := preferredServiceEndpoint(inkServicePorts(result.Ports), ""); endpoint != "" {
		kv("Endpoint", accent.Render(endpoint))
	}
	fmt.Println()
}

func runUpdate(cmd *cobra.Command, client *ink.Client, name string) {
	input := ink.UpdateServiceInput{
		Name:          name,
		WorkspaceSlug: cfg.Workspace,
		Project:       cfg.Project,
	}

	if cmd.Flags().Changed("image") {
		v, _ := cmd.Flags().GetString("image")
		input.Source = ptr("image")
		input.Image = ptr(v)
	}
	if cmd.Flags().Changed("repo") {
		v, _ := cmd.Flags().GetString("repo")
		input.Source = ptr("repo")
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
		input.VCPUs = ptr(v)
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
		input.Ports = singlePublicHTTPPort(v)
	}
	if cmd.Flags().Changed("destroy-timeout-seconds") {
		v, _ := cmd.Flags().GetInt("destroy-timeout-seconds")
		input.DestroyTimeoutSeconds = &v
	}

	input.EnvVars = collectEnvVars(cmd)

	result, err := client.UpdateService(ctx(), input)
	if err != nil {
		fatal(err.Error())
	}

	if jsonOutput {
		printJSON(result)
		return
	}

	fmt.Println()
	success(fmt.Sprintf("Service updated: %s", bold.Render(result.Name)))
	kv("Status", renderStatus(result.Status))
	fmt.Println()
}

func collectEnvVars(cmd *cobra.Command) []ink.EnvVar {
	vars := make(map[string]string)

	if files, _ := cmd.Flags().GetStringArray("env-file"); len(files) > 0 {
		for _, f := range files {
			parsed, err := parseEnvFile(f)
			if err != nil {
				fatal(err.Error())
			}
			for _, v := range parsed {
				vars[v.Key] = v.Value
			}
		}
	}

	if envs, _ := cmd.Flags().GetStringArray("env"); len(envs) > 0 {
		for _, e := range envs {
			if k, v, ok := strings.Cut(e, "="); ok {
				vars[k] = v
			}
		}
	}

	if len(vars) == 0 {
		return nil
	}

	result := make([]ink.EnvVar, 0, len(vars))
	for k, v := range vars {
		result = append(result, ink.EnvVar{Key: k, Value: v})
	}
	return result
}

func parseEnvFile(path string) ([]ink.EnvVar, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read env file %s: %w", path, err)
	}
	defer f.Close()

	var vars []ink.EnvVar
	scanner := bufio.NewScanner(f)
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
