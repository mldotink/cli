package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func init() {
	dnsCmd.AddCommand(dnsZonesCmd)
	dnsCmd.AddCommand(dnsRecordsCmd)
	dnsAddCmd.Flags().Int("ttl", 300, "TTL in seconds")
	dnsCmd.AddCommand(dnsAddCmd)
	dnsCmd.AddCommand(dnsDeleteCmd)
	rootCmd.AddCommand(dnsCmd)
}

var dnsCmd = &cobra.Command{
	Use:   "dns",
	Short: "Manage DNS zones and records",
}

var dnsZonesCmd = &cobra.Command{
	Use:   "zones",
	Short: "List DNS zones",
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()

		var result struct {
			DnsListZones []struct {
				ID     string  `json:"id"`
				Zone   string  `json:"zone"`
				Status string  `json:"status"`
				Error  *string `json:"error"`
			} `json:"dnsListZones"`
		}

		err := client.Do(`query($ws: String) {
			dnsListZones(workspaceSlug: $ws) { id zone status error }
		}`, defaultVars(), &result)
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(result.DnsListZones)
			return
		}

		zones := result.DnsListZones
		if len(zones) == 0 {
			fmt.Println(dim.Render("  No DNS zones"))
			return
		}

		var rows [][]string
		for _, z := range zones {
			status := renderStatus(z.Status)
			if z.Error != nil {
				status += dim.Render(" — "+*z.Error)
			}
			rows = append(rows, []string{z.Zone, status})
		}

		fmt.Println()
		fmt.Println(styledTable([]string{"ZONE", "STATUS"}, rows))
		tableFooter(len(zones), "zone")
		fmt.Println()
	},
}

var dnsRecordsCmd = &cobra.Command{
	Use:   "records <zone>",
	Short: "List DNS records in a zone",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		zone := args[0]
		client := newClient()

		var result struct {
			DnsListRecords []struct {
				ID      string `json:"id"`
				Name    string `json:"name"`
				Type    string `json:"type"`
				Content string `json:"content"`
				TTL     int    `json:"ttl"`
				Managed bool   `json:"managed"`
			} `json:"dnsListRecords"`
		}

		err := client.Do(`query($zone: String!, $ws: String) {
			dnsListRecords(zone: $zone, workspaceSlug: $ws) {
				id name type content ttl managed
			}
		}`, mergeVars(map[string]any{"zone": zone}), &result)
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(result.DnsListRecords)
			return
		}

		records := result.DnsListRecords
		if len(records) == 0 {
			fmt.Println(dim.Render("  No records"))
			return
		}

		var rows [][]string
		for _, r := range records {
			managed := ""
			if r.Managed {
				managed = dim.Render("system")
			}
			rows = append(rows, []string{r.Name, r.Type, r.Content, strconv.Itoa(r.TTL), managed})
		}

		fmt.Println()
		fmt.Println(styledTable([]string{"NAME", "TYPE", "CONTENT", "TTL", "MANAGED"}, rows))
		tableFooter(len(records), "record")
		fmt.Println()
	},
}

var dnsAddCmd = &cobra.Command{
	Use:   "add <zone> <name> <type> <content>",
	Short: "Add a DNS record",
	Long:  "Add a DNS record. Types: A, AAAA, CNAME, MX, TXT, CAA",
	Args:  cobra.ExactArgs(4),
	Run: func(cmd *cobra.Command, args []string) {
		zone, name, typ, content := args[0], args[1], args[2], args[3]
		ttl, _ := cmd.Flags().GetInt("ttl")
		client := newClient()

		var result struct {
			DnsAddRecord struct {
				ID      string `json:"id"`
				Name    string `json:"name"`
				Type    string `json:"type"`
				Content string `json:"content"`
				TTL     int    `json:"ttl"`
			} `json:"dnsAddRecord"`
		}

		err := client.Do(`mutation($zone: String!, $name: String!, $type: String!, $content: String!, $ttl: Int, $ws: String) {
			dnsAddRecord(zone: $zone, name: $name, type: $type, content: $content, ttl: $ttl, workspaceSlug: $ws) {
				id name type content ttl
			}
		}`, mergeVars(map[string]any{
			"zone": zone, "name": name, "type": typ, "content": content, "ttl": ttl,
		}), &result)
		if err != nil {
			fatal(err.Error())
		}

		r := result.DnsAddRecord
		if jsonOutput {
			printJSON(r)
			return
		}

		success(fmt.Sprintf("Record added: %s %s %s", r.Name, bold.Render(r.Type), r.Content))
	},
}

var dnsDeleteCmd = &cobra.Command{
	Use:   "delete <zone> <record-id>",
	Short: "Delete a DNS record",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		zone, recordID := args[0], args[1]
		client := newClient()

		var result struct {
			DnsDeleteRecord bool `json:"dnsDeleteRecord"`
		}

		err := client.Do(`mutation($zone: String!, $recordId: ID!, $ws: String) {
			dnsDeleteRecord(zone: $zone, recordId: $recordId, workspaceSlug: $ws)
		}`, mergeVars(map[string]any{"zone": zone, "recordId": recordID}), &result)
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(map[string]any{"deleted": result.DnsDeleteRecord})
			return
		}

		success("Record deleted")
	},
}
