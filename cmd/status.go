package cmd

import (
	"fmt"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/mldotink/cli/internal/gql"
	"github.com/spf13/cobra"
)

var statusIncludeEnv bool

func init() {
	statusCmd.Flags().BoolVarP(&statusIncludeEnv, "env", "e", false, "Show environment variables")
	statusCmd.Flags().Int("deploy-logs", 0, "Include N deploy log lines (max 500)")
	statusCmd.Flags().Int("runtime-logs", 0, "Include N runtime log lines (max 500)")
	statusCmd.Flags().String("metrics", "", "Include CPU/memory metrics: 1h, 6h, 7d, 30d")
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status <name>",
	Short: "Get service details",
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
		name := args[0]
		client := newClient()

		svc := findService(name)
		if svc == nil {
			fatal(fmt.Sprintf("Service %q not found", name))
		}

		if jsonOutput {
			printJSON(svc)
			return
		}

		d := newDetail(deref(svc.Name, ""))
		d.kv("Status", renderStatus(svc.Status))
		if svc.ErrorMessage != nil {
			d.kv("Error", red.Render(*svc.ErrorMessage))
		}
		if svc.Fqdn != nil {
			d.kv("URL", accent.Render(*svc.Fqdn))
		}
		d.kv("Repo", svc.Repo)
		d.kv("Branch", svc.Branch)
		if svc.CommitHash != nil {
			hash := *svc.CommitHash
			if len(hash) > 12 {
				hash = hash[:12]
			}
			d.kv("Commit", dim.Render(hash))
		}
		d.kv("Memory", svc.Memory)
		d.kv("vCPU", svc.Vcpus)
		d.kv("Port", svc.Port)
		d.kv("Git host", svc.GitProvider)
		if svc.CustomDomain != nil {
			status := ""
			if svc.CustomDomainStatus != nil {
				status = " " + renderStatus(*svc.CustomDomainStatus)
			}
			d.kv("Domain", *svc.CustomDomain+status)
		}
		d.kv("Internal URL", svc.InternalUrl)

		if statusIncludeEnv && len(svc.EnvVars) > 0 {
			d.section("Environment")
			for _, e := range svc.EnvVars {
				d.line(fmt.Sprintf("  %s=%s", bold.Render(e.Key), e.Value))
			}
		}

		fmt.Println()
		fmt.Println(d.String())

		// Deploy logs
		if deployLines, _ := cmd.Flags().GetInt("deploy-logs"); deployLines > 0 {
			if deployLines > 500 {
				deployLines = 500
			}
			fetchAndPrintLogs(client, svc.Id, gql.LogTypeBuild, deployLines, "Deploy Logs")
		}

		// Runtime logs
		if runtimeLines, _ := cmd.Flags().GetInt("runtime-logs"); runtimeLines > 0 {
			if runtimeLines > 500 {
				runtimeLines = 500
			}
			fetchAndPrintLogs(client, svc.Id, gql.LogTypeRuntime, runtimeLines, "Runtime Logs")
		}

		// Metrics
		if metricsRange, _ := cmd.Flags().GetString("metrics"); metricsRange != "" {
			fetchAndPrintMetrics(client, svc.Id, metricsRange)
		}

		fmt.Println()
	},
}

func fetchAndPrintLogs(client graphql.Client, serviceID string, logType gql.LogType, lines int, title string) {
	result, err := gql.ServiceLogs(ctx(), client, gql.LogsInput{
		ServiceId: serviceID,
		LogType:   logType,
		Limit:     &lines,
	})
	if err != nil {
		fmt.Println()
		fmt.Printf("  %s %s\n", red.Render("!"), dim.Render(title+": "+err.Error()))
		return
	}

	entries := result.ServiceLogs.Entries
	if len(entries) == 0 {
		return
	}

	fmt.Println()
	fmt.Println(bold.Render("  " + title))
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
		fmt.Printf("  %s %s%s\n", ts, level, e.Message)
	}
}

func fetchAndPrintMetrics(client graphql.Client, serviceID, timeRange string) {
	gqlRange := map[string]gql.MetricTimeRange{
		"1h":  gql.MetricTimeRangeOneHour,
		"6h":  gql.MetricTimeRangeSixHours,
		"7d":  gql.MetricTimeRangeSevenDays,
		"30d": gql.MetricTimeRangeThirtyDays,
	}

	tr, ok := gqlRange[strings.ToLower(timeRange)]
	if !ok {
		fmt.Printf("  %s Invalid metrics range %q (use 1h, 6h, 7d, 30d)\n", red.Render("!"), timeRange)
		return
	}

	result, err := gql.ServiceMetrics(ctx(), client, serviceID, tr)
	if err != nil {
		fmt.Println()
		fmt.Printf("  %s %s\n", red.Render("!"), dim.Render("Metrics: "+err.Error()))
		return
	}

	m := result.ServiceMetrics
	cpuPts := m.CpuUsage.DataPoints
	memPts := m.MemoryUsageMB.DataPoints
	if len(cpuPts) == 0 && len(memPts) == 0 {
		return
	}

	fmt.Println()
	fmt.Printf("  %s  %s\n", bold.Render("Metrics"), dim.Render("("+timeRange+")"))

	if len(cpuPts) > 0 {
		latest := cpuPts[len(cpuPts)-1]
		fmt.Printf("  CPU        %.4f / %.2f vCPUs  %s\n",
			latest.Value, m.CpuLimitVCPUs, dim.Render(latest.Timestamp))
	}
	if len(memPts) > 0 {
		latest := memPts[len(memPts)-1]
		fmt.Printf("  Memory     %.1f / %.0f MB  %s\n",
			latest.Value, m.MemoryLimitMB, dim.Render(latest.Timestamp))
	}
}
