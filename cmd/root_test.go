package cmd

import (
	"strings"
	"testing"

	ink "github.com/mldotink/sdk-go"
	"github.com/spf13/cobra"
)

func TestLooksLikeSchemaMismatch(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		msg  string
		want bool
	}{
		{
			name: "graphql validation failed",
			msg:  `returned error 422: {"errors":[{"extensions":{"code":"GRAPHQL_VALIDATION_FAILED"},"message":"Cannot query field \"fqdn\" on type \"Service\"."}]}`,
			want: true,
		},
		{
			name: "unknown input field",
			msg:  `graphql: Unknown input field "port" on CreateServiceInput`,
			want: true,
		},
		{
			name: "ordinary not found",
			msg:  `serviceDelete service not found: debug1-site in project research-preview-august`,
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := looksLikeSchemaMismatch(tc.msg); got != tc.want {
				t.Fatalf("looksLikeSchemaMismatch(%q) = %v, want %v", tc.msg, got, tc.want)
			}
		})
	}
}

func TestUpgradeHintLinesForSchemaMismatch(t *testing.T) {
	t.Parallel()

	lines := upgradeHintLines(`Cannot query field "fqdn" on type "Service".`)
	if len(lines) == 0 {
		t.Fatal("expected upgrade hint lines")
	}
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "ink update") && !strings.Contains(joined, "brew") && !strings.Contains(joined, "npm") {
		t.Fatalf("expected update instructions in %q", joined)
	}
}

func TestValidateProjectSelection(t *testing.T) {
	t.Parallel()

	projects := []ink.Project{
		{Slug: "default"},
		{Slug: "backend"},
	}

	if err := validateProjectSelection("team", "backend", projects, nil); err != nil {
		t.Fatalf("validateProjectSelection() returned error: %v", err)
	}
	if err := validateProjectSelection("team", "", projects, nil); err != nil {
		t.Fatalf("validateProjectSelection() returned error for default project: %v", err)
	}

	err := validateProjectSelection("team", "missing", projects, nil)
	if err == nil {
		t.Fatal("validateProjectSelection() unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), `configured project "missing" was not found`) {
		t.Fatalf("validateProjectSelection() error = %v", err)
	}
	if !strings.Contains(err.Error(), "available projects: backend, default") {
		t.Fatalf("validateProjectSelection() error missing project hint: %v", err)
	}
}

func TestShouldValidateContext(t *testing.T) {
	t.Parallel()

	root := &cobra.Command{Use: "ink"}
	deploy := &cobra.Command{Use: "deploy"}
	root.AddCommand(deploy)
	if !shouldValidateContext(deploy) {
		t.Fatal("deploy command should validate context")
	}
	deleteCmd := &cobra.Command{Use: "delete"}
	root.AddCommand(deleteCmd)
	if !shouldValidateContext(deleteCmd) {
		t.Fatal("root delete command should validate context")
	}
	metrics := &cobra.Command{Use: "metrics"}
	root.AddCommand(metrics)
	if !shouldValidateContext(metrics) {
		t.Fatal("metrics command should validate context")
	}

	projects := &cobra.Command{Use: "project"}
	projectDelete := &cobra.Command{Use: "delete"}
	projects.AddCommand(projectDelete)
	root.AddCommand(projects)
	if shouldValidateContext(projects) {
		t.Fatal("project command should not validate context")
	}
	if shouldValidateContext(projectDelete) {
		t.Fatal("project delete command should not validate project context")
	}

	services := &cobra.Command{Use: "service"}
	services.Flags().Bool("all", false, "")
	root.AddCommand(services)
	if !shouldValidateContext(services) {
		t.Fatal("service command should validate context")
	}
	if err := services.Flags().Set("all", "true"); err != nil {
		t.Fatal(err)
	}
	if shouldValidateContext(services) {
		t.Fatal("service --all should not validate configured context")
	}

	templates := &cobra.Command{Use: "template"}
	templateInfo := &cobra.Command{Use: "info"}
	templateDeploy := &cobra.Command{Use: "deploy"}
	templates.AddCommand(templateInfo, templateDeploy)
	root.AddCommand(templates)
	if shouldValidateContext(templateInfo) {
		t.Fatal("template info should not validate project context")
	}
	if !shouldValidateContext(templateDeploy) {
		t.Fatal("template deploy should validate context")
	}

	repos := &cobra.Command{Use: "repo"}
	repoCreate := &cobra.Command{Use: "create"}
	repoToken := &cobra.Command{Use: "token"}
	repos.AddCommand(repoCreate, repoToken)
	root.AddCommand(repos)
	if !shouldValidateContext(repoCreate) {
		t.Fatal("repo create should validate context")
	}
	if shouldValidateContext(repoToken) {
		t.Fatal("repo token should not validate project context")
	}
}
