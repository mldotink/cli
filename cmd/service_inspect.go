package cmd

import (
	"fmt"
	"math"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/mldotink/cli/internal/gql"
	"github.com/spf13/cobra"
)

type serviceInspectOptions struct {
	includeEnv   bool
	deployLines  int
	runtimeLines int
	metricsRange string
}

func inspectOptionsFromCommand(cmd *cobra.Command) serviceInspectOptions {
	includeEnv, _ := cmd.Flags().GetBool("env")
	deployLines, _ := cmd.Flags().GetInt("deploy-logs")
	runtimeLines, _ := cmd.Flags().GetInt("runtime-logs")
	metricsRange, _ := cmd.Flags().GetString("metrics")

	return serviceInspectOptions{
		includeEnv:   includeEnv,
		deployLines:  clampLogLines(deployLines),
		runtimeLines: clampLogLines(runtimeLines),
		metricsRange: strings.TrimSpace(metricsRange),
	}
}

func hasInspectFlags(opts serviceInspectOptions) bool {
	return opts.includeEnv || opts.deployLines > 0 || opts.runtimeLines > 0 || opts.metricsRange != ""
}

func clampLogLines(lines int) int {
	switch {
	case lines < 0:
		return 0
	case lines > 500:
		return 500
	default:
		return lines
	}
}

func inspectService(name string, opts serviceInspectOptions, printTip bool) {
	svc := findService(name)
	if svc == nil {
		fatal(fmt.Sprintf("Service %q not found", name))
	}

	if jsonOutput {
		printJSON(svc)
		return
	}

	fmt.Println()
	fmt.Println(renderServiceDetail(svc, opts.includeEnv))

	client := newClient()

	if opts.deployLines > 0 {
		fetchAndPrintLogs(client, svc.Id, gql.LogTypeBuild, opts.deployLines, "Deploy Logs")
	}

	if opts.runtimeLines > 0 {
		fetchAndPrintLogs(client, svc.Id, gql.LogTypeRuntime, opts.runtimeLines, "Runtime Logs")
	}

	if opts.metricsRange != "" {
		fetchAndPrintMetrics(client, svc.Id, opts.metricsRange)
	}

	if printTip && !hasInspectFlags(opts) {
		fmt.Println()
		fmt.Println(dim.Render(fmt.Sprintf("  Tip: ink service %s --deploy-logs 20 --runtime-logs 50 --metrics 1h", name)))
	}

	fmt.Println()
}

func renderServiceDetail(svc *gql.FindServiceServiceListServiceConnectionNodesService, includeEnv bool) string {
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
	if svc.Project != nil && svc.Project.Slug != "" {
		d.kv("Project", svc.Project.Slug)
	}
	if svc.CreatedAt != "" {
		d.kv("Created", dim.Render(fmtTime(svc.CreatedAt)))
	}
	if svc.UpdatedAt != "" {
		d.kv("Updated", dim.Render(fmtTime(svc.UpdatedAt)))
	}

	if includeEnv && len(svc.EnvVars) > 0 {
		d.section("Environment")
		for _, e := range svc.EnvVars {
			d.line(fmt.Sprintf("  %s=%s", bold.Render(e.Key), e.Value))
		}
	} else if len(svc.EnvVars) > 0 {
		d.blank()
		d.line(dim.Render(fmt.Sprintf("  %d env var%s (use --env to view)", len(svc.EnvVars), pluralS(len(svc.EnvVars)))))
	}

	return d.String()
}

func fetchServiceLogs(client graphql.Client, serviceID string, logType gql.LogType, lines int) (gql.ServiceLogsServiceLogsLogsResult, error) {
	result, err := gql.ServiceLogs(ctx(), client, gql.LogsInput{
		ServiceId: serviceID,
		LogType:   logType,
		Limit:     &lines,
	})
	if err != nil {
		return gql.ServiceLogsServiceLogsLogsResult{}, err
	}
	return result.ServiceLogs, nil
}

func fetchAndPrintLogs(client graphql.Client, serviceID string, logType gql.LogType, lines int, title string) {
	result, err := fetchServiceLogs(client, serviceID, logType, lines)
	if err != nil {
		fmt.Println()
		fmt.Printf("  %s %s\n", red.Render("!"), dim.Render(title+": "+err.Error()))
		return
	}

	if len(result.Entries) == 0 {
		return
	}

	fmt.Println()
	fmt.Println(bold.Render("  " + title))
	printLogEntries(result.Entries, "  ")
}

func printLogEntries(entries []gql.ServiceLogsServiceLogsLogsResultEntriesLogEntry, indent string) {
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
		fmt.Printf("%s%s %s%s\n", indent, ts, level, e.Message)
	}
}

