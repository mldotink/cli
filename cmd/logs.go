package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	logsCmd.Flags().Bool("build", false, "Show build logs (default: runtime)")
	logsCmd.Flags().IntP("lines", "n", 100, "Number of log lines (max 500)")
	rootCmd.AddCommand(logsCmd)
}

var logsCmd = &cobra.Command{
	Use:   "logs <name>",
	Short: "View service logs",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		showBuild, _ := cmd.Flags().GetBool("build")
		lines, _ := cmd.Flags().GetInt("lines")
		if lines > 500 {
			lines = 500
		}
		client := newClient()

		svc, err := findService(client, name)
		if err != nil {
			fatal(err.Error())
		}
		if svc == nil {
			fatal(fmt.Sprintf("Service %q not found", name))
		}

		logType := "RUNTIME"
		if showBuild {
			logType = "BUILD"
		}

		var result struct {
			ServiceLogs struct {
				Entries []struct {
					Timestamp string  `json:"timestamp"`
					Level     *string `json:"level"`
					Message   string  `json:"message"`
				} `json:"entries"`
				HasMore bool `json:"hasMore"`
			} `json:"serviceLogs"`
		}

		err = client.Do(`query($input: LogsInput!) {
			serviceLogs(input: $input) {
				entries { timestamp level message }
				hasMore
			}
		}`, map[string]any{
			"input": map[string]any{
				"serviceId": svc.ID,
				"logType":   logType,
				"limit":     lines,
			},
		}, &result)
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
