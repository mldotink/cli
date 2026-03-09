package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/mldotink/ink-cli/internal/config"
	"github.com/spf13/cobra"
)

var loginGlobal bool

func init() {
	loginCmd.Flags().BoolVar(&loginGlobal, "global", true, "Save globally (~/.config/ink/config)")
	rootCmd.AddCommand(loginCmd)
}

var loginCmd = &cobra.Command{
	Use:   "login [api-key]",
	Short: "Authenticate with Ink",
	Long:  "Store your API key. Get one at https://ml.ink/account/api-keys",
	Example: `# Interactive prompt
ink login

# Pass key directly
ink login dk_live_abc123`,
	Args: maxArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var key string
		if len(args) > 0 {
			key = strings.TrimSpace(args[0])
		} else {
			fmt.Print("Enter API key: ")
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				key = strings.TrimSpace(scanner.Text())
			}
		}

		if !strings.HasPrefix(key, "dk_") {
			fatal("Invalid API key — keys start with dk_live_ or dk_test_")
		}

		c := &config.Config{APIKey: key}
		var err error
		if loginGlobal {
			err = config.SaveGlobal(c)
		} else {
			err = config.SaveLocal(c)
		}
		if err != nil {
			fatal(fmt.Sprintf("Failed to save: %v", err))
		}

		if loginGlobal {
			success("Saved to ~/.config/ink/config")
		} else {
			success("Saved to .ink (project-local)")
		}
	},
}
