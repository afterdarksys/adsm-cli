package cmd

import "github.com/spf13/cobra"

var billingCmd = &cobra.Command{
	Use:     "billing",
	Short:   "View invoices, plans and payment methods",
	Long:    "Review invoices and usage charges, and manage plans and payment methods.",
	GroupID: groupAccount,
	RunE: func(cmd *cobra.Command, args []string) error {
		return notImplemented("billing")
	},
}

func init() {
	rootCmd.AddCommand(billingCmd)
}
