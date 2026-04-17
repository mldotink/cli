package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	ink "github.com/mldotink/sdk-go"
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

		result, err := client.DeleteService(ctx(), ink.DeleteServiceInput{
			Name:          name,
			Project:       cfg.Project,
			WorkspaceSlug: cfg.Workspace,
		})
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(result)
			return
		}

		success(result.Message)
	},
}
