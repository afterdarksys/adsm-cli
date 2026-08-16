package cmd

import "github.com/spf13/cobra"

var emailCmd = &cobra.Command{
	Use:     "email",
	Short:   "Manage email domains and delivery",
	Long:    "Configure sending domains, authentication records, and review delivery.",
	GroupID: groupDelivery,
	RunE: func(cmd *cobra.Command, args []string) error {
		return notImplemented("email")
	},
}

func init() {
	rootCmd.AddCommand(emailCmd)
}
