package cmd

import "github.com/spf13/cobra"

var cdnCmd = &cobra.Command{
	Use:     "cdn",
	Short:   "Manage CDN distributions and cache",
	Long:    "Configure CDN distributions and purge cached content.",
	GroupID: groupDelivery,
	RunE: func(cmd *cobra.Command, args []string) error {
		return notImplemented("cdn")
	},
}

func init() {
	rootCmd.AddCommand(cdnCmd)
}
