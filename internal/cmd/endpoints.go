package cmd

import "github.com/spf13/cobra"

var endpointsCmd = &cobra.Command{
	Use:     "endpoints",
	Short:   "List service endpoints",
	Long:    "Show the network endpoints exposed by your services.",
	GroupID: groupPlatform,
	RunE: func(cmd *cobra.Command, args []string) error {
		return notImplemented("endpoints")
	},
}

func init() {
	rootCmd.AddCommand(endpointsCmd)
}
