package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const githubRepo = "mldotink/cli"

const (
	homebrewFormula = "mldotink/tap/ink"
	npmPackage      = "@mldotink/cli"
)

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
				"current":        current,
				"latest":         latest,
				"up_to_date":     current == latest,
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
			fmt.Println()
			for _, line := range updateInstructionLines(method, latest) {
				fmt.Println(line)
			}
			fmt.Println()
			return
		}

		fmt.Printf("  Update available: %s → %s\n", dim.Render("v"+current), bold.Render("v"+latest))

		updateCmd, updateArgs := updateCommand(method, latest)
		if updateCmd == "" {
			fmt.Println()
			for _, line := range updateInstructionLines(method, latest) {
				fmt.Println(line)
			}
			fmt.Println()
			return
		}

		fmt.Printf("  Running: %s %s\n", updateCmd, strings.Join(updateArgs, " "))
		c := exec.Command(updateCmd, updateArgs...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			fatal("Update failed: " + err.Error())
		}
		success(fmt.Sprintf("Updated to v%s", latest))
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

	resolved := exe
	if eval, err := filepath.EvalSymlinks(exe); err == nil {
		resolved = eval
	}

	return detectInstallMethodFromPaths(exe, resolved)
}

func updateCommand(method, latest string) (string, []string) {
	switch method {
	case "homebrew":
		return "brew", []string{"upgrade", homebrewFormula}
	case "npm":
		return "npm", []string{"install", "-g", fmt.Sprintf("%s@%s", npmPackage, latest)}
	default:
		return "", nil
	}
}

func updateInstructionLines(method, latest string) []string {
	switch method {
	case "homebrew":
		return []string{
			"  Run: " + bold.Render("brew upgrade "+homebrewFormula),
		}
	case "npm":
		return []string{
			"  Run: " + bold.Render(fmt.Sprintf("npm install -g %s@%s", npmPackage, latest)),
		}
	default:
		return []string{
			"  Install the latest version:",
			"    " + accent.Render("brew install "+homebrewFormula),
			"    " + dim.Render("or"),
			"    " + accent.Render("npx "+npmPackage+"@latest"),
			"    " + dim.Render("or download from"),
			"    " + accent.Render(fmt.Sprintf("https://github.com/%s/releases/tag/v%s", githubRepo, latest)),
		}
	}
}

func detectInstallMethodFromPaths(paths ...string) string {
	for _, path := range paths {
		if strings.Contains(path, "/node_modules/") || strings.Contains(path, `\node_modules\`) {
			return "npm"
		}
	}
	for _, path := range paths {
		if strings.Contains(path, "/Cellar/") {
			return "homebrew"
		}
	}
	return "binary"
}
