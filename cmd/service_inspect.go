package cmd

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/mldotink/cli/internal/gql"
	"github.com/spf13/cobra"
)

type logFilterOptions struct {
	query     *string
	startTime *string
	endTime   *string
}

func (f logFilterOptions) used() bool {
	return f.query != nil || f.startTime != nil || f.endTime != nil
}

type serviceInspectOptions struct {
	includeEnv   bool
	deployLines  int
	runtimeLines int
	metricsRange string
	logFilters   logFilterOptions
}

const defaultMetricsMaxDataPoints = 50

type metricSample struct {
	timestamp string
	value     float64
}

func inspectOptionsFromCommand(cmd *cobra.Command) (serviceInspectOptions, error) {
	includeEnv, _ := cmd.Flags().GetBool("env")
	deployLines, _ := cmd.Flags().GetInt("deploy-logs")
	runtimeLines, _ := cmd.Flags().GetInt("runtime-logs")
	metricsRange, _ := cmd.Flags().GetString("metrics")
	logFilters, err := logFiltersFromCommand(cmd, "log-query")
	if err != nil {
		return serviceInspectOptions{}, err
	}

	return serviceInspectOptions{
		includeEnv:   includeEnv,
		deployLines:  clampLogLines(deployLines),
		runtimeLines: clampLogLines(runtimeLines),
		metricsRange: strings.TrimSpace(metricsRange),
		logFilters:   logFilters,
	}, nil
}

func logFiltersFromCommand(cmd *cobra.Command, queryFlagName string) (logFilterOptions, error) {
	query, _ := cmd.Flags().GetString(queryFlagName)
	since, _ := cmd.Flags().GetString("since")
	until, _ := cmd.Flags().GetString("until")

	filters := logFilterOptions{
		query: ptr(strings.TrimSpace(query)),
	}

	if strings.TrimSpace(since) != "" {
		parsed, err := parseLogTimeFlag(since)
		if err != nil {
			return logFilterOptions{}, fmt.Errorf("invalid --since: %w", err)
		}
		filters.startTime = &parsed
	}

	if strings.TrimSpace(until) != "" {
		parsed, err := parseLogTimeFlag(until)
		if err != nil {
			return logFilterOptions{}, fmt.Errorf("invalid --until: %w", err)
		}
		filters.endTime = &parsed
	}

	if filters.startTime != nil && filters.endTime != nil && *filters.startTime > *filters.endTime {
		return logFilterOptions{}, fmt.Errorf("--since must be before --until")
	}

	return filters, nil
}

func parseLogTimeFlag(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC().Format(time.RFC3339Nano), nil
		}
	}

	if duration, err := time.ParseDuration(value); err == nil {
		return time.Now().Add(-duration).UTC().Format(time.RFC3339Nano), nil
	}

	return "", fmt.Errorf("use RFC3339 or a relative duration like 1h")
}

func validateInspectOptions(opts serviceInspectOptions) error {
	if opts.logFilters.used() && opts.deployLines == 0 && opts.runtimeLines == 0 {
		return fmt.Errorf("log filters require --deploy-logs and/or --runtime-logs")
	}
	return nil
}

