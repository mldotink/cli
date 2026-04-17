package cmd

import (
	"fmt"
	"os"
	"strings"

	ink "github.com/mldotink/sdk-go"
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec <service> -- <command...>",
	Short: "Run a command in a running service container",
	Example: `ink exec myapi -- ls -la /app
ink exec myapi -- cat /var/log/app.log
ink exec myapi --project backend -- env`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		command := strings.Join(args[1:], " ")

		client := newClient()
		result, err := client.Exec(ctx(), ink.ExecInput{
			Name:          name,
			Project:       cfg.Project,
			WorkspaceSlug: cfg.Workspace,
		}, command)
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(result)
			os.Exit(result.ExitCode)
		}

		if result.Stdout != "" {
			fmt.Fprint(os.Stdout, result.Stdout)
		}
		if result.Stderr != "" {
			fmt.Fprint(os.Stderr, result.Stderr)
		}
		os.Exit(result.ExitCode)
	},
}
