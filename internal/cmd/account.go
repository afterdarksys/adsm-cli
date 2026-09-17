package cmd

import (
	"fmt"
	"github.com/afterdarksys/adsm/internal/output"
	"github.com/spf13/cobra"
)

var accountCmd = &cobra.Command{
	Use:     "account",
	Short:   "View and manage your account",
	Long:    "Inspect your profile, organisation membership, and account settings.",
	GroupID: groupIdentity,
	RunE: func(cmd *cobra.Command, args []string) error {
		context, err := apiClient().Context(cmd.Context())
		if err != nil {
			return err
		}
		if outFormat != output.FormatTable {
			return output.Render(cmd.OutOrStdout(), outFormat, context)
		}
		return output.Render(cmd.OutOrStdout(), outFormat, &output.Table{Headers: []string{"FIELD", "VALUE"}, Rows: [][]string{{"Organization", fmt.Sprint(context["tenant_id"])}, {"Subject", fmt.Sprint(context["subject_id"])}, {"Authentication", fmt.Sprint(context["authentication"])}, {"Entitlement revision", fmt.Sprint(context["entitlement_revision"])}}})
	},
}

func init() {
	rootCmd.AddCommand(accountCmd)
}
