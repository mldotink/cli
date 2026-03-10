package cmd

import (
	"fmt"

	"github.com/mldotink/cli/internal/gql"
	"github.com/spf13/cobra"
)

func init() {
	chatSendCmd.Flags().String("channel", "", "Project slug to scope to a project channel")
	chatReadCmd.Flags().String("channel", "", "Project slug to scope to a project channel")
	chatReadCmd.Flags().Int("cursor", 0, "Pagination cursor from previous read")
	chatReadCmd.Flags().Int("limit", 50, "Number of messages to fetch (max 100)")
	chatCmd.AddCommand(chatSendCmd)
	chatCmd.AddCommand(chatReadCmd)
}

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Send and read messages in workspace or project channels",
	Example: `ink chat send "hello team" -w my-team
ink chat read -w my-team
ink chat read -w my-team --channel backend`,
}

var chatSendCmd = &cobra.Command{
	Use:   "send <message>",
	Short: "Send a message to workspace chat",
	Example: `ink chat send "Deploy is done!"
ink chat send "Frontend update" --channel my-project`,
	Args: exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		content := args[0]
		channel, _ := cmd.Flags().GetString("channel")
		client := newClient()

		ws := cfg.Workspace
		if ws == "" {
			fatal("Workspace required — set in config or use --workspace")
		}

		result, err := gql.SendChatMessage(ctx(), client, ws, ptr(channel), content)
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(result.ChatSend)
			return
		}

		success(fmt.Sprintf("Message sent (seq: %d)", result.ChatSend.Seq))
	},
}

var chatReadCmd = &cobra.Command{
	Use:   "read",
	Short: "Read workspace chat messages",
	Run: func(cmd *cobra.Command, args []string) {
		channel, _ := cmd.Flags().GetString("channel")
		cursor, _ := cmd.Flags().GetInt("cursor")
		limit, _ := cmd.Flags().GetInt("limit")
		client := newClient()

		ws := cfg.Workspace
		if ws == "" {
			fatal("Workspace required — set in config or use --workspace")
		}

		var cursorPtr *int
		if cursor > 0 {
			cursorPtr = &cursor
		}

		result, err := gql.ReadChat(ctx(), client, ws, ptr(channel), cursorPtr, &limit)
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(result.ChatRead)
			return
		}

		msgs := result.ChatRead.Messages
		if len(msgs) == 0 {
			fmt.Println(dim.Render("  No messages"))
			return
		}

		fmt.Println()
		for _, m := range msgs {
			fmt.Printf("  %s  %s\n", bold.Render(m.SenderName), dim.Render(fmtTime(m.CreatedAt)))
			fmt.Printf("  %s\n\n", m.Content)
		}
		if result.ChatRead.HasMore {
			fmt.Println(dim.Render(fmt.Sprintf("  More messages available — use --cursor %d", result.ChatRead.NextCursor)))
		}
	},
}
