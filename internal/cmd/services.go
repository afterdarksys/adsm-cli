package cmd

import "github.com/spf13/cobra"

var servicesCmd = &cobra.Command{
	Use:     "services",
	Short:   "List and inspect platform services",
	Long:    "List the services available to your account and inspect their configuration.",
	GroupID: groupPlatform,
	RunE: func(cmd *cobra.Command, args []string) error {
		return notImplemented("services")
	},
}

func init() {
	rootCmd.AddCommand(servicesCmd)
}
