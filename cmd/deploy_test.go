package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mldotink/cli/internal/config"
	ink "github.com/mldotink/sdk-go"
	"github.com/spf13/cobra"
)

type capturedRequest struct {
	OpName    string
	Variables map[string]json.RawMessage
}

func newTestServer(t *testing.T, captured *capturedRequest, responseBody string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			OperationName string                     `json:"operationName"`
			Variables     map[string]json.RawMessage `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		captured.OpName = req.OperationName
		captured.Variables = req.Variables
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(responseBody))
	}))
}

func newTestClient(t *testing.T, serverURL string) *ink.Client {
	t.Helper()
	return ink.NewClient(ink.Config{APIKey: "test-key", BaseURL: serverURL})
}

func TestRunCreateIncludesResolvedWorkspace(t *testing.T) {
	restoreConfig(t, &config.Resolved{Workspace: "team-local"})

	var captured capturedRequest
	srv := newTestServer(t, &captured, `{"data":{"serviceCreate":{"serviceId":"svc_1","name":"my-app","status":"queued","repo":"my-app","ports":[]}}}`)
	defer srv.Close()

	cmd := newDeployCommandForTests()
	runCreate(cmd, newTestClient(t, srv.URL), "my-app")

	if captured.OpName != "createService" {
		t.Fatalf("operationName = %q, want %q", captured.OpName, "createService")
	}

	var input struct {
		WorkspaceSlug string `json:"workspaceSlug"`
	}
	raw, _ := json.Marshal(captured.Variables["input"])
	json.Unmarshal(raw, &input)

	if input.WorkspaceSlug != "team-local" {
		t.Fatalf("workspaceSlug = %q, want %q", input.WorkspaceSlug, "team-local")
	}
}

func TestRunUpdateIncludesResolvedWorkspace(t *testing.T) {
	restoreConfig(t, &config.Resolved{Workspace: "team-local"})

	var captured capturedRequest
	srv := newTestServer(t, &captured, `{"data":{"serviceUpdate":{"serviceId":"svc_1","name":"my-app","status":"queued"}}}`)
	defer srv.Close()

	cmd := newDeployCommandForTests()
	runUpdate(cmd, newTestClient(t, srv.URL), "my-app")

	if captured.OpName != "updateService" {
		t.Fatalf("operationName = %q, want %q", captured.OpName, "updateService")
	}

	var input struct {
		WorkspaceSlug string `json:"workspaceSlug"`
	}
	raw, _ := json.Marshal(captured.Variables["input"])
	json.Unmarshal(raw, &input)

	if input.WorkspaceSlug != "team-local" {
		t.Fatalf("workspaceSlug = %q, want %q", input.WorkspaceSlug, "team-local")
	}
}

func TestRunCreateMapsPortFlagToPublicHTTPPort(t *testing.T) {
	restoreConfig(t, &config.Resolved{Workspace: "team-local"})

	var captured capturedRequest
	srv := newTestServer(t, &captured, `{"data":{"serviceCreate":{"serviceId":"svc_1","name":"my-app","status":"queued","repo":"my-app","ports":[]}}}`)
	defer srv.Close()

	cmd := newDeployCommandForTests()
	if err := cmd.Flags().Set("port", "8080"); err != nil {
		t.Fatalf("set port flag: %v", err)
	}

	runCreate(cmd, newTestClient(t, srv.URL), "my-app")

	var input struct {
		Ports []struct {
			Name       string `json:"name"`
			Port       int    `json:"port"`
			Protocol   string `json:"protocol"`
			Visibility string `json:"visibility"`
		} `json:"ports"`
	}
	raw, _ := json.Marshal(captured.Variables["input"])
	json.Unmarshal(raw, &input)

	if len(input.Ports) != 1 {
		t.Fatalf("ports len = %d, want 1", len(input.Ports))
	}
	port := input.Ports[0]
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
