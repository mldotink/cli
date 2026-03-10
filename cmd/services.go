package cmd

import (
	"fmt"

	"github.com/Khan/genqlient/graphql"
	"github.com/mldotink/cli/internal/gql"
	"github.com/spf13/cobra"
)


var listCmd = &cobra.Command{
	GroupID: "core",
	Use:     "services [name]",
	Aliases: []string{"service"},
	Short:   "List services or show service details",
	Long:    "Lists services across all workspaces by default. Pass a service name to see details.",
	Example: `# List all services
ink services

# Show service details
ink services myapp
ink service myapp`,
	Args: maxArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// If a name is given, show service details
		if len(args) == 1 {
			showServiceDetail(args[0])
			return
		}

		client := newClient()

		// If workspace explicitly set, list only that workspace
		if ws := wsPtr(); ws != nil {
			listServicesForWorkspace(client, *ws)
			return
		}

		// Otherwise list across all workspaces
		wsResult, err := gql.ListWorkspaces(ctx(), client)
		if err != nil {
			fatal(err.Error())
		}

		type svcRow struct {
			name, workspace, project, status, url, memory, vcpus string
		}
		var allRows []svcRow

		for _, ws := range wsResult.WorkspaceList {
			result, err := gql.ListServices(ctx(), client, &ws.Slug)
			if err != nil {
				continue
			}
			projMap := make(map[string]string)
			for _, p := range result.ProjectList.Nodes {
				projMap[p.Id] = p.Slug
			}
			for _, s := range result.ServiceList.Nodes {
				url := dim.Render("—")
				if s.Fqdn != nil {
					url = *s.Fqdn
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

		// If all services are in the same workspace, hide the workspace column
		singleWS := true
		for _, r := range allRows[1:] {
			if r.workspace != allRows[0].workspace {
				singleWS = false
				break
			}
		}

		var rows [][]string
		if singleWS {
			for _, r := range allRows {
				rows = append(rows, []string{r.name, r.project, renderStatus(r.status), r.url, r.memory, r.vcpus})
			}
			fmt.Println()
			fmt.Println(styledTable([]string{"NAME", "PROJECT", "STATUS", "URL", "MEMORY", "vCPU"}, rows))
		} else {
			for _, r := range allRows {
				rows = append(rows, []string{r.name, r.workspace, r.project, renderStatus(r.status), r.url, r.memory, r.vcpus})
			}
			fmt.Println()
			fmt.Println(styledTable([]string{"NAME", "WORKSPACE", "PROJECT", "STATUS", "URL", "MEMORY", "vCPU"}, rows))
		}
		tableFooter(len(allRows), "service")
		fmt.Println()
	},
}

func listServicesForWorkspace(client graphql.Client, ws string) {
	result, err := gql.ListServices(ctx(), client, &ws)
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
		if s.Fqdn != nil {
			url = *s.Fqdn
		}
		rows = append(rows, []string{deref(s.Name, ""), projMap[s.ProjectId], renderStatus(s.Status), url, s.Memory, s.Vcpus})
	}

	fmt.Println()
	fmt.Println(styledTable([]string{"NAME", "PROJECT", "STATUS", "URL", "MEMORY", "vCPU"}, rows))
	tableFooter(len(nodes), "service")
	fmt.Println()
}

func showServiceDetail(name string) {
	svc := findService(name)
	if svc == nil {
		fatal(fmt.Sprintf("Service %q not found", name))
	}

	if jsonOutput {
		printJSON(svc)
		return
	}

	d := newDetail(deref(svc.Name, ""))
	d.kv("Status", renderStatus(svc.Status))
	if svc.ErrorMessage != nil {
		d.kv("Error", red.Render(*svc.ErrorMessage))
	}
	if svc.Fqdn != nil {
		d.kv("URL", accent.Render(*svc.Fqdn))
	}
	d.kv("Repo", svc.Repo)
	d.kv("Branch", svc.Branch)
	if svc.CommitHash != nil {
		hash := *svc.CommitHash
		if len(hash) > 12 {
			hash = hash[:12]
		}
		d.kv("Commit", dim.Render(hash))
	}
	d.kv("Memory", svc.Memory)
	d.kv("vCPU", svc.Vcpus)
	d.kv("Port", svc.Port)
	d.kv("Git host", svc.GitProvider)
	if svc.CustomDomain != nil {
		status := ""
		if svc.CustomDomainStatus != nil {
			status = " " + renderStatus(*svc.CustomDomainStatus)
		}
		d.kv("Domain", *svc.CustomDomain+status)
	}
	d.kv("Internal URL", svc.InternalUrl)
	if svc.Project != nil && svc.Project.Slug != "" {
		d.kv("Project", svc.Project.Slug)
	}

	// Timestamps
	if svc.CreatedAt != "" {
		d.kv("Created", dim.Render(svc.CreatedAt))
	}
	if svc.UpdatedAt != "" {
		d.kv("Updated", dim.Render(svc.UpdatedAt))
	}

	// Env var count hint
	if len(svc.EnvVars) > 0 {
		d.blank()
		d.line(dim.Render(fmt.Sprintf("  %d env var%s (use ink status %s -e to view)", len(svc.EnvVars), pluralS(len(svc.EnvVars)), name)))
	}

	fmt.Println()
	fmt.Println(d.String())
	fmt.Println()

	// Hint for more details
	fmt.Println(dim.Render(fmt.Sprintf("  Tip: ink status %s --deploy-logs 20 --runtime-logs 50 --metrics 1h", name)))
	fmt.Println()
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
