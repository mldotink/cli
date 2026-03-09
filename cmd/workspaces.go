package cmd

import (
	"fmt"

	"github.com/mldotink/ink-cli/internal/api"
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
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()

		var result struct {
			WorkspaceList []struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				Slug      string `json:"slug"`
				IsDefault bool   `json:"isDefault"`
				Role      string `json:"role"`
			} `json:"workspaceList"`
		}

		err := client.Do(`{ workspaceList { id name slug isDefault role } }`, nil, &result)
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

		var result struct {
			WorkspaceCreate struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Slug string `json:"slug"`
			} `json:"workspaceCreate"`
		}

		vars := map[string]any{"name": name, "slug": slug}
		if desc != "" {
			vars["desc"] = desc
		}

		err := client.Do(`mutation($name: String!, $slug: String!, $desc: String) {
			workspaceCreate(name: $name, slug: $slug, description: $desc) { id name slug }
		}`, vars, &result)
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

		var result struct {
			WorkspaceDelete bool `json:"workspaceDelete"`
		}

		err = client.Do(`mutation($id: ID!) { workspaceDelete(id: $id) }`,
			map[string]any{"id": id}, &result)
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

		var result struct {
			WorkspaceListMembers []struct {
				UserID      string  `json:"userId"`
				Email       *string `json:"email"`
				Username    *string `json:"username"`
				DisplayName *string `json:"displayName"`
				Role        string  `json:"role"`
			} `json:"workspaceListMembers"`
		}

		err := client.Do(`query($slug: String!) {
			workspaceListMembers(workspaceSlug: $slug) {
				userId email username displayName role
			}
		}`, map[string]any{"slug": slug}, &result)
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
			name := deref(m.DisplayName, deref(m.Username, m.UserID))
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

		var result struct {
			WorkspaceInvite struct {
				ID     string `json:"id"`
				Role   string `json:"role"`
				Status string `json:"status"`
			} `json:"workspaceInvite"`
		}

		err = client.Do(`mutation($id: ID!, $user: String!, $role: String) {
			workspaceInvite(workspaceId: $id, user: $user, role: $role) { id role status }
		}`, map[string]any{"id": id, "user": user, "role": role}, &result)
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

		type invite struct {
			ID            string  `json:"id"`
			WorkspaceName *string `json:"workspaceName"`
			WorkspaceSlug *string `json:"workspaceSlug"`
			Role          string  `json:"role"`
			Status        string  `json:"status"`
		}

		if len(args) == 0 {
			var result struct {
				WorkspaceListMyInvites []invite `json:"workspaceListMyInvites"`
			}
			err := client.Do(`{ workspaceListMyInvites { id workspaceName workspaceSlug role status } }`, nil, &result)
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
				rows = append(rows, []string{inv.ID, wsName, inv.Role, renderStatus(inv.Status)})
			}

			fmt.Println()
			fmt.Println(styledTable([]string{"ID", "WORKSPACE", "ROLE", "STATUS"}, rows))
			fmt.Println()
		} else {
			slug := args[0]
			var result struct {
				WorkspaceListInvites []invite `json:"workspaceListInvites"`
			}
			err := client.Do(`query($slug: String!) {
				workspaceListInvites(workspaceSlug: $slug) { id role status }
			}`, map[string]any{"slug": slug}, &result)
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
		var result struct {
			WorkspaceAcceptInvite bool `json:"workspaceAcceptInvite"`
		}
		err := client.Do(`mutation($id: ID!) { workspaceAcceptInvite(inviteId: $id) }`,
			map[string]any{"id": args[0]}, &result)
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
		var result struct {
			WorkspaceDeclineInvite bool `json:"workspaceDeclineInvite"`
		}
		err := client.Do(`mutation($id: ID!) { workspaceDeclineInvite(inviteId: $id) }`,
			map[string]any{"id": args[0]}, &result)
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
		var result struct {
			WorkspaceRevokeInvite bool `json:"workspaceRevokeInvite"`
		}
		err := client.Do(`mutation($id: ID!) { workspaceRevokeInvite(inviteId: $id) }`,
			map[string]any{"id": args[0]}, &result)
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

		var result struct {
			WorkspaceRemoveMember bool `json:"workspaceRemoveMember"`
		}
		err = client.Do(`mutation($wsId: ID!, $userId: ID!) {
			workspaceRemoveMember(workspaceId: $wsId, userId: $userId)
		}`, map[string]any{"wsId": wsID, "userId": userID}, &result)
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

func resolveWorkspaceID(client *api.Client, slug string) (string, error) {
	var result struct {
		WorkspaceList []struct {
			ID   string `json:"id"`
			Slug string `json:"slug"`
		} `json:"workspaceList"`
	}
	err := client.Do(`{ workspaceList { id slug } }`, nil, &result)
	if err != nil {
		return "", err
	}
	for _, ws := range result.WorkspaceList {
		if ws.Slug == slug {
			return ws.ID, nil
		}
	}
	return "", fmt.Errorf("workspace %q not found", slug)
}
