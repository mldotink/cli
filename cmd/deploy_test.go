package cmd

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/mldotink/cli/internal/config"
	"github.com/mldotink/cli/internal/gql"
	"github.com/spf13/cobra"
)

func TestRunCreateIncludesResolvedWorkspace(t *testing.T) {
	restoreConfig(t, &config.Resolved{Workspace: "team-local"})

	cmd := newDeployCommandForTests()
	client := &captureGraphQLClient{t: t}

	runCreate(cmd, client, "my-app")

	if client.opName != "CreateService" {
		t.Fatalf("operation = %q, want %q", client.opName, "CreateService")
	}
	if client.workspace == nil || *client.workspace != "team-local" {
		t.Fatalf("workspace = %v, want %q", client.workspace, "team-local")
	}
}

func TestRunUpdateIncludesResolvedWorkspace(t *testing.T) {
	restoreConfig(t, &config.Resolved{Workspace: "team-local"})

	cmd := newDeployCommandForTests()
	client := &captureGraphQLClient{t: t}

	runUpdate(cmd, client, "my-app")

	if client.opName != "UpdateService" {
		t.Fatalf("operation = %q, want %q", client.opName, "UpdateService")
	}
	if client.workspace == nil || *client.workspace != "team-local" {
		t.Fatalf("workspace = %v, want %q", client.workspace, "team-local")
	}
}

type captureGraphQLClient struct {
	t         *testing.T
	opName    string
	workspace *string
	ports     []gql.ServicePortInput
}

func (c *captureGraphQLClient) MakeRequest(_ context.Context, req *graphql.Request, resp *graphql.Response) error {
	c.opName = req.OpName

	var payload struct {
		Input struct {
			Name          string                 `json:"name"`
			WorkspaceSlug *string                `json:"workspaceSlug"`
			Ports         []gql.ServicePortInput `json:"ports"`
		} `json:"input"`
	}
	raw, err := json.Marshal(req.Variables)
	if err != nil {
		c.t.Fatalf("marshal variables: %v", err)
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		c.t.Fatalf("unmarshal variables: %v", err)
	}
	c.workspace = payload.Input.WorkspaceSlug
	c.ports = payload.Input.Ports

	switch req.OpName {
	case "CreateService":
		data, ok := resp.Data.(*gql.CreateServiceResponse)
		if !ok {
			c.t.Fatalf("unexpected create response type %T", resp.Data)
		}
		data.ServiceCreate = gql.CreateServiceServiceCreateCreateServiceResult{
			ServiceId: "svc_123",
			Name:      payload.Input.Name,
			Status:    "queued",
			Repo:      "my-app",
		}
	case "UpdateService":
		data, ok := resp.Data.(*gql.UpdateServiceResponse)
		if !ok {
			c.t.Fatalf("unexpected update response type %T", resp.Data)
		}
		data.ServiceUpdate = gql.UpdateServiceServiceUpdateUpdateServiceResult{
			ServiceId: "svc_123",
			Name:      payload.Input.Name,
			Status:    "queued",
		}
	default:
		c.t.Fatalf("unexpected operation %q", req.OpName)
	}

	return nil
}

func TestRunCreateMapsPortFlagToPublicHTTPPort(t *testing.T) {
	restoreConfig(t, &config.Resolved{Workspace: "team-local"})

	cmd := newDeployCommandForTests()
	if err := cmd.Flags().Set("port", "8080"); err != nil {
		t.Fatalf("set port flag: %v", err)
	}
	client := &captureGraphQLClient{t: t}

	runCreate(cmd, client, "my-app")

	if len(client.ports) != 1 {
		t.Fatalf("ports len = %d, want 1", len(client.ports))
	}
	port := client.ports[0]
	if port.Name != "http" || port.Port != 8080 || port.Protocol != "http" || port.Visibility != "public" {
		t.Fatalf("port = %#v, want public http 8080", port)
	}
}

func newDeployCommandForTests() *cobra.Command {
	cmd := &cobra.Command{Use: "deploy"}
	cmd.Flags().StringP("repo", "r", "", "")
	cmd.Flags().IntP("port", "p", 0, "")
	cmd.Flags().String("host", "ink", "")
	cmd.Flags().String("branch", "main", "")
	cmd.Flags().String("region", "eu-central-1", "")
	addServiceFlags(cmd)
	return cmd
}

func restoreConfig(t *testing.T, resolved *config.Resolved) {
	t.Helper()

	prevCfg := cfg
	cfg = resolved
	t.Cleanup(func() {
		cfg = prevCfg
	})
}