func resolveMetricTimeRange(timeRange string) (gql.MetricTimeRange, string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(timeRange))
	gqlRange := map[string]gql.MetricTimeRange{
		"1h":  gql.MetricTimeRangeOneHour,
		"6h":  gql.MetricTimeRangeSixHours,
		"7d":  gql.MetricTimeRangeSevenDays,
		"30d": gql.MetricTimeRangeThirtyDays,
	}

	tr, ok := gqlRange[normalized]
	return tr, normalized, ok
}

func fetchAndPrintMetrics(client graphql.Client, serviceID, timeRange string) {
	tr, normalized, ok := resolveMetricTimeRange(timeRange)
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

	printMetricsSection(result.ServiceMetrics, normalized)
}

func latestMetricPoint[T any](points []T, timestamp func(T) string, value func(T) float64) (string, float64, bool) {
	if len(points) == 0 {
		return "", 0, false
	}
	latest := points[len(points)-1]
	return timestamp(latest), value(latest), true
}

func printMetricsSection(m gql.ServiceMetricsServiceMetrics, timeRange string) {
	if len(m.CpuUsage.DataPoints) == 0 &&
		len(m.MemoryUsageMB.DataPoints) == 0 &&
		len(m.NetworkReceiveBytesPerSec.DataPoints) == 0 &&
		len(m.NetworkTransmitBytesPerSec.DataPoints) == 0 {
		return
	}

	fmt.Println()
	fmt.Printf("  %s  %s\n", bold.Render("Metrics"), dim.Render("("+timeRange+")"))

	if ts, value, ok := latestMetricPoint(m.CpuUsage.DataPoints,
		func(point gql.ServiceMetricsServiceMetricsCpuUsageMetricSeriesDataPointsMetricDataPoint) string {
			return point.Timestamp
		},
		func(point gql.ServiceMetricsServiceMetricsCpuUsageMetricSeriesDataPointsMetricDataPoint) float64 {
			return point.Value
		},
	); ok {
		fmt.Printf("  CPU        %.4f / %.2f vCPUs  %s\n",
			value, m.CpuLimitVCPUs, dim.Render(ts))
	}

	if ts, value, ok := latestMetricPoint(m.MemoryUsageMB.DataPoints,
		func(point gql.ServiceMetricsServiceMetricsMemoryUsageMBMetricSeriesDataPointsMetricDataPoint) string {
			return point.Timestamp
		},
		func(point gql.ServiceMetricsServiceMetricsMemoryUsageMBMetricSeriesDataPointsMetricDataPoint) float64 {
			return point.Value
		},
	); ok {
		fmt.Printf("  Memory     %.1f / %.0f MB  %s\n",
			value, m.MemoryLimitMB, dim.Render(ts))
	}

	if ts, value, ok := latestMetricPoint(m.NetworkReceiveBytesPerSec.DataPoints,
		func(point gql.ServiceMetricsServiceMetricsNetworkReceiveBytesPerSecMetricSeriesDataPointsMetricDataPoint) string {
			return point.Timestamp
		},
		func(point gql.ServiceMetricsServiceMetricsNetworkReceiveBytesPerSecMetricSeriesDataPointsMetricDataPoint) float64 {
			return point.Value
		},
	); ok {
		fmt.Printf("  Net RX     %s  %s\n",
			formatBytesPerSecond(value), dim.Render(ts))
	}

	if ts, value, ok := latestMetricPoint(m.NetworkTransmitBytesPerSec.DataPoints,
		func(point gql.ServiceMetricsServiceMetricsNetworkTransmitBytesPerSecMetricSeriesDataPointsMetricDataPoint) string {
			return point.Timestamp
		},
		func(point gql.ServiceMetricsServiceMetricsNetworkTransmitBytesPerSecMetricSeriesDataPointsMetricDataPoint) float64 {
			return point.Value
		},
	); ok {
		fmt.Printf("  Net TX     %s  %s\n",
			formatBytesPerSecond(value), dim.Render(ts))
	}
}

func formatBytesPerSecond(value float64) string {
	units := []string{"B/s", "KiB/s", "MiB/s", "GiB/s", "TiB/s"}
	v := math.Abs(value)
	unit := 0
	for v >= 1024 && unit < len(units)-1 {
		v /= 1024
		unit++
	}

	scaled := value / math.Pow(1024, float64(unit))
	switch {
	case math.Abs(scaled) >= 100:
		return fmt.Sprintf("%.0f %s", scaled, units[unit])
	case math.Abs(scaled) >= 10:
		return fmt.Sprintf("%.1f %s", scaled, units[unit])
	default:
		return fmt.Sprintf("%.2f %s", scaled, units[unit])
	}
}
