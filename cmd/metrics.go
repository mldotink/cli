package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	metricsCmd.Flags().StringP("range", "r", "1h", "Metrics range: 1h, 6h, 7d, 30d")
}

var metricsCmd = &cobra.Command{
	Use:   "metrics <name>",
	Short: "View CPU/memory/network metrics for a service",
	Example: `# View the last hour of metrics
ink metrics myapi

# View the last 7 days
ink metrics myapi --range 7d

# Output raw metric series as JSON
ink metrics myapi --range 30d --json`,
	Args: exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rangeFlag, _ := cmd.Flags().GetString("range")

		svc := findService(args[0])
		if svc == nil {
			fatal(fmt.Sprintf("Service %q not found", args[0]))
		}

		metrics, normalized, err := fetchServiceMetrics(newClient(), svc.Id, rangeFlag)
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(metrics)
			return
		}

		fmt.Println()
		fmt.Println(titleStyle.Render(deref(svc.Name, args[0])))
		if !printMetricsSection(metrics, normalized) {
			fmt.Println(dim.Render("  No metrics"))
		}
		fmt.Println()
	},
}
