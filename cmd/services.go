package cmd

import (
	"fmt"

	"github.com/Khan/genqlient/graphql"
	"github.com/mldotink/cli/internal/gql"
	"github.com/spf13/cobra"
)

func init() {
	listCmd.Flags().Bool("all", false, "List services across all workspaces and projects")
	listCmd.Flags().BoolP("env", "e", false, "Show environment variables when inspecting one service")
	listCmd.Flags().BoolP("template", "t", false, "Show template outputs (credentials, connection info) if deployed from a template")
	listCmd.Flags().Int("deploy-logs", 0, "Include N deploy log lines when inspecting one service (max 500)")
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
			listServicesForWorkspace(client, ws, projPtr())
		}
	},
}

func listAllServices(client graphql.Client) {
	wsResult, err := gql.ListWorkspaces(ctx(), client)
	if err != nil {
		fatal(err.Error())
	}

	type svcRow struct {
		name, workspace, project, status, url, memory, vcpus string
	}
	var allRows []svcRow

	for _, ws := range wsResult.WorkspaceList {
		result, err := gql.ListServices(ctx(), client, &ws.Slug, nil)
		if err != nil {
			continue
		}
		projMap := make(map[string]string)
		for _, p := range result.ProjectList.Nodes {
			projMap[p.Id] = p.Slug
		}
		for _, s := range result.ServiceList.Nodes {
			url := dim.Render("—")
			if endpoint := preferredServiceEndpoint(listServicePorts(s.Ports), s.CustomDomain); endpoint != "" {
				url = endpoint
			}
			allRows = append(allRows, svcRow{
				name:      deref(s.Name, ""),
				workspace: ws.Slug,
				project:   projMap[s.ProjectId],
				status:    s.Status,
				url:       url,
				memory:    s.Memory,
				vcpus:     s.Vcpus,
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

func listServicesForWorkspace(client graphql.Client, ws string, proj *string) {
	result, err := gql.ListServices(ctx(), client, &ws, proj)
	if err != nil {
		fatal(err.Error())
	}

	if jsonOutput {
		printJSON(result.ServiceList)
		return
	}

	nodes := result.ServiceList.Nodes
	if len(nodes) == 0 {
		fmt.Println(dim.Render("  No services"))
		return
	}

	projMap := make(map[string]string)
	for _, p := range result.ProjectList.Nodes {
		projMap[p.Id] = p.Slug
	}

	var rows [][]string
	for _, s := range nodes {
		url := dim.Render("—")
		if endpoint := preferredServiceEndpoint(listServicePorts(s.Ports), s.CustomDomain); endpoint != "" {
			url = endpoint
		}
		rows = append(rows, []string{deref(s.Name, ""), projMap[s.ProjectId], renderStatus(s.Status), url, s.Memory, s.Vcpus})
	}

	fmt.Println()
	fmt.Println(styledTable([]string{"NAME", "PROJECT", "STATUS", "URL", "MEMORY", "vCPU"}, rows))
	tableFooter(len(nodes), "service")
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

func findService(name string) *gql.FindServiceServiceListServiceConnectionNodesService {
	client := newClient()
	result, err := gql.FindService(ctx(), client, wsPtr())
	if err != nil {
		fatal(err.Error())
	}

	for i := range result.ServiceList.Nodes {
		if deref(result.ServiceList.Nodes[i].Name, "") == name {
			return &result.ServiceList.Nodes[i]
		}
	}
	return nil
}
