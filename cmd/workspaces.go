package cmd

import (
	"fmt"

	ink "github.com/mldotink/sdk-go"
	"github.com/spf13/cobra"
)

func init() {
	workspacesCreateCmd.Flags().String("description", "", "Workspace description")
	workspacesCmd.AddCommand(workspacesCreateCmd)
	workspacesCmd.AddCommand(workspacesDeleteCmd)
	workspacesCmd.AddCommand(workspacesMembersCmd)
	workspacesCmd.AddCommand(workspacesInviteCmd)
	workspacesCmd.AddCommand(workspacesInvitesCmd)
	workspacesCmd.AddCommand(workspacesAcceptCmd)
	workspacesCmd.AddCommand(workspacesDeclineCmd)
	workspacesCmd.AddCommand(workspacesRevokeCmd)
	workspacesCmd.AddCommand(workspacesRemoveCmd)
}

var workspacesCmd = &cobra.Command{
	Use:     "workspace",
	Aliases: []string{"workspaces", "ws"},
	Short:   "Create and manage team workspaces, members, and invites",
	Example: `# List workspaces
ink workspace

# Create a workspace
ink workspace create "My Team" my-team

# Invite a member
ink workspace invite my-team user@example.com admin

# List members
ink workspace members my-team`,
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()

		wsList, err := client.ListWorkspaces(ctx())
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(wsList)
			return
		}

		if len(wsList) == 0 {
			fmt.Println(dim.Render("  No workspaces"))
		} else {
			var rows [][]string
			for _, ws := range wsList {
				name := ws.Name
				if ws.IsDefault {
					name += dim.Render(" (default)")
				}
				rows = append(rows, []string{name, ws.Slug, ws.Role})
			}

			fmt.Println()
			fmt.Println(styledTable([]string{"NAME", "SLUG", "ROLE"}, rows))
			tableFooter(len(wsList), "workspace")
		}

		printSubcommands(cmd)
	},
}

var workspacesCreateCmd = &cobra.Command{
	Use:   "create <name> <slug>",
	Short: "Create a workspace",
	Args:  exactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		name, slug := args[0], args[1]
		desc, _ := cmd.Flags().GetString("description")
		client := newClient()

		ws, err := client.CreateWorkspace(ctx(), name, slug, desc)
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(ws)
			return
		}

		success(fmt.Sprintf("Workspace created: %s (%s)", bold.Render(ws.Name), ws.Slug))
	},
}

var workspacesDeleteCmd = &cobra.Command{
	Use:   "delete <slug>",
	Short: "Delete a workspace",
	Args:  exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		slug := args[0]
		client := newClient()

		id, err := resolveWorkspaceID(client, slug)
		if err != nil {
			fatal(err.Error())
		}

		if err := client.DeleteWorkspace(ctx(), id); err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(map[string]any{"deleted": true, "slug": slug})
			return
		}

		success(fmt.Sprintf("Workspace %s deleted", bold.Render(slug)))
	},
}

var workspacesMembersCmd = &cobra.Command{
	Use:   "members <slug>",
	Short: "List workspace members",
	Args:  exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		slug := args[0]
		client := newClient()

		members, err := client.ListWorkspaceMembers(ctx(), slug)
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(members)
			return
		}

		if len(members) == 0 {
			fmt.Println(dim.Render("  No members"))
			return
		}

		var rows [][]string
		for _, m := range members {
			name := m.DisplayName
			if name == "" {
				name = m.Username
			}
			if name == "" {
				name = m.UserID
			}
			email := m.Email
			if email == "" {
				email = dim.Render("—")
			}
			rows = append(rows, []string{name, email, m.Role})
		}

		fmt.Println()
		fmt.Println(styledTable([]string{"USER", "EMAIL", "ROLE"}, rows))
		tableFooter(len(members), "member")
		fmt.Println()
	},
}

