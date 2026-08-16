package cmd

import "github.com/spf13/cobra"

var dnsCmd = &cobra.Command{
	Use:     "dns",
	Short:   "Manage DNS zones and records",
	Long:    "Enrol domains, manage zones, and edit DNS records.",
	GroupID: groupDelivery,
	RunE: func(cmd *cobra.Command, args []string) error {
		return notImplemented("dns")
	},
}

func init() {
	rootCmd.AddCommand(dnsCmd)
}