func hasInspectFlags(opts serviceInspectOptions) bool {
	return opts.includeEnv || opts.deployLines > 0 || opts.runtimeLines > 0 || opts.metricsRange != "" || opts.logFilters.used()
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
		fetchAndPrintLogs(client, svc.Id, gql.LogTypeBuild, opts.deployLines, opts.logFilters, "Deploy Logs")
	}

	if opts.runtimeLines > 0 {
		fetchAndPrintLogs(client, svc.Id, gql.LogTypeRuntime, opts.runtimeLines, opts.logFilters, "Runtime Logs")
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
	if svc.Source == "image" {
		if svc.Image != nil {
			d.kv("Image", *svc.Image)
		}
	} else {
		d.kv("Repo", svc.Repo)
		d.kv("Branch", svc.Branch)
		if svc.CommitHash != nil {
			hash := *svc.CommitHash
			if len(hash) > 12 {
				hash = hash[:12]
			}
			d.kv("Commit", dim.Render(hash))
		}
		d.kv("Git host", svc.GitProvider)
	}
	d.kv("Memory", svc.Memory)
	d.kv("vCPU", svc.Vcpus)
	d.kv("Port", svc.Port)
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

func fetchServiceLogs(client graphql.Client, serviceID string, logType gql.LogType, lines int, filters logFilterOptions) (gql.ServiceLogsServiceLogsLogsResult, error) {
	result, err := gql.ServiceLogs(ctx(), client, gql.LogsInput{
		ServiceId: serviceID,
		LogType:   logType,
		StartTime: filters.startTime,
		EndTime:   filters.endTime,
		Query:     filters.query,
		Limit:     &lines,
	})
	if err != nil {
		return gql.ServiceLogsServiceLogsLogsResult{}, err
	}
	return result.ServiceLogs, nil
}

func fetchAndPrintLogs(client graphql.Client, serviceID string, logType gql.LogType, lines int, filters logFilterOptions, title string) {
	result, err := fetchServiceLogs(client, serviceID, logType, lines, filters)
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
		"24h": gql.MetricTimeRangeTwentyFourHours,
		"7d":  gql.MetricTimeRangeSevenDays,
		"30d": gql.MetricTimeRangeThirtyDays,
	}

	tr, ok := gqlRange[normalized]
	return tr, normalized, ok
}

func fetchAndPrintMetrics(client graphql.Client, serviceID, timeRange string) {
	metrics, normalized, err := fetchServiceMetrics(client, serviceID, timeRange)
	if err != nil {
		fmt.Println()
		fmt.Printf("  %s %s\n", red.Render("!"), dim.Render(err.Error()))
		return
	}

	printMetricsSection(metrics, normalized)
}

func latestMetricPoint[T any](points []T, timestamp func(T) string, value func(T) float64) (string, float64, bool) {
	if len(points) == 0 {
		return "", 0, false
	}
	latest := points[len(points)-1]
	return timestamp(latest), value(latest), true
}

func fetchServiceMetrics(client graphql.Client, serviceID, timeRange string) (gql.ServiceMetricsServiceMetrics, string, error) {
	tr, normalized, ok := resolveMetricTimeRange(timeRange)
	if !ok {
		return gql.ServiceMetricsServiceMetrics{}, "", fmt.Errorf("invalid metrics range %q (use 1h, 6h, 24h, 7d, 30d)", timeRange)
	}

	maxDataPoints := defaultMetricsMaxDataPoints
	result, err := gql.ServiceMetrics(ctx(), client, serviceID, tr, &maxDataPoints)
	if err != nil {
		return gql.ServiceMetricsServiceMetrics{}, "", fmt.Errorf("metrics: %w", err)
	}

	return result.ServiceMetrics, normalized, nil
}

