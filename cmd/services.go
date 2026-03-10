package cmd

import (
	"fmt"

	"github.com/mldotink/cli/internal/gql"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "services",
	Short: "List services",
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()

		result, err := gql.ListServices(ctx(), client, wsPtr())
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
	},
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
