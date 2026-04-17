package cmd

import (
	"fmt"

	ink "github.com/mldotink/sdk-go"
	"github.com/spf13/cobra"
)

func init() {
	listCmd.Flags().Bool("all", false, "List services across all workspaces and projects")
	listCmd.Flags().BoolP("env", "e", false, "Show environment variables when inspecting one service")
	listCmd.Flags().BoolP("template", "t", false, "Show template outputs (credentials, connection info) if deployed from a template")
	listCmd.Flags().Int("build-logs", 0, "Include N build log lines when inspecting one service (max 500)")
	listCmd.Flags().Int("deploy-logs", 0, "Alias for --build-logs")
	listCmd.Flags().MarkHidden("deploy-logs")
	listCmd.Flags().Int("runtime-logs", 0, "Include N runtime log lines when inspecting one service (max 500)")
	listCmd.Flags().String("metrics", "", "Include CPU/memory/network metrics when inspecting one service: 1h, 6h, 24h, 7d, 30d")
	listCmd.Flags().String("log-query", "", "Filter included logs by text query when inspecting one service")
	listCmd.Flags().String("since", "", "Filter included logs from this time (RFC3339 or relative duration like 1h)")
	listCmd.Flags().String("until", "", "Filter included logs until this time (RFC3339 or relative duration like 30m)")
}

var listCmd = &cobra.Command{
	Use:     "service [name]",
	Aliases: []string{"services"},
	Short:   "List all deployed services or show details for one",
	Long: `Lists deployed services in the current workspace and project (from config).
Use --all to list across all workspaces and projects regardless of config.
Pass a service name to see full details including repo, branch, resources, URLs,
and optional logs/metrics.`,
	Example: `# List services in configured workspace/project
ink service

# List across all workspaces and projects
ink service --all

# Show service details
ink service myapp

# Include logs and metrics in the service detail view
ink service myapp --runtime-logs 50 --metrics 1h

# Include metrics for the last 24 hours
ink service myapp --metrics 24h

# Include build logs
ink service myapp --build-logs 50

# Filter included runtime logs
ink service myapp --runtime-logs 100 --log-query timeout --since 1h`,
	Args: maxArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		opts, err := inspectOptionsFromCommand(cmd)
		if err != nil {
			fatal(err.Error())
		}
		if err := validateInspectOptions(opts); err != nil {
			fatal(err.Error())
		}

		if len(args) == 1 {
			showServiceDetail(args[0], opts)
			return
		}

		if hasInspectFlags(opts) {
			fatal("Detail flags require a service name: ink service <name> --metrics 1h")
		}

		all, _ := cmd.Flags().GetBool("all")
		client := newClient()

		if all {
			listAllServices(client)
		} else {
			ws := cfg.Workspace
			if ws == "" {
				ws = "default"
			}
			listServicesForWorkspace(client, ws, cfg.Project)
		}
	},
}

func listAllServices(client *ink.Client) {
	workspaces, err := client.ListWorkspaces(ctx())
	if err != nil {
		fatal(err.Error())
	}

	type svcRow struct {
		name, workspace, project, status, url, memory, vcpus string
	}
	var allRows []svcRow

	for _, ws := range workspaces {
		services, err := client.ListServices(ctx(), ws.Slug, "")
		if err != nil {
			continue
		}
		projects, _ := client.ListProjects(ctx(), ws.Slug)
		projMap := make(map[string]string)
		for _, p := range projects {
			projMap[p.ID] = p.Slug
		}
		for _, s := range services {
			url := dim.Render("—")
			if endpoint := preferredServiceEndpoint(inkServicePorts(s.Ports), s.CustomDomain); endpoint != "" {
				url = endpoint
			}
			allRows = append(allRows, svcRow{
				name:      s.Name,
				workspace: ws.Slug,
				project:   projMap[s.ProjectID],
				status:    s.Status,
				url:       url,
				memory:    s.Memory,
				vcpus:     s.VCPUs,
			})
		}
	}

	if jsonOutput {
		printJSON(allRows)
		return
	}

	if len(allRows) == 0 {
		fmt.Println(dim.Render("  No services"))
		return
	}

	var rows [][]string
	for _, r := range allRows {
		rows = append(rows, []string{r.name, r.workspace, r.project, renderStatus(r.status), r.url, r.memory, r.vcpus})
	}

	fmt.Println()
	fmt.Println(styledTable([]string{"NAME", "WORKSPACE", "PROJECT", "STATUS", "URL", "MEMORY", "vCPU"}, rows))
	tableFooter(len(allRows), "service")
	serviceHints()
}

func listServicesForWorkspace(client *ink.Client, ws, proj string) {
	services, err := client.ListServices(ctx(), ws, proj)
	if err != nil {
		fatal(err.Error())
	}

	projects, _ := client.ListProjects(ctx(), ws)
	projMap := make(map[string]string)
	for _, p := range projects {
		projMap[p.ID] = p.Slug
	}

	if jsonOutput {
		printJSON(services)
		return
	}

	if len(services) == 0 {
		fmt.Println(dim.Render("  No services"))
		return
	}

	var rows [][]string
	for _, s := range services {
		url := dim.Render("—")
		if endpoint := preferredServiceEndpoint(inkServicePorts(s.Ports), s.CustomDomain); endpoint != "" {
			url = endpoint
		}
		rows = append(rows, []string{s.Name, projMap[s.ProjectID], renderStatus(s.Status), url, s.Memory, s.VCPUs})
	}

	fmt.Println()
	fmt.Println(styledTable([]string{"NAME", "PROJECT", "STATUS", "URL", "MEMORY", "vCPU"}, rows))
	tableFooter(len(services), "service")
	serviceHints()
}

func serviceHints() {
	fmt.Println()
	fmt.Println(dim.Render("  ink service <name>       Show service details"))
	fmt.Println(dim.Render("  ink deploy <name>        Deploy a new service"))
	fmt.Println(dim.Render("  ink redeploy <name>      Rebuild and update a service"))
	fmt.Println(dim.Render("  ink service --all        List across all workspaces"))
	fmt.Println()
}

func showServiceDetail(name string, opts serviceInspectOptions) {
	inspectService(name, opts, true)
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func findService(name string) *ink.Service {
	client := newClient()
	services, err := client.ListServices(ctx(), cfg.Workspace, cfg.Project)
	if err != nil {
		fatal(err.Error())
	}

	for i := range services {
		if services[i].Name == name {
			return &services[i]
		}
	}
	return nil
}
