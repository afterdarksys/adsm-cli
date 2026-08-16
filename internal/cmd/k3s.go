package cmd

import "github.com/spf13/cobra"

var k3sCmd = &cobra.Command{
	Use:     "k3s",
	Short:   "Interact with Kubernetes (k3s) clusters",
	Long:    "Inspect and manage your k3s clusters, nodes, and workloads.",
	GroupID: groupCompute,
	RunE: func(cmd *cobra.Command, args []string) error {
		return notImplemented("k3s")
	},
}

func init() {
	rootCmd.AddCommand(k3sCmd)
}
