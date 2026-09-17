package cmd

import (
	"fmt"
	"github.com/afterdarksys/adsm/internal/output"
	"github.com/spf13/cobra"
)

var servicesCmd = &cobra.Command{
	Use:     "services",
	Short:   "List and inspect platform services",
	Long:    "List the services available to your account and inspect their configuration.",
	GroupID: groupPlatform,
	RunE: func(cmd *cobra.Command, args []string) error {
		services, revision, err := apiClient().Services(cmd.Context())
		if err != nil {
			return err
		}
		if outFormat != output.FormatTable {
			return output.Render(cmd.OutOrStdout(), outFormat, map[string]any{"entitlement_revision": revision, "services": services})
		}
		table := &output.Table{Headers: []string{"ID", "NAME", "STATE", "ACCESS"}}
		for _, service := range services {
			access := "not entitled"
			if service.Available {
				access = "available"
			}
			table.Rows = append(table.Rows, []string{service.ID, service.Name, service.State, access})
		}
		if flagVerbose {
			fmt.Fprintln(cmd.ErrOrStderr(), "entitlement revision:", revision)
		}
		return output.Render(cmd.OutOrStdout(), outFormat, table)
	},
}

func init() {
	rootCmd.AddCommand(servicesCmd)
}
