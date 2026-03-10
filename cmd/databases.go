package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/mldotink/cli/internal/gql"
	"github.com/spf13/cobra"
)

func init() {
	databasesCmd.AddCommand(databasesCreateCmd)
	databasesCmd.AddCommand(databasesGetCmd)
	databasesCreateCmd.Flags().String("type", "sqlite", "Database type")
	databasesCreateCmd.Flags().String("size", "100mb", "Storage limit")
	databasesCreateCmd.Flags().String("region", "eu-central", "Region")
	databasesDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
	databasesCmd.AddCommand(databasesDeleteCmd)
}

var databasesCmd = &cobra.Command{
	GroupID: "manage",
	Use:     "databases",
	Aliases: []string{"db"},
	Short:   "Manage databases",
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()

		result, err := gql.ListResources(ctx(), client, wsPtr())
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
	Example: `# Create a SQLite database
ink db create mydb

# Use the returned credentials as env vars
ink deploy myapi --env DATABASE_URL=libsql://... --env DATABASE_AUTH_TOKEN=...`,
	Args: exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		client := newClient()

		input := gql.CreateResourceInput{
			Name:          name,
			WorkspaceSlug: wsPtr(),
		}
		if cmd.Flags().Changed("type") {
			v, _ := cmd.Flags().GetString("type")
			input.Type = ptr(v)
		}
		if cmd.Flags().Changed("size") {
			v, _ := cmd.Flags().GetString("size")
			input.Size = ptr(v)
		}
		if cmd.Flags().Changed("region") {
			v, _ := cmd.Flags().GetString("region")
			input.Region = ptr(v)
		}

		result, err := gql.CreateResource(ctx(), client, input)
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
		kv("URL", r.DatabaseUrl)
		kv("Token", r.AuthToken)
		kv("Region", r.Region)
		fmt.Println()
		fmt.Println(dim.Render("  Use these as env vars:"))
		fmt.Printf("  ink deploy -n myapp --env DATABASE_URL=%s --env DATABASE_AUTH_TOKEN=%s\n", r.DatabaseUrl, r.AuthToken)
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
		listResult, err := gql.GetResourceIDByName(ctx(), client, wsPtr())
		if err != nil {
			fatal(err.Error())
		}

		var resourceID string
		for _, r := range listResult.ResourceList.Nodes {
			if r.Name == name {
				resourceID = r.Id
				break
			}
		}
		if resourceID == "" {
			fatal(fmt.Sprintf("Database %q not found", name))
		}

		result, err := gql.GetResource(ctx(), client, resourceID)
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

		result, err := gql.DeleteResource(ctx(), client, name, wsPtr())
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
