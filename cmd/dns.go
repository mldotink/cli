package cmd

import (
	"fmt"
	"strconv"

	"github.com/mldotink/cli/internal/gql"
	"github.com/spf13/cobra"
)

func init() {
	dnsCmd.AddCommand(dnsZonesCmd)
	dnsCmd.AddCommand(dnsRecordsCmd)
	dnsAddCmd.Flags().Int("ttl", 300, "Time to live in seconds")
	dnsCmd.AddCommand(dnsAddCmd)
	dnsCmd.AddCommand(dnsDeleteCmd)
}

var dnsCmd = &cobra.Command{
	Use:   "dns",
	Short: "Manage DNS zones and records for delegated domains",
	Long: `Manage DNS zones and records. Zone delegation must be set up first at
https://ml.ink/dns — add a TXT verification record and point NS records
to ns1.ml.ink / ns2.ml.ink. Once active, add A, AAAA, CNAME, MX, TXT, and CAA records.`,
	Example: `ink dns zones
ink dns records example.com
ink dns add example.com www A 1.2.3.4
ink dns add example.com api CNAME myapi.ml.ink
ink dns delete example.com rec_abc123`,
}

var dnsZonesCmd = &cobra.Command{
	Use:   "zones",
	Short: "List DNS zones",
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()

		result, err := gql.ListDnsZones(ctx(), client, wsPtr())
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
				status += dim.Render(" — " + *z.Error)
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
	Example: `ink dns records example.com`,
	Args:    exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		zone := args[0]
		client := newClient()

		result, err := gql.ListDnsRecords(ctx(), client, zone, wsPtr())
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
			rows = append(rows, []string{r.Name, r.Type, r.Content, strconv.Itoa(r.Ttl), managed})
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
	Long:  "Add a DNS record. Supported types: A, AAAA, CNAME, MX, TXT, CAA",
	Example: `# Add an A record
ink dns add example.com @ A 1.2.3.4

# Add a CNAME record
ink dns add example.com www CNAME example.com

# Add a TXT record with custom TTL
ink dns add example.com @ TXT "v=spf1 include:_spf.google.com ~all" --ttl 3600`,
	Args: exactArgs(4),
	Run: func(cmd *cobra.Command, args []string) {
		zone, name, typ, content := args[0], args[1], args[2], args[3]
		ttl, _ := cmd.Flags().GetInt("ttl")
		client := newClient()

		result, err := gql.AddDnsRecord(ctx(), client, zone, name, typ, content, &ttl, wsPtr())
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
	Example: `# List records first, then delete by ID
ink dns records example.com
ink dns delete example.com rec_abc123`,
	Args: exactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		zone, recordID := args[0], args[1]
		client := newClient()

		result, err := gql.DeleteDnsRecord(ctx(), client, zone, recordID, wsPtr())
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
