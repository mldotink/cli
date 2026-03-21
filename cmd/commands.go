package cmd

// registerCommands adds all commands to rootCmd in display order.
// Called from Execute() before running, so init() order doesn't matter.
func registerCommands() {
	// Core (ordered by importance)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(whoamiCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(listCmd) // services
	rootCmd.AddCommand(reposCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(metricsCmd)
	rootCmd.AddCommand(redeployCmd)
	rootCmd.AddCommand(workspacesCmd)

	// Manage
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(secretsCmd)
	rootCmd.AddCommand(databasesCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(domainsCmd)
	rootCmd.AddCommand(dnsCmd)
	rootCmd.AddCommand(projectsCmd)
	rootCmd.AddCommand(templateCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(chatCmd)
}
