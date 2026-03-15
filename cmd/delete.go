package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/mldotink/cli/internal/gql"
	"github.com/spf13/cobra"
)

var deleteYes bool

func init() {
	deleteCmd.Flags().BoolVarP(&deleteYes, "yes", "y", false, "Skip confirmation prompt")
}

var deleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Permanently delete a service and its deployment",
	Example: `ink delete myapi
ink delete myapi -y`,
	Args: exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		if !deleteYes && !jsonOutput {
			fmt.Printf("Delete service %s? This cannot be undone. [y/N] ", bold.Render(name))
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				if strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" {
					fmt.Println(dim.Render("  Cancelled"))
					return
				}
			}
		}

		client := newClient()

		result, err := gql.DeleteService(ctx(), client, name, projPtr(), nil, wsPtr())
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(result.ServiceDelete)
			return
		}

		success(result.ServiceDelete.Message)
	},
}
