package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	domainsCmd.AddCommand(domainsAddCmd)
	domainsCmd.AddCommand(domainsRemoveCmd)
}

var domainsCmd = &cobra.Command{
	Use:     "domain",
	Aliases: []string{"domains"},
	Short:   "Attach or remove custom domains from services",
	Long: `Manage custom domains for your services. Requires a delegated DNS zone first —
visit https://ml.ink/dns to add a TXT verification record and point NS records
to ns1.ml.ink / ns2.ml.ink. Once the zone is active, attach subdomains to services.`,
	Example: `ink domain add myapi api.example.com
ink domain remove myapi`,
}

var domainsAddCmd = &cobra.Command{
	Use:   "add <service> <domain>",
	Short: "Add a custom domain to a service",
	Example: `ink domain add myapi api.example.com`,
	Args:    exactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		svc, domain := args[0], args[1]
		client := newClient()

		result, err := client.AddDomain(ctx(), svc, domain, cfg.Project, cfg.Workspace)
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(result)
			return
		}

		fmt.Println()
		success(fmt.Sprintf("Domain %s added to %s", bold.Render(result.Domain), bold.Render(svc)))
		kv("Status", renderStatus(result.Status))
		if result.Message != "" {
			kv("Note", result.Message)
		}
		fmt.Println()
	},
}

var domainsRemoveCmd = &cobra.Command{
	Use:   "remove <service>",
	Short: "Remove custom domain from a service",
	Example: `ink domain remove myapi`,
	Args:    exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc := args[0]
		client := newClient()

		result, err := client.RemoveDomain(ctx(), svc, cfg.Project, cfg.Workspace)
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(result)
			return
		}

		success(result.Message)
	},
}
