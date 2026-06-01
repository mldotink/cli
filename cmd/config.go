package cmd

import (
	"fmt"

	"github.com/mldotink/cli/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	configSetCmd.Flags().Bool("global", true, "Save to global config (~/.config/ink/config)")
	configSetCmd.Flags().Bool("local", false, "Save to local config (.ink)")
	configSetCmd.Flags().StringP("workspace", "w", "", "Default workspace slug")
	configSetCmd.Flags().StringP("project", "p", "", "Default project slug")
	configSetCmd.Flags().String("api-url", "", "GraphQL API URL")
	configSetCmd.Flags().String("oauth-url", "", "OAuth server URL")
	configSetCmd.Flags().String("web-url", "", "Ink web app URL")

	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configShowCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Set default workspace and project so you don't need --workspace and --project on every command",
	Example: `# Set your default workspace and project (recommended)
ink config set --workspace my-team --project backend

# Point the CLI at an enterprise Ink host
ink config set --api-url https://api.example.com/graphql --oauth-url https://mcp.example.com --web-url https://ink.example.com

# Per-repo override via local .ink file
ink config set --workspace my-team --local

# View current config
ink config show`,
}

var configSetCmd = &cobra.Command{
	Use:   "set [--workspace <slug>] [--project <slug>] [--api-url <url>] [--oauth-url <url>] [--web-url <url>]",
	Short: "Set default workspace, project, and host URLs",
	Long:  "Set workspace, project, and host URL defaults. Saves to global config by default, use --local for project-scoped.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		ws, _ := cmd.Flags().GetString("workspace")
		proj, _ := cmd.Flags().GetString("project")
		apiURL, _ := cmd.Flags().GetString("api-url")
		oauthURL, _ := cmd.Flags().GetString("oauth-url")
		webURL, _ := cmd.Flags().GetString("web-url")
		local, _ := cmd.Flags().GetBool("local")

		if ws == "" && proj == "" && apiURL == "" && oauthURL == "" && webURL == "" {
			fatal("Provide at least one of --workspace, --project, --api-url, --oauth-url, or --web-url")
		}

		c := &config.Config{
			Workspace: ws,
			Project:   proj,
			APIURL:    apiURL,
			OAuthURL:  oauthURL,
			WebURL:    webURL,
		}

		var err error
		if local {
			err = config.SaveLocal(c)
		} else {
			err = config.SaveGlobal(c)
		}
		if err != nil {
			fatal(fmt.Sprintf("Failed to save: %v", err))
		}

		target := "~/.config/ink/config"
		if local {
			target = ".ink"
		}
		if ws != "" {
			success(fmt.Sprintf("Set workspace=%s in %s", bold.Render(ws), target))
		}
		if proj != "" {
			success(fmt.Sprintf("Set project=%s in %s", bold.Render(proj), target))
		}
		if apiURL != "" {
			success(fmt.Sprintf("Set api_url=%s in %s", bold.Render(apiURL), target))
		}
		if oauthURL != "" {
			success(fmt.Sprintf("Set oauth_url=%s in %s", bold.Render(oauthURL), target))
		}
		if webURL != "" {
			success(fmt.Sprintf("Set web_url=%s in %s", bold.Render(webURL), target))
		}
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current config",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println()
		kv("API Key", maskKey(cfg.APIKey)+" "+dim.Render("("+sourceLabel(cfg.Sources["api_key"])+")"))
		if cfg.Workspace != "" {
			kv("Workspace", cfg.Workspace+" "+dim.Render("("+sourceLabel(cfg.Sources["workspace"])+")"))
		}
		if cfg.Project != "" {
			kv("Project", cfg.Project+" "+dim.Render("("+sourceLabel(cfg.Sources["project"])+")"))
		}
		if cfg.APIURL != "" {
			kv("API URL", cfg.APIURL+" "+dim.Render("("+sourceLabel(cfg.Sources["api_url"])+")"))
		}
		if cfg.OAuthURL != "" {
			kv("OAuth URL", cfg.OAuthURL+" "+dim.Render("("+sourceLabel(cfg.Sources["oauth_url"])+")"))
		}
		if cfg.WebURL != "" {
			kv("Web URL", cfg.WebURL+" "+dim.Render("("+sourceLabel(cfg.Sources["web_url"])+")"))
		}
		fmt.Println()
	},
}

func maskKey(key string) string {
	if key == "" {
		return dim.Render("(not set)")
	}
	if len(key) <= 12 {
		return key[:4] + "****"
	}
	return key[:8] + "…" + key[len(key)-4:]
}
