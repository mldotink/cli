package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/mldotink/cli/internal/gql"
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
		result, err := gql.ServiceExec(ctx(), client, &name, nil, command, projPtr(), wsPtr())
		if err != nil {
			fatal(err.Error())
		}

		e := result.ServiceExec
		if jsonOutput {
			printJSON(e)
			os.Exit(e.ExitCode)
		}

		if e.Stdout != "" {
			fmt.Fprint(os.Stdout, e.Stdout)
		}
		if e.Stderr != "" {
			fmt.Fprint(os.Stderr, e.Stderr)
		}
		os.Exit(e.ExitCode)
	},
}
