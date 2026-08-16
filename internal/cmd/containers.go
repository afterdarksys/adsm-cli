package cmd

import "github.com/spf13/cobra"

var containersCmd = &cobra.Command{
	Use:     "containers",
	Short:   "Manage containers",
	Long:    "List, inspect, and control containers running on your hosts.",
	GroupID: groupCompute,
	RunE: func(cmd *cobra.Command, args []string) error {
		return notImplemented("containers")
	},
}

func init() {
	rootCmd.AddCommand(containersCmd)
}
