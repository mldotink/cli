package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	statusCmd.Flags().BoolP("env", "e", false, "Show environment variables")
	statusCmd.Flags().Int("deploy-logs", 0, "Include N deploy log lines (max 500)")
	statusCmd.Flags().Int("runtime-logs", 0, "Include N runtime log lines (max 500)")
	statusCmd.Flags().String("metrics", "", "Include CPU/memory/network metrics: 1h, 6h, 24h, 7d, 30d")
	statusCmd.Flags().String("log-query", "", "Filter included logs by text query")
	statusCmd.Flags().String("since", "", "Filter included logs from this time (RFC3339 or relative duration like 1h)")
	statusCmd.Flags().String("until", "", "Filter included logs until this time (RFC3339 or relative duration like 30m)")
}

var statusCmd = &cobra.Command{
	Use:   "status <name>",
	Short: "Get service details with optional logs and CPU/memory/network metrics",
	Example: `# Show service status
ink status myapi

# Include deploy and runtime logs
ink status myapi --deploy-logs 50 --runtime-logs 100

# Include usage metrics for the last hour
ink status myapi --metrics 1h

# Include usage metrics for the last 24 hours
ink status myapi --metrics 24h

# Filter runtime logs from the last hour
ink status myapi --runtime-logs 100 --log-query timeout --since 1h

# Show everything including env vars
ink status myapi -e --deploy-logs 20 --runtime-logs 50 --metrics 7d`,
	Args: exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		opts, err := inspectOptionsFromCommand(cmd)
		if err != nil {
			fatal(err.Error())
		}
		if err := validateInspectOptions(opts); err != nil {
			fatal(err.Error())
		}
		inspectService(args[0], opts, false)
	},
}
