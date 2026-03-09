package cmd

import (
	"fmt"

	"github.com/mldotink/ink-cli/internal/api"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(servicesCmd)
}

type serviceInfo struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Status             string   `json:"status"`
	ErrorMessage       *string  `json:"errorMessage"`
	FQDN               *string  `json:"fqdn"`
	InternalURL        string   `json:"internalUrl"`
	Repo               string   `json:"repo"`
	Branch             string   `json:"branch"`
	CommitHash         *string  `json:"commitHash"`
	GitProvider        string   `json:"gitProvider"`
	Memory             string   `json:"memory"`
	VCPUs              string   `json:"vcpus"`
	Port               string   `json:"port"`
	CustomDomain       *string  `json:"customDomain"`
	CustomDomainStatus *string  `json:"customDomainStatus"`
	EnvVars            []envVar `json:"envVars"`
	CreatedAt          string   `json:"createdAt"`
	UpdatedAt          string   `json:"updatedAt"`
}

type envVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

var servicesCmd = &cobra.Command{
	Use:     "services",
	Aliases: []string{"ls"},
	Short:   "List services",
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()

		var result struct {
			ServiceList struct {
				Nodes      []serviceInfo `json:"nodes"`
				TotalCount int           `json:"totalCount"`
			} `json:"serviceList"`
		}

		err := client.Do(`query($ws: String) {
			serviceList(workspaceSlug: $ws) {
				nodes { name status fqdn memory vcpus }
				totalCount
			}
		}`, defaultVars(), &result)
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(result.ServiceList)
			return
		}

		nodes := result.ServiceList.Nodes
		if len(nodes) == 0 {
			fmt.Println(dim.Render("  No services"))
			return
		}

		var rows [][]string
		for _, s := range nodes {
			url := dim.Render("—")
			if s.FQDN != nil {
				url = *s.FQDN
			}
			rows = append(rows, []string{s.Name, renderStatus(s.Status), url, s.Memory})
		}

		fmt.Println()
		fmt.Println(styledTable([]string{"NAME", "STATUS", "URL", "MEMORY"}, rows))
		tableFooter(len(nodes), "service")
		fmt.Println()
	},
}

func findService(client *api.Client, name string) (*serviceInfo, error) {
	var result struct {
		ServiceList struct {
			Nodes []serviceInfo `json:"nodes"`
		} `json:"serviceList"`
	}

	err := client.Do(`query($ws: String) {
		serviceList(workspaceSlug: $ws) {
			nodes {
				id name status errorMessage fqdn internalUrl
				repo branch commitHash gitProvider memory vcpus port
				customDomain customDomainStatus
				envVars { key value }
			}
		}
	}`, defaultVars(), &result)
	if err != nil {
		return nil, err
	}

	for i := range result.ServiceList.Nodes {
		if result.ServiceList.Nodes[i].Name == name {
			return &result.ServiceList.Nodes[i], nil
		}
	}
	return nil, nil
}
