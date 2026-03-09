package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	chatSendCmd.Flags().String("channel", "", "Project slug to scope to a project channel")
	chatReadCmd.Flags().String("channel", "", "Project slug to scope to a project channel")
	chatReadCmd.Flags().Int("cursor", 0, "Pagination cursor from previous read")
	chatReadCmd.Flags().Int("limit", 50, "Number of messages to fetch (max 100)")
	chatCmd.AddCommand(chatSendCmd)
	chatCmd.AddCommand(chatReadCmd)
	rootCmd.AddCommand(chatCmd)
}

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Workspace chat",
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

		vars := map[string]any{
			"ws":      ws,
			"content": content,
		}
		if channel != "" {
			vars["channel"] = channel
		}

		var result struct {
			ChatSend struct {
				Seq       int    `json:"seq"`
				MessageID string `json:"messageId"`
			} `json:"chatSend"`
		}

		err := client.Do(`mutation($ws: String!, $channel: String, $content: String!) {
			chatSend(workspaceSlug: $ws, channel: $channel, content: $content) {
				seq messageId
			}
		}`, vars, &result)
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

		vars := map[string]any{
			"ws":    ws,
			"limit": limit,
		}
		if channel != "" {
			vars["channel"] = channel
		}
		if cursor > 0 {
			vars["cursor"] = cursor
		}

		var result struct {
			ChatRead struct {
				Messages []struct {
					Seq        int    `json:"seq"`
					SenderName string `json:"senderName"`
					Content    string `json:"content"`
					Channel    string `json:"channel"`
					CreatedAt  string `json:"createdAt"`
				} `json:"messages"`
				NextCursor int  `json:"nextCursor"`
				HasMore    bool `json:"hasMore"`
			} `json:"chatRead"`
		}

		err := client.Do(`query($ws: String!, $channel: String, $cursor: Int, $limit: Int) {
			chatRead(workspaceSlug: $ws, channel: $channel, cursor: $cursor, limit: $limit) {
				messages { seq senderName content channel createdAt }
				nextCursor hasMore
			}
		}`, vars, &result)
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
			fmt.Printf("  %s  %s\n", bold.Render(m.SenderName), dim.Render(m.CreatedAt))
			fmt.Printf("  %s\n\n", m.Content)
		}
		if result.ChatRead.HasMore {
			fmt.Println(dim.Render(fmt.Sprintf("  More messages available — use --cursor %d", result.ChatRead.NextCursor)))
		}
	},
}
