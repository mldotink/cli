package cmd

import (
	"fmt"

	"github.com/mldotink/cli/internal/gql"
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

		result, err := gql.AddDomain(ctx(), client, svc, domain, projPtr(), wsPtr())
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
	Example: `ink domain remove myapi`,
	Args:    exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc := args[0]
		client := newClient()

		result, err := gql.RemoveDomain(ctx(), client, svc, projPtr(), wsPtr())
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
