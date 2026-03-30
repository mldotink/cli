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

	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configShowCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Set default workspace and project so you don't need --workspace and --project on every command",
	Example: `# Set your default workspace and project (recommended)
ink config set --workspace my-team --project backend

# Per-repo override via local .ink file
ink config set --workspace my-team --local

# View current config
ink config show`,
}

var configSetCmd = &cobra.Command{
	Use:   "set [--workspace <slug>] [--project <slug>]",
	Short: "Set default workspace and/or project",
	Long:  "Set workspace and/or project defaults. Saves to global config by default, use --local for project-scoped.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		ws, _ := cmd.Flags().GetString("workspace")
		proj, _ := cmd.Flags().GetString("project")
		local, _ := cmd.Flags().GetBool("local")

		if ws == "" && proj == "" {
			fatal("Provide at least one of --workspace or --project")
		}

		c := &config.Config{
			Workspace: ws,
			Project:   proj,
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
