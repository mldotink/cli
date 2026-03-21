package cmd

import (
	"strings"
	"testing"
)

func TestDetectInstallMethodFromPaths(t *testing.T) {
	tests := []struct {
		name     string
		exe      string
		resolved string
		want     string
	}{
		{
			name:     "npm install in homebrew prefix",
			exe:      "/opt/homebrew/bin/ink",
			resolved: "/opt/homebrew/lib/node_modules/@mldotink/cli/node_modules/@mldotink/cli-darwin-arm64/bin/ink",
			want:     "npm",
		},
		{
			name:     "npm install inside node cellar",
			exe:      "/opt/homebrew/Cellar/node/25.6.1/libexec/lib/node_modules/@mldotink/cli/node_modules/@mldotink/cli-darwin-arm64/bin/ink",
			resolved: "/opt/homebrew/Cellar/node/25.6.1/libexec/lib/node_modules/@mldotink/cli/node_modules/@mldotink/cli-darwin-arm64/bin/ink",
			want:     "npm",
		},
		{
			name:     "homebrew formula",
			exe:      "/opt/homebrew/bin/ink",
			resolved: "/opt/homebrew/Cellar/ink/0.1.32/bin/ink",
			want:     "homebrew",
		},
		{
			name:     "standalone binary",
			exe:      "/usr/local/bin/ink",
			resolved: "/usr/local/bin/ink",
			want:     "binary",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := detectInstallMethodFromPaths(tc.exe, tc.resolved); got != tc.want {
				t.Fatalf("detectInstallMethodFromPaths(%q, %q) = %q, want %q", tc.exe, tc.resolved, got, tc.want)
			}
		})
	}
}

func TestUpdateInstructionLines(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		latest   string
		contains []string
	}{
		{
			name:   "homebrew",
			method: "homebrew",
			latest: "0.1.32",
			contains: []string{
				"brew upgrade mldotink/tap/ink",
			},
		},
		{
			name:   "npm",
			method: "npm",
			latest: "0.1.32",
			contains: []string{
				"npm install -g @mldotink/cli@0.1.32",
			},
		},
		{
			name:   "binary",
			method: "binary",
			latest: "0.1.32",
			contains: []string{
				"brew install mldotink/tap/ink",
				"npx @mldotink/cli@latest",
				"https://github.com/mldotink/cli/releases/tag/v0.1.32",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lines := updateInstructionLines(tc.method, tc.latest)
			joined := strings.Join(lines, "\n")
			for _, want := range tc.contains {
				if !strings.Contains(joined, want) {
					t.Fatalf("updateInstructionLines(%q, %q) missing %q in %q", tc.method, tc.latest, want, joined)
				}
			}
		})
	}
}
