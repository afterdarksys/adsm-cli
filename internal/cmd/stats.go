package cmd

import "github.com/spf13/cobra"

var statsCmd = &cobra.Command{
	Use:     "stats",
	Short:   "Show usage statistics",
	Long:    "Report usage across your services over a chosen period.",
	GroupID: groupAccount,
	RunE: func(cmd *cobra.Command, args []string) error {
		return notImplemented("stats")
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)
}
