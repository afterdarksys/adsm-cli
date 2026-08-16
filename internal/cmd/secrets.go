package cmd

import "github.com/spf13/cobra"

var secretsCmd = &cobra.Command{
	Use:     "secrets",
	Short:   "Manage secrets and credentials",
	Long:    "Store, retrieve, and rotate secrets held for your account.",
	GroupID: groupIdentity,
	RunE: func(cmd *cobra.Command, args []string) error {
		return notImplemented("secrets")
	},
}

func init() {
	rootCmd.AddCommand(secretsCmd)
}
