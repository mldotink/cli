package cmd

import (
	"strings"
	"testing"

	ink "github.com/mldotink/sdk-go"
)

func TestInkServicePortsPreservesAuthPolicy(t *testing.T) {
	ports := inkServicePorts([]ink.ServicePort{{
		Name:             "http",
		Port:             "3000",
		Protocol:         "http",
		Visibility:       "public",
		AuthPolicy:       "org_sso",
		InternalEndpoint: "http://svc:3000",
		PublicEndpoint:   "https://svc.example.com",
	}})

	if len(ports) != 1 {
		t.Fatalf("ports len = %d, want 1", len(ports))
	}
	if ports[0].authPolicy != "org_sso" {
		t.Fatalf("authPolicy = %q, want org_sso", ports[0].authPolicy)
	}
}

func TestRenderPortSummaryIncludesHTTPAuthPolicy(t *testing.T) {
	summary := renderPortSummary(endpointPort{
		name:             "http",
		port:             "3000",
		protocol:         "http",
		visibility:       "public",
		authPolicy:       "deployer_sso",
		internalEndpoint: "http://svc:3000",
		publicEndpoint:   "https://svc.example.com",
	})

	if !strings.Contains(summary, "auth=deployer_sso") {
		t.Fatalf("summary = %q, want auth policy", summary)
	}
}
