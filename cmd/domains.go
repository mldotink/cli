package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	domainsCmd.AddCommand(domainsAddCmd)
	domainsCmd.AddCommand(domainsRemoveCmd)
	rootCmd.AddCommand(domainsCmd)
}

var domainsCmd = &cobra.Command{
	Use:   "domains",
	Short: "Manage custom domains",
}

var domainsAddCmd = &cobra.Command{
	Use:   "add <service> <domain>",
	Short: "Add a custom domain to a service",
	Example: `ink domains add myapi api.example.com`,
	Args:    exactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		svc, domain := args[0], args[1]
		client := newClient()

		var result struct {
			DomainAdd struct {
				ServiceID string `json:"serviceId"`
				Domain    string `json:"domain"`
				Status    string `json:"status"`
				Message   string `json:"message"`
			} `json:"domainAdd"`
		}

		err := client.Do(`mutation($name: String!, $domain: String!, $ws: String, $proj: String) {
			domainAdd(name: $name, domain: $domain, workspaceSlug: $ws, project: $proj) {
				serviceId domain status message
			}
		}`, mergeVars(map[string]any{"name": svc, "domain": domain}), &result)
		if err != nil {
			fatal(err.Error())
		}

		d := result.DomainAdd
		if jsonOutput {
			printJSON(d)
			return
		}

		fmt.Println()
		success(fmt.Sprintf("Domain %s added to %s", bold.Render(d.Domain), bold.Render(svc)))
		kv("Status", renderStatus(d.Status))
		if d.Message != "" {
			kv("Note", d.Message)
		}
		fmt.Println()
	},
}

var domainsRemoveCmd = &cobra.Command{
	Use:   "remove <service>",
	Short: "Remove custom domain from a service",
	Example: `ink domains remove myapi`,
	Args:    exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc := args[0]
		client := newClient()

		var result struct {
			DomainRemove struct {
				ServiceID string `json:"serviceId"`
				Message   string `json:"message"`
			} `json:"domainRemove"`
		}

		err := client.Do(`mutation($name: String!, $ws: String, $proj: String) {
			domainRemove(name: $name, workspaceSlug: $ws, project: $proj) { serviceId message }
		}`, mergeVars(map[string]any{"name": svc}), &result)
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(result.DomainRemove)
			return
		}

		success(result.DomainRemove.Message)
	},
}