func printMetricsSection(m gql.ServiceMetricsServiceMetrics, timeRange string) bool {
	cpu := metricSamples(m.CpuUsage.DataPoints,
		func(point gql.ServiceMetricsServiceMetricsCpuUsageMetricSeriesDataPointsMetricDataPoint) string {
			return point.Timestamp
		},
		func(point gql.ServiceMetricsServiceMetricsCpuUsageMetricSeriesDataPointsMetricDataPoint) float64 {
			return point.Value
		},
	)
	cpu = clampMetricSamples(cpu, defaultMetricsMaxDataPoints)
	memory := metricSamples(m.MemoryUsageMB.DataPoints,
		func(point gql.ServiceMetricsServiceMetricsMemoryUsageMBMetricSeriesDataPointsMetricDataPoint) string {
			return point.Timestamp
		},
		func(point gql.ServiceMetricsServiceMetricsMemoryUsageMBMetricSeriesDataPointsMetricDataPoint) float64 {
			return point.Value
		},
	)
	memory = clampMetricSamples(memory, defaultMetricsMaxDataPoints)
	netRx := metricSamples(m.NetworkReceiveBytesPerSec.DataPoints,
		func(point gql.ServiceMetricsServiceMetricsNetworkReceiveBytesPerSecMetricSeriesDataPointsMetricDataPoint) string {
			return point.Timestamp
		},
		func(point gql.ServiceMetricsServiceMetricsNetworkReceiveBytesPerSecMetricSeriesDataPointsMetricDataPoint) float64 {
			return point.Value
		},
	)
	netRx = clampMetricSamples(netRx, defaultMetricsMaxDataPoints)
	netTx := metricSamples(m.NetworkTransmitBytesPerSec.DataPoints,
		func(point gql.ServiceMetricsServiceMetricsNetworkTransmitBytesPerSecMetricSeriesDataPointsMetricDataPoint) string {
			return point.Timestamp
		},
		func(point gql.ServiceMetricsServiceMetricsNetworkTransmitBytesPerSecMetricSeriesDataPointsMetricDataPoint) float64 {
			return point.Value
		},
	)
	netTx = clampMetricSamples(netTx, defaultMetricsMaxDataPoints)

	if maxInt(len(cpu), len(memory), len(netRx), len(netTx)) == 0 {
		return false
	}

	start, end := metricWindowBounds(cpu, memory, netRx, netTx)
	pointCount := maxInt(len(cpu), len(memory), len(netRx), len(netTx))

	fmt.Println()
	fmt.Printf("  %s  %s\n",
		bold.Render("Metrics"),
		dim.Render(fmt.Sprintf("(%s, up to %d points)", timeRange, defaultMetricsMaxDataPoints)))
	if start != "" && end != "" {
		fmt.Printf("  %-10s %s -> %s  %s\n",
			"Window",
			dim.Render(fmtTime(start)),
			dim.Render(fmtTime(end)),
			dim.Render(fmt.Sprintf("(%d samples)", pointCount)),
		)
	}

	printMetricSeries(
		"CPU",
		cpu,
		func(value float64) string { return fmt.Sprintf("%.4f vCPUs", value) },
		&m.CpuLimitVCPUs,
	)
	printMetricSeries(
		"Memory",
		memory,
		func(value float64) string { return fmt.Sprintf("%.1f MB", value) },
		&m.MemoryLimitMB,
	)
	printMetricSeries(
		"Net RX",
		netRx,
		formatBytesPerSecond,
		nil,
	)
	printMetricSeries(
		"Net TX",
		netTx,
		formatBytesPerSecond,
		nil,
	)
	printMetricHistoryTable(timeRange, cpu, memory, netRx, netTx)

	return true
}

func metricSamples[T any](points []T, timestamp func(T) string, value func(T) float64) []metricSample {
	samples := make([]metricSample, 0, len(points))
	for _, point := range points {
		samples = append(samples, metricSample{
			timestamp: timestamp(point),
			value:     value(point),
		})
	}
	return samples
}

func clampMetricSamples(samples []metricSample, maxPoints int) []metricSample {
	if len(samples) <= maxPoints || maxPoints <= 0 {
		return samples
	}
	if maxPoints == 1 {
		return []metricSample{samples[len(samples)-1]}
	}

	out := make([]metricSample, 0, maxPoints)
	lastIndex := len(samples) - 1
	for i := 0; i < maxPoints; i++ {
		index := int(math.Round(float64(i*lastIndex) / float64(maxPoints-1)))
		out = append(out, samples[index])
	}
	return out
}

func printMetricSeries(label string, samples []metricSample, formatValue func(float64) string, limit *float64) {
	if len(samples) == 0 {
		return
	}

	latest := samples[len(samples)-1]
	avg, minValue, peak := metricStats(samples)

	if limit != nil && *limit > 0 {
		percent := (latest.value / *limit) * 100
		fmt.Printf("  %-10s current %s  %s of limit\n", label, formatValue(latest.value), dim.Render(fmt.Sprintf("%.1f%%", percent)))
		fmt.Printf("  %-10s avg %s  min %s  peak %s  %s\n",
			"",
			formatValue(avg),
			formatValue(minValue),
			formatValue(peak),
			dim.Render(fmt.Sprintf("(limit %s)", formatValue(*limit))),
		)
	} else {
		fmt.Printf("  %-10s current %s\n", label, formatValue(latest.value))
		fmt.Printf("  %-10s avg %s  min %s  peak %s\n",
			"",
			formatValue(avg),
			formatValue(minValue),
			formatValue(peak),
		)
	}
}

