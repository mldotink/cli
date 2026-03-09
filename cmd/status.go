package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var statusIncludeEnv bool

func init() {
	statusCmd.Flags().BoolVarP(&statusIncludeEnv, "env", "e", false, "Show environment variables")
	statusCmd.Flags().Int("build-logs", 0, "Number of build log lines to include (max 500)")
	statusCmd.Flags().Int("runtime-logs", 0, "Number of runtime log lines to include (max 500)")
	statusCmd.Flags().String("metrics", "", "Include usage metrics: 1h, 6h, 7d, 30d")
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status <name>",
	Short: "Get service details",
	Args:  exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		client := newClient()

		svc, err := findService(client, name)
		if err != nil {
			fatal(err.Error())
		}
		if svc == nil {
			fatal(fmt.Sprintf("Service %q not found", name))
		}

		if jsonOutput {
			printJSON(svc)
			return
		}

		d := newDetail(svc.Name)
		d.kv("Status", renderStatus(svc.Status))
		if svc.ErrorMessage != nil {
			d.kv("Error", red.Render(*svc.ErrorMessage))
		}
		if svc.FQDN != nil {
			d.kv("URL", accent.Render(*svc.FQDN))
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
		d.kv("vCPUs", svc.VCPUs)
		d.kv("Git provider", svc.GitProvider)
		if svc.CustomDomain != nil {
			status := ""
			if svc.CustomDomainStatus != nil {
				status = " " + renderStatus(*svc.CustomDomainStatus)
			}
			d.kv("Domain", *svc.CustomDomain+status)
		}
		d.kv("Internal URL", svc.InternalURL)

		if statusIncludeEnv && len(svc.EnvVars) > 0 {
			d.section("Environment")
			for _, e := range svc.EnvVars {
				d.line(fmt.Sprintf("  %s=%s", bold.Render(e.Key), e.Value))
			}
		}

		fmt.Println()
		fmt.Println(d.String())

		// Build logs
		if buildLines, _ := cmd.Flags().GetInt("build-logs"); buildLines > 0 {
			if buildLines > 500 {
				buildLines = 500
			}
			fetchAndPrintLogs(client, svc.ID, "BUILD", buildLines, "Build Logs")
		}

		// Runtime logs
		if runtimeLines, _ := cmd.Flags().GetInt("runtime-logs"); runtimeLines > 0 {
			if runtimeLines > 500 {
				runtimeLines = 500
			}
			fetchAndPrintLogs(client, svc.ID, "RUNTIME", runtimeLines, "Runtime Logs")
		}

		// Metrics
		if metricsRange, _ := cmd.Flags().GetString("metrics"); metricsRange != "" {
			fetchAndPrintMetrics(client, svc.ID, metricsRange)
		}

		fmt.Println()
	},
}

func fetchAndPrintLogs(client interface{ Do(string, map[string]any, any) error }, serviceID, logType string, lines int, title string) {
	var result struct {
		ServiceLogs struct {
			Entries []struct {
				Timestamp string  `json:"timestamp"`
				Level     *string `json:"level"`
				Message   string  `json:"message"`
			} `json:"entries"`
		} `json:"serviceLogs"`
	}

	err := client.Do(`query($input: LogsInput!) {
		serviceLogs(input: $input) {
			entries { timestamp level message }
		}
	}`, map[string]any{
		"input": map[string]any{
			"serviceId": serviceID,
			"logType":   logType,
			"limit":     lines,
		},
	}, &result)
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

func fetchAndPrintMetrics(client interface{ Do(string, map[string]any, any) error }, serviceID, timeRange string) {
	gqlRange := map[string]string{
		"1h":  "ONE_HOUR",
		"6h":  "SIX_HOURS",
		"7d":  "SEVEN_DAYS",
		"30d": "THIRTY_DAYS",
	}

	tr, ok := gqlRange[strings.ToLower(timeRange)]
	if !ok {
		fmt.Printf("  %s Invalid metrics range %q (use 1h, 6h, 7d, 30d)\n", red.Render("!"), timeRange)
		return
	}

	var result struct {
		ServiceMetrics struct {
			CPUUsage struct {
				DataPoints []struct {
					Timestamp string  `json:"timestamp"`
					Value     float64 `json:"value"`
				} `json:"dataPoints"`
			} `json:"cpuUsage"`
			MemoryUsageMB struct {
				DataPoints []struct {
					Timestamp string  `json:"timestamp"`
					Value     float64 `json:"value"`
				} `json:"dataPoints"`
			} `json:"memoryUsageMB"`
			MemoryLimitMB float64 `json:"memoryLimitMB"`
			CPULimitVCPUs float64 `json:"cpuLimitVCPUs"`
		} `json:"serviceMetrics"`
	}

	err := client.Do(`query($id: ID!, $range: MetricTimeRange!) {
		serviceMetrics(serviceId: $id, timeRange: $range) {
			cpuUsage { dataPoints { timestamp value } }
			memoryUsageMB { dataPoints { timestamp value } }
			memoryLimitMB cpuLimitVCPUs
		}
	}`, map[string]any{
		"id":    serviceID,
		"range": tr,
	}, &result)
	if err != nil {
		fmt.Println()
		fmt.Printf("  %s %s\n", red.Render("!"), dim.Render("Metrics: "+err.Error()))
		return
	}

	m := result.ServiceMetrics
	cpuPts := m.CPUUsage.DataPoints
	memPts := m.MemoryUsageMB.DataPoints
	if len(cpuPts) == 0 && len(memPts) == 0 {
		return
	}

	fmt.Println()
	fmt.Printf("  %s  %s\n", bold.Render("Metrics"), dim.Render("("+timeRange+")"))

	if len(cpuPts) > 0 {
		latest := cpuPts[len(cpuPts)-1]
		fmt.Printf("  CPU        %.4f / %.2f vCPUs  %s\n",
			latest.Value, m.CPULimitVCPUs, dim.Render(latest.Timestamp))
	}
	if len(memPts) > 0 {
		latest := memPts[len(memPts)-1]
		fmt.Printf("  Memory     %.1f / %.0f MB  %s\n",
			latest.Value, m.MemoryLimitMB, dim.Render(latest.Timestamp))
	}
}
