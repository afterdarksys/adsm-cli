package cmd

import "github.com/spf13/cobra"

var telcoCmd = &cobra.Command{
	Use:     "telco",
	Short:   "Manage SIP trunks and telephony",
	Long:    "Configure SIP trunks, numbers, and routing.",
	GroupID: groupDelivery,
	RunE: func(cmd *cobra.Command, args []string) error {
		return notImplemented("telco")
	},
}

func init() {
	rootCmd.AddCommand(telcoCmd)
}
