package cmd

import "github.com/spf13/cobra"

var hostsCmd = &cobra.Command{
	Use:     "hosts",
	Short:   "List and inspect hosts",
	Long:    "List the hosts allocated to your account and inspect their capacity.",
	GroupID: groupCompute,
	RunE: func(cmd *cobra.Command, args []string) error {
		return notImplemented("hosts")
	},
}

func init() {
	rootCmd.AddCommand(hostsCmd)
}
