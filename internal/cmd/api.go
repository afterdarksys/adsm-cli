package cmd

import "github.com/spf13/cobra"

var apiCmd = &cobra.Command{
	Use:     "api",
	Short:   "Manage API keys and explore the API",
	Long:    "Create, list, and revoke API keys, and explore available API operations.",
	GroupID: groupPlatform,
	RunE: func(cmd *cobra.Command, args []string) error {
		return notImplemented("api")
	},
}

func init() {
	rootCmd.AddCommand(apiCmd)
}
