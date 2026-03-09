package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	databasesCmd.AddCommand(databasesCreateCmd)
	databasesCmd.AddCommand(databasesGetCmd)
	databasesCreateCmd.Flags().String("type", "", "Database type (default: sqlite)")
	databasesCreateCmd.Flags().String("size", "", "Size limit (default: 100mb)")
	databasesCreateCmd.Flags().String("region", "", "Region (default: eu-central)")
	databasesDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
	databasesCmd.AddCommand(databasesDeleteCmd)
	rootCmd.AddCommand(databasesCmd)
}

var databasesCmd = &cobra.Command{
	Use:     "databases",
	Aliases: []string{"db"},
	Short:   "Manage databases",
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()

		var result struct {
			ResourceList struct {
				Nodes []struct {
					ID     string `json:"id"`
					Name   string `json:"name"`
					Type   string `json:"type"`
					Status string `json:"status"`
					Region string `json:"region"`
				} `json:"nodes"`
				TotalCount int `json:"totalCount"`
			} `json:"resourceList"`
		}

		err := client.Do(`query($ws: String) {
			resourceList(workspaceSlug: $ws) {
				nodes { id name type status region }
				totalCount
			}
		}`, defaultVars(), &result)
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(result.ResourceList)
			return
		}

		nodes := result.ResourceList.Nodes
		if len(nodes) == 0 {
			fmt.Println(dim.Render("  No databases"))
			return
		}

		var rows [][]string
		for _, r := range nodes {
			rows = append(rows, []string{r.Name, r.Type, renderStatus(r.Status), r.Region})
		}

		fmt.Println()
		fmt.Println(styledTable([]string{"NAME", "TYPE", "STATUS", "REGION"}, rows))
		tableFooter(len(nodes), "database")
		fmt.Println()
	},
}

var databasesCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a database",
	Args:  exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		client := newClient()

		input := map[string]any{"name": name}
		addDefaults(input)
		addFlagStr(cmd, input, "type", "type")
		addFlagStr(cmd, input, "size", "size")
		addFlagStr(cmd, input, "region", "region")

		var result struct {
			ResourceCreate struct {
				ResourceID  string `json:"resourceId"`
				Name        string `json:"name"`
				Type        string `json:"type"`
				Region      string `json:"region"`
				DatabaseURL string `json:"databaseUrl"`
				AuthToken   string `json:"authToken"`
				Status      string `json:"status"`
			} `json:"resourceCreate"`
		}

		err := client.Do(`mutation($input: CreateResourceInput!) {
			resourceCreate(input: $input) {
				resourceId name type region databaseUrl authToken status
			}
		}`, map[string]any{"input": input}, &result)
		if err != nil {
			fatal(err.Error())
		}

		r := result.ResourceCreate
		if jsonOutput {
			printJSON(r)
			return
		}

		fmt.Println()
		success(fmt.Sprintf("Database created: %s", bold.Render(r.Name)))
		kv("URL", r.DatabaseURL)
		kv("Token", r.AuthToken)
		kv("Region", r.Region)
		fmt.Println()
		fmt.Println(dim.Render("  Use these as env vars:"))
		fmt.Printf("  ink deploy -n myapp --env DATABASE_URL=%s --env DATABASE_AUTH_TOKEN=%s\n", r.DatabaseURL, r.AuthToken)
		fmt.Println()
	},
}

var databasesGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Get database details and credentials",
	Args:  exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		client := newClient()

		// resourceGet uses ID, so we list and find by name
		var listResult struct {
			ResourceList struct {
				Nodes []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"nodes"`
			} `json:"resourceList"`
		}

		err := client.Do(`query($ws: String) {
			resourceList(workspaceSlug: $ws) { nodes { id name } }
		}`, defaultVars(), &listResult)
		if err != nil {
			fatal(err.Error())
		}

		var resourceID string
		for _, r := range listResult.ResourceList.Nodes {
			if r.Name == name {
				resourceID = r.ID
				break
			}
		}
		if resourceID == "" {
			fatal(fmt.Sprintf("Database %q not found", name))
		}

		var result struct {
			ResourceGet *struct {
				ID       string `json:"id"`
				Name     string `json:"name"`
				Type     string `json:"type"`
				Region   string `json:"region"`
				Status   string `json:"status"`
				Metadata *struct {
					Size     *string `json:"size"`
					Hostname *string `json:"hostname"`
				} `json:"metadata"`
			} `json:"resourceGet"`
		}

		err = client.Do(`query($id: ID!) {
			resourceGet(id: $id) { id name type region status metadata { size hostname } }
		}`, map[string]any{"id": resourceID}, &result)
		if err != nil {
			fatal(err.Error())
		}

		if result.ResourceGet == nil {
			fatal(fmt.Sprintf("Database %q not found", name))
		}

		r := result.ResourceGet
		if jsonOutput {
			printJSON(r)
			return
		}

		d := newDetail(r.Name)
		d.kv("Type", r.Type)
		d.kv("Status", renderStatus(r.Status))
		d.kv("Region", r.Region)
		if r.Metadata != nil && r.Metadata.Hostname != nil {
			d.kv("Host", *r.Metadata.Hostname)
		}

		fmt.Println()
		fmt.Println(d.String())
		fmt.Println()
	},
}

var databasesDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a database",
	Args:  exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		yes, _ := cmd.Flags().GetBool("yes")

		if !yes && !jsonOutput {
			fmt.Printf("Delete database %s? This cannot be undone. [y/N] ", bold.Render(name))
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				if strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" {
					fmt.Println(dim.Render("  Cancelled"))
					return
				}
			}
		}

		client := newClient()

		var result struct {
			ResourceDelete struct {
				ResourceID string `json:"resourceId"`
				Name       string `json:"name"`
				Message    string `json:"message"`
			} `json:"resourceDelete"`
		}

		err := client.Do(`mutation($name: String!, $ws: String) {
			resourceDelete(name: $name, workspaceSlug: $ws) { resourceId name message }
		}`, mergeVars(map[string]any{"name": name}), &result)
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(result.ResourceDelete)
			return
		}

		success(result.ResourceDelete.Message)
	},
}
