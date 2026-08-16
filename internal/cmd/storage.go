package cmd

import "github.com/spf13/cobra"

var storageCmd = &cobra.Command{
	Use:     "storage",
	Short:   "Manage object storage",
	Long:    "Create and manage storage buckets, and transfer objects.",
	GroupID: groupDelivery,
	RunE: func(cmd *cobra.Command, args []string) error {
		return notImplemented("storage")
	},
}

func init() {
	rootCmd.AddCommand(storageCmd)
}
