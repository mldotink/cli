package cmd

import (
	"fmt"

	"github.com/mldotink/cli/internal/gql"
	"github.com/spf13/cobra"
)

func init() {
	logsCmd.Flags().Bool("deploy", false, "Show deploy logs instead of runtime logs")
	logsCmd.Flags().IntP("lines", "n", 100, "Number of lines to show (max 500)")
	logsCmd.Flags().String("query", "", "Filter logs by text query")
	logsCmd.Flags().String("since", "", "Filter logs from this time (RFC3339 or relative duration like 1h)")
	logsCmd.Flags().String("until", "", "Filter logs until this time (RFC3339 or relative duration like 30m)")
}

var logsCmd = &cobra.Command{
	Use:   "logs <name>",
	Short: "View runtime or deploy/build logs for a service",
	Example: `# View runtime logs (last 100 lines)
ink logs myapi

# View deploy/build logs
ink logs myapi --deploy

# Search runtime logs from the last hour
ink logs myapi --query timeout --since 1h

# View last 500 lines
ink logs myapi -n 500`,
	Args: exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		showBuild, _ := cmd.Flags().GetBool("deploy")
		lines, _ := cmd.Flags().GetInt("lines")
		lines = clampLogLines(lines)
		client := newClient()

		svc := findService(name)
		if svc == nil {
			fatal(fmt.Sprintf("Service %q not found", name))
		}

		logType := gql.LogTypeRuntime
		if showBuild {
			logType = gql.LogTypeBuild
		}

		filters, err := logFiltersFromCommand(cmd, "query")
		if err != nil {
			fatal(err.Error())
		}

		result, err := fetchServiceLogs(client, svc.Id, logType, lines, filters)
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(result)
			return
		}

		entries := result.Entries
		if len(entries) == 0 {
			fmt.Println(dim.Render("  No logs"))
			return
		}

		printLogEntries(entries, "")

		if result.HasMore {
			fmt.Println(dim.Render(fmt.Sprintf("\n  ... more available — use -n %d", lines+100)))
		}
	},
}
