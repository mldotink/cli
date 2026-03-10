package cmd

import (
	"fmt"

	"github.com/Khan/genqlient/graphql"
	"github.com/mldotink/cli/internal/gql"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "services",
	Short: "List services",
	Long:  "Lists services across all workspaces by default. Use -w to filter by workspace.",
	Run: func(cmd *cobra.Command, args []string) {
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
