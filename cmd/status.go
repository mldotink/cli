package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	statusCmd.Flags().BoolP("env", "e", false, "Show environment variables")
	statusCmd.Flags().Int("deploy-logs", 0, "Include N deploy log lines (max 500)")
	statusCmd.Flags().Int("runtime-logs", 0, "Include N runtime log lines (max 500)")
	statusCmd.Flags().String("metrics", "", "Include CPU/memory/network metrics: 1h, 6h, 7d, 30d")
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

# Show everything including env vars
ink status myapi -e --deploy-logs 20 --runtime-logs 50 --metrics 7d`,
	Args: exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inspectService(args[0], inspectOptionsFromCommand(cmd), false)
	},
}
