package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const githubRepo = "mldotink/cli"

type ghRelease struct {
	TagName string `json:"tag_name"`
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for updates and show how to upgrade",
	Run: func(cmd *cobra.Command, args []string) {
		current := strings.TrimPrefix(Version, "v")

		// Fetch latest release
		if !jsonOutput {
			fmt.Println(dim.Render("  Checking for updates..."))
		}

		rel, err := fetchLatestRelease()
		if err != nil {
			fatal("Failed to check for updates: " + err.Error())
		}
		latest := strings.TrimPrefix(rel.TagName, "v")

		// Detect install method
		method := detectInstallMethod()

		if jsonOutput {
			printJSON(map[string]any{
				"current":       current,
				"latest":        latest,
				"up_to_date":    current == latest,
				"install_method": method,
			})
			return
		}

		if current == latest {
			success(fmt.Sprintf("Already up to date (%s)", Version))
			return
		}

		if Version == "dev" {
			fmt.Println(yellow.Render("  ⚠ ") + "Running a dev build. Latest release is " + bold.Render("v"+latest))
		} else {
			fmt.Printf("  Update available: %s → %s\n", dim.Render("v"+current), bold.Render("v"+latest))
		}

		fmt.Println()
		switch method {
		case "homebrew":
			fmt.Println("  Run: " + bold.Render("brew upgrade ink"))
		case "npm":
			fmt.Println("  Run: " + bold.Render("npm update -g @mldotink/ink-cli"))
		default:
			fmt.Println("  Install the latest version:")
			fmt.Println("    " + accent.Render("brew install mldotink/tap/ink"))
			fmt.Println("    " + dim.Render("or"))
			fmt.Println("    " + accent.Render("npx @mldotink/ink-cli@latest"))
			fmt.Println("    " + dim.Render("or download from"))
			fmt.Println("    " + accent.Render(fmt.Sprintf("https://github.com/%s/releases/tag/v%s", githubRepo, latest)))
		}
		fmt.Println()
	},
}

func fetchLatestRelease() (*ghRelease, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRepo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ink-cli/"+Version)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

func detectInstallMethod() string {
	exe, err := os.Executable()
	if err != nil {
		return "unknown"
	}
	exe, _ = filepath.EvalSymlinks(exe)

	if strings.Contains(exe, "/Cellar/") || strings.Contains(exe, "/homebrew/") {
		return "homebrew"
	}
	if strings.Contains(exe, "/node_modules/") || strings.Contains(exe, `\node_modules\`) {
		return "npm"
	}
	return "binary"
}
