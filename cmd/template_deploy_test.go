package cmd

import (
	"testing"

	"github.com/mldotink/cli/internal/config"
	ink "github.com/mldotink/sdk-go"
)

func TestNewTemplateDeployInputIncludesRegion(t *testing.T) {
	restoreConfig(t, &config.Resolved{
		Workspace: "team-local",
		Project:   "demo",
	})

	input := newTemplateDeployInput("postgres", "pg-us", "us-east-1", []ink.TemplateVariableValue{
		{Key: "database_name", Value: "app"},
	})

	if len(input.Regions) != 1 || input.Regions[0] != "us-east-1" {
		t.Fatalf("regions = %#v, want [us-east-1]", input.Regions)
	}
	if input.WorkspaceSlug != "team-local" {
		t.Fatalf("workspaceSlug = %q, want %q", input.WorkspaceSlug, "team-local")
	}
	if input.Project != "demo" {
		t.Fatalf("project = %q, want %q", input.Project, "demo")
	}
	if len(input.Variables) != 1 || input.Variables[0].Key != "database_name" || input.Variables[0].Value != "app" {
		t.Fatalf("variables = %#v, want database_name=app", input.Variables)
	}
}