var workspacesInviteCmd = &cobra.Command{
	Use:   "invite <slug> <user> [role]",
	Short: "Invite a user to a workspace",
	Args:  rangeArgs(2, 3),
	Run: func(cmd *cobra.Command, args []string) {
		slug, user := args[0], args[1]
		role := "member"
		if len(args) > 2 {
			role = args[2]
		}

		client := newClient()
		id, err := resolveWorkspaceID(client, slug)
		if err != nil {
			fatal(err.Error())
		}

		invite, err := client.InviteToWorkspace(ctx(), id, user, role)
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(invite)
			return
		}

		success(fmt.Sprintf("Invited %s to %s as %s", bold.Render(user), bold.Render(slug), role))
	},
}

var workspacesInvitesCmd = &cobra.Command{
	Use:   "invites [slug]",
	Short: "List pending invites (yours if no slug, workspace's if slug given)",
	Args:  maxArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()

		if len(args) == 0 {
			invites, err := client.ListMyInvites(ctx())
			if err != nil {
				fatal(err.Error())
			}
			if jsonOutput {
				printJSON(invites)
				return
			}
			if len(invites) == 0 {
				fmt.Println(dim.Render("  No pending invites"))
				return
			}

			var rows [][]string
			for _, inv := range invites {
				rows = append(rows, []string{inv.ID, inv.WorkspaceName, inv.Role, renderStatus(inv.Status)})
			}

			fmt.Println()
			fmt.Println(styledTable([]string{"ID", "WORKSPACE", "ROLE", "STATUS"}, rows))
			fmt.Println()
		} else {
			slug := args[0]
			invites, err := client.ListWorkspaceInvites(ctx(), slug)
			if err != nil {
				fatal(err.Error())
			}
			if jsonOutput {
				printJSON(invites)
				return
			}
			if len(invites) == 0 {
				fmt.Println(dim.Render("  No pending invites"))
				return
			}

			var rows [][]string
			for _, inv := range invites {
				rows = append(rows, []string{inv.ID, inv.Role, renderStatus(inv.Status)})
			}

			fmt.Println()
			fmt.Println(styledTable([]string{"ID", "ROLE", "STATUS"}, rows))
			fmt.Println()
		}
	},
}

var workspacesAcceptCmd = &cobra.Command{
	Use:   "accept-invite <invite-id>",
	Short: "Accept a workspace invite",
	Args:  exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()
		if err := client.AcceptInvite(ctx(), args[0]); err != nil {
			fatal(err.Error())
		}
		if jsonOutput {
			printJSON(map[string]any{"accepted": true})
			return
		}
		success("Invite accepted")
	},
}

var workspacesDeclineCmd = &cobra.Command{
	Use:   "decline-invite <invite-id>",
	Short: "Decline a workspace invite",
	Args:  exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()
		if err := client.DeclineInvite(ctx(), args[0]); err != nil {
			fatal(err.Error())
		}
		if jsonOutput {
			printJSON(map[string]any{"declined": true})
			return
		}
		success("Invite declined")
	},
}

var workspacesRevokeCmd = &cobra.Command{
	Use:   "revoke-invite <invite-id>",
	Short: "Revoke a pending invite",
	Args:  exactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()
		if err := client.RevokeInvite(ctx(), args[0]); err != nil {
			fatal(err.Error())
		}
		if jsonOutput {
			printJSON(map[string]any{"revoked": true})
			return
		}
		success("Invite revoked")
	},
}

var workspacesRemoveCmd = &cobra.Command{
	Use:   "remove-member <slug> <user-id>",
	Short: "Remove a member from a workspace",
	Args:  exactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		slug, userID := args[0], args[1]
		client := newClient()

		wsID, err := resolveWorkspaceID(client, slug)
		if err != nil {
			fatal(err.Error())
		}

		if err := client.RemoveWorkspaceMember(ctx(), wsID, userID); err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(map[string]any{"removed": true})
			return
		}
		success("Member removed")
	},
}

func resolveWorkspaceID(client *ink.Client, slug string) (string, error) {
	wsList, err := client.ListWorkspaces(ctx())
	if err != nil {
		return "", err
	}
	for _, ws := range wsList {
		if ws.Slug == slug {
			return ws.ID, nil
		}
	}
	return "", fmt.Errorf("workspace %q not found", slug)
}
