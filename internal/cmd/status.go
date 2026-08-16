package cmd

import "github.com/spf13/cobra"

var statusCmd = &cobra.Command{
	Use:     "status",
	Short:   "Show platform and service health",
	Long:    "Report the current health of After Dark Systems services, including any\nactive incidents affecting your account.",
	GroupID: groupPlatform,
	RunE: func(cmd *cobra.Command, args []string) error {
		return notImplemented("status")
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
