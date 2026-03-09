package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	activityCmd.Flags().String("entity-type", "", "Filter by entity type (service, resource, domain, workspace, member, invite, repo)")
	activityCmd.Flags().String("entity-id", "", "Filter by entity ID (requires --entity-type)")
	activityCmd.Flags().IntP("limit", "n", 20, "Max entries (1-100)")
	rootCmd.AddCommand(activityCmd)
}

var activityCmd = &cobra.Command{
	Use:     "activity",
	Aliases: []string{"audit"},
	Short:   "View action log",
	Run: func(cmd *cobra.Command, args []string) {
		entityType, _ := cmd.Flags().GetString("entity-type")
		entityID, _ := cmd.Flags().GetString("entity-id")
		limit, _ := cmd.Flags().GetInt("limit")
		client := newClient()

		vars := defaultVars()
		vars["limit"] = limit
		if entityType != "" {
			vars["entityType"] = entityType
		}
		if entityID != "" {
			vars["entityId"] = entityID
		}

		var result struct {
			ActionLogList struct {
				Nodes []struct {
					ID         string `json:"id"`
					Action     string `json:"action"`
					EntityType string `json:"entityType"`
					EntityID   string `json:"entityId"`
					Source     string `json:"source"`
					CreatedAt  string `json:"createdAt"`
				} `json:"nodes"`
			} `json:"actionLogList"`
		}

		err := client.Do(`query($ws: String, $entityType: String, $entityId: String, $limit: Int) {
			actionLogList(workspaceSlug: $ws, entityType: $entityType, entityId: $entityId, limit: $limit) {
				nodes { id action entityType entityId source createdAt }
			}
		}`, vars, &result)
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(result.ActionLogList.Nodes)
			return
		}

		entries := result.ActionLogList.Nodes
		if len(entries) == 0 {
			fmt.Println(dim.Render("  No activity"))
			return
		}

		var rows [][]string
		for _, e := range entries {
			rows = append(rows, []string{
				dim.Render(e.CreatedAt),
				e.Action,
				e.EntityType,
				e.EntityID,
				dim.Render(e.Source),
			})
		}

		fmt.Println()
		fmt.Println(styledTable([]string{"TIME", "ACTION", "TYPE", "ENTITY", "SOURCE"}, rows))
		tableFooter(len(entries), "entry")
		fmt.Println()
	},
}
