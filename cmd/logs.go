package cmd

import (
	"fmt"

	"github.com/mldotink/cli/internal/gql"
	"github.com/spf13/cobra"
)

func init() {
	logsCmd.Flags().Bool("deploy", false, "Show deploy logs instead of runtime logs")
	logsCmd.Flags().IntP("lines", "n", 100, "Number of lines to show (max 500)")
}

var logsCmd = &cobra.Command{
	Use:   "logs <name>",
	Short: "View runtime or deploy/build logs for a service",
	Example: `# View runtime logs (last 100 lines)
ink logs myapi

# View deploy/build logs
ink logs myapi --deploy

# View last 500 lines
ink logs myapi -n 500`,
	Args: exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		showBuild, _ := cmd.Flags().GetBool("deploy")
		lines, _ := cmd.Flags().GetInt("lines")
		if lines > 500 {
			lines = 500
		}
		client := newClient()

		svc := findService(name)
		if svc == nil {
			fatal(fmt.Sprintf("Service %q not found", name))
		}

		logType := gql.LogTypeRuntime
		if showBuild {
			logType = gql.LogTypeBuild
		}

		result, err := gql.ServiceLogs(ctx(), client, gql.LogsInput{
			ServiceId: svc.Id,
			LogType:   logType,
			Limit:     &lines,
		})
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(result.ServiceLogs)
			return
		}

		entries := result.ServiceLogs.Entries
		if len(entries) == 0 {
			fmt.Println(dim.Render("  No logs"))
			return
		}

		for _, e := range entries {
			ts := dim.Render(e.Timestamp)
			level := ""
			if e.Level != nil {
				switch *e.Level {
				case "error", "ERROR":
					level = red.Render("[ERR] ")
				case "warn", "WARN":
					level = yellow.Render("[WRN] ")
				}
			}
			fmt.Printf("%s %s%s\n", ts, level, e.Message)
		}

		if result.ServiceLogs.HasMore {
			fmt.Println(dim.Render(fmt.Sprintf("\n  ... more available — use -n %d", lines+100)))
		}
	},
}
