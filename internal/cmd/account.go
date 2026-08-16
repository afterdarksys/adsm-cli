package cmd

import "github.com/spf13/cobra"

var accountCmd = &cobra.Command{
	Use:     "account",
	Short:   "View and manage your account",
	Long:    "Inspect your profile, organisation membership, and account settings.",
	GroupID: groupIdentity,
	RunE: func(cmd *cobra.Command, args []string) error {
		return notImplemented("account")
	},
}

func init() {
	rootCmd.AddCommand(accountCmd)
}
