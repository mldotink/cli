package cmd

import (
	"strings"
	"testing"
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