func metricStats(samples []metricSample) (avg, minValue, peak float64) {
	if len(samples) == 0 {
		return 0, 0, 0
	}

	minValue = samples[0].value
	peak = samples[0].value
	for _, sample := range samples {
		avg += sample.value
		if sample.value < minValue {
			minValue = sample.value
		}
		if sample.value > peak {
			peak = sample.value
		}
	}
	return avg / float64(len(samples)), minValue, peak
}

func metricWindowBounds(series ...[]metricSample) (string, string) {
	var start string
	var end string
	for _, samples := range series {
		if len(samples) == 0 {
			continue
		}
		if start == "" || samples[0].timestamp < start {
			start = samples[0].timestamp
		}
		last := samples[len(samples)-1].timestamp
		if end == "" || last > end {
			end = last
		}
	}
	return start, end
}

func printMetricHistoryTable(timeRange string, cpu, memory, netRx, netTx []metricSample) {
	total := maxInt(len(cpu), len(memory), len(netRx), len(netTx))
	if total == 0 {
		return
	}

	indices := selectEvenIndices(total, 8)
	var rows [][]string
	for _, idx := range indices {
		row := []string{
			formatMetricTimeLabel(sampleAtRelativeIndex(cpu, idx, total), timeRange),
			formatMetricValueAt(cpu, idx, total, func(value float64) string { return fmt.Sprintf("%.4f", value) }),
			formatMetricValueAt(memory, idx, total, func(value float64) string { return fmt.Sprintf("%.1f MB", value) }),
			formatMetricValueAt(netRx, idx, total, formatBytesPerSecond),
			formatMetricValueAt(netTx, idx, total, formatBytesPerSecond),
		}
		rows = append(rows, row)
	}

	if len(rows) == 0 {
		return
	}

	fmt.Println()
	fmt.Println(bold.Render("  History"))
	fmt.Println(styledTable([]string{"TIME", "CPU", "MEMORY", "NET RX", "NET TX"}, rows))
}

func selectEvenIndices(total, maxRows int) []int {
	if total <= 0 || maxRows <= 0 {
		return nil
	}
	if total <= maxRows {
		out := make([]int, total)
		for i := range total {
			out[i] = i
		}
		return out
	}

	out := make([]int, 0, maxRows)
	seen := make(map[int]struct{}, maxRows)
	for i := 0; i < maxRows; i++ {
		idx := int(math.Round(float64(i*(total-1)) / float64(maxRows-1)))
		if _, ok := seen[idx]; ok {
			continue
		}
		seen[idx] = struct{}{}
		out = append(out, idx)
	}
	return out
}

func sampleAtRelativeIndex(samples []metricSample, index, total int) *metricSample {
	if len(samples) == 0 || total <= 0 {
		return nil
	}
	if len(samples) == total {
		return &samples[index]
	}
	if total == 1 {
		return &samples[len(samples)-1]
	}

	mapped := int(math.Round(float64(index*(len(samples)-1)) / float64(total-1)))
	if mapped < 0 {
		mapped = 0
	}
	if mapped >= len(samples) {
		mapped = len(samples) - 1
	}
	return &samples[mapped]
}

func formatMetricValueAt(samples []metricSample, index, total int, formatValue func(float64) string) string {
	sample := sampleAtRelativeIndex(samples, index, total)
	if sample == nil {
		return dim.Render("—")
	}
	return formatValue(sample.value)
}

func formatMetricTimeLabel(sample *metricSample, timeRange string) string {
	if sample == nil {
		return dim.Render("—")
	}

	layout := "15:04"
	if timeRange == "24h" || timeRange == "7d" || timeRange == "30d" {
		layout = "01-02 15:04"
	}

	t, err := time.Parse(time.RFC3339Nano, sample.timestamp)
	if err != nil {
		return sample.timestamp
	}
	return t.Local().Format(layout)
}

func maxInt(values ...int) int {
	best := 0
	for _, value := range values {
		if value > best {
			best = value
		}
	}
	return best
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
