package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var deleteYes bool

func init() {
	deleteCmd.Flags().BoolVarP(&deleteYes, "yes", "y", false, "Skip confirmation")
	rootCmd.AddCommand(deleteCmd)
}

var deleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a service",
	Args:  exactArgs(1),
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

		var result struct {
			ServiceDelete struct {
				ServiceID string `json:"serviceId"`
				Name      string `json:"name"`
				Message   string `json:"message"`
			} `json:"serviceDelete"`
		}

		err := client.Do(`mutation($name: String!, $ws: String, $proj: String) {
			serviceDelete(name: $name, workspaceSlug: $ws, project: $proj) { serviceId name message }
		}`, mergeVars(map[string]any{"name": name}), &result)
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
