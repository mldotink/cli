package cmd

import (
	"fmt"

	"github.com/mldotink/cli/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	configSetCmd.Flags().Bool("global", true, "Save to global config (~/.config/ink/config)")
	configSetCmd.Flags().Bool("local", false, "Save to local config (.ink)")

	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configShowCmd)
}

var configCmd = &cobra.Command{
	Use:     "config",
	Short:   "Manage CLI configuration",
	Example: `ink config set workspace my-team
ink config set project backend
ink config set workspace my-team --local
ink config show`,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Long:  "Set workspace or project defaults. Saves to global config by default, use --local for project-scoped.",
	Args:  exactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key, value := args[0], args[1]
		local, _ := cmd.Flags().GetBool("local")

		c := &config.Config{}
		switch key {
		case "workspace":
			c.Workspace = value
		case "project":
			c.Project = value
		default:
			fatal(fmt.Sprintf("Unknown config key %q — use 'workspace' or 'project'", key))
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
		success(fmt.Sprintf("Set %s=%s in %s", key, bold.Render(value), target))
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
