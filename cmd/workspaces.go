package cmd

import (
	"fmt"

	"github.com/Khan/genqlient/graphql"
	"github.com/mldotink/cli/internal/gql"
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
	rootCmd.AddCommand(workspacesCmd)
}

var workspacesCmd = &cobra.Command{
	Use:     "workspaces",
	Aliases: []string{"ws"},
	Short:   "Manage workspaces",
	Example: `# List workspaces
ink ws

# Create a workspace
ink ws create "My Team" my-team

# Invite a member
ink ws invite my-team user@example.com admin`,
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()

		result, err := gql.ListWorkspaces(ctx(), client)
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(result.WorkspaceList)
			return
		}

		wsList := result.WorkspaceList
		if len(wsList) == 0 {
			fmt.Println(dim.Render("  No workspaces"))
			return
		}

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
		fmt.Println()
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

		result, err := gql.CreateWorkspace(ctx(), client, name, slug, ptr(desc))
		if err != nil {
			fatal(err.Error())
		}

		ws := result.WorkspaceCreate
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

		result, err := gql.DeleteWorkspace(ctx(), client, id)
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(map[string]any{"deleted": result.WorkspaceDelete, "slug": slug})
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

		result, err := gql.ListWorkspaceMembers(ctx(), client, slug)
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(result.WorkspaceListMembers)
			return
		}

		members := result.WorkspaceListMembers
		if len(members) == 0 {
			fmt.Println(dim.Render("  No members"))
			return
		}

		var rows [][]string
		for _, m := range members {
			name := deref(m.DisplayName, deref(m.Username, m.UserId))
			email := deref(m.Email, dim.Render("—"))
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

		result, err := gql.InviteToWorkspace(ctx(), client, id, user, ptr(role))
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(result.WorkspaceInvite)
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
			result, err := gql.ListMyInvites(ctx(), client)
			if err != nil {
				fatal(err.Error())
			}
			if jsonOutput {
				printJSON(result.WorkspaceListMyInvites)
				return
			}
			invites := result.WorkspaceListMyInvites
			if len(invites) == 0 {
				fmt.Println(dim.Render("  No pending invites"))
				return
			}

			var rows [][]string
			for _, inv := range invites {
				wsName := deref(inv.WorkspaceName, "?")
				rows = append(rows, []string{inv.Id, wsName, inv.Role, renderStatus(inv.Status)})
			}

			fmt.Println()
			fmt.Println(styledTable([]string{"ID", "WORKSPACE", "ROLE", "STATUS"}, rows))
			fmt.Println()
		} else {
			slug := args[0]
			result, err := gql.ListWorkspaceInvites(ctx(), client, slug)
			if err != nil {
				fatal(err.Error())
			}
			if jsonOutput {
				printJSON(result.WorkspaceListInvites)
				return
			}
			invites := result.WorkspaceListInvites
			if len(invites) == 0 {
				fmt.Println(dim.Render("  No pending invites"))
				return
			}

			var rows [][]string
			for _, inv := range invites {
				rows = append(rows, []string{inv.Id, inv.Role, renderStatus(inv.Status)})
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
		_, err := gql.AcceptInvite(ctx(), client, args[0])
		if err != nil {
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
		_, err := gql.DeclineInvite(ctx(), client, args[0])
		if err != nil {
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
		_, err := gql.RevokeInvite(ctx(), client, args[0])
		if err != nil {
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

		_, err = gql.RemoveWorkspaceMember(ctx(), client, wsID, userID)
		if err != nil {
			fatal(err.Error())
		}

		if jsonOutput {
			printJSON(map[string]any{"removed": true})
			return
		}
		success("Member removed")
	},
}

func resolveWorkspaceID(client graphql.Client, slug string) (string, error) {
	result, err := gql.ListWorkspaces(ctx(), client)
	if err != nil {
		return "", err
	}
	for _, ws := range result.WorkspaceList {
		if ws.Slug == slug {
			return ws.Id, nil
		}
	}
	return "", fmt.Errorf("workspace %q not found", slug)
}
