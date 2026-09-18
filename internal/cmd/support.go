package cmd

import (
	"fmt"

	clientapi "github.com/afterdarksys/adsm/internal/api"
	"github.com/afterdarksys/adsm/internal/output"
	"github.com/spf13/cobra"
)

var (
	supportStatus   string
	supportPriority string
	supportCategory string

	newSupportSubject     string
	newSupportDescription string
	newSupportPriority    string
	newSupportCategory    string
	newSupportRequester   string

	escalateReason string
)

var supportCmd = &cobra.Command{
	Use:     "support",
	Short:   "Manage customer support tickets (helpdesk)",
	Long:    "Manage customer helpdesk tickets. Qualifying tickets escalate into an internal change/ticket, linked automatically.",
	GroupID: groupDelivery,
}

var supportListCmd = &cobra.Command{Use: "list", Aliases: []string{"ls"}, Short: "List support tickets", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
	items, err := apiClient().SupportTickets(cmd.Context(), clientapi.SupportTicketFilter{Status: supportStatus, Priority: supportPriority, Category: supportCategory})
	if err != nil {
		return err
	}
	if outFormat != output.FormatTable {
		return output.Render(cmd.OutOrStdout(), outFormat, items)
	}
	table := &output.Table{Headers: []string{"ID", "NUMBER", "STATUS", "PRIORITY", "LINKED", "SUBJECT"}}
	for _, t := range items {
		table.Rows = append(table.Rows, []string{t.ID, t.Number, t.Status, t.Priority, t.LinkedChangesRef, t.Subject})
	}
	return output.Render(cmd.OutOrStdout(), outFormat, table)
}}

var supportGetCmd = &cobra.Command{Use: "get ID", Short: "Get a support ticket", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
	item, err := apiClient().SupportTicket(cmd.Context(), args[0])
	if err != nil {
		return err
	}
	if outFormat != output.FormatTable {
		return output.Render(cmd.OutOrStdout(), outFormat, item)
	}
	return output.Render(cmd.OutOrStdout(), outFormat, &output.Table{
		Headers: []string{"ID", "NUMBER", "STATUS", "PRIORITY", "REQUESTER", "LINKED CHANGE", "SUBJECT"},
		Rows:    [][]string{{item.ID, item.Number, item.Status, item.Priority, item.Requester, item.LinkedChangesRef, item.Subject}},
	})
}}

var supportCreateCmd = &cobra.Command{Use: "create", Short: "Open a support ticket", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
	if newSupportSubject == "" {
		return fmt.Errorf("--subject is required")
	}
	item, err := apiClient().CreateSupportTicket(cmd.Context(), clientapi.SupportTicketCreate{
		Subject:     newSupportSubject,
		Description: newSupportDescription,
		Priority:    newSupportPriority,
		Category:    newSupportCategory,
		Requester:   newSupportRequester,
	}, operationKey("support-create"))
	if err != nil {
		return err
	}
	if outFormat != output.FormatTable {
		return output.Render(cmd.OutOrStdout(), outFormat, item)
	}
	return output.Render(cmd.OutOrStdout(), outFormat, &output.Table{
		Headers: []string{"ID", "NUMBER", "STATUS", "SUBJECT"},
		Rows:    [][]string{{item.ID, item.Number, item.Status, item.Subject}},
	})
}}

var supportEscalateCmd = &cobra.Command{Use: "escalate ID", Short: "Escalate a support ticket into an internal change/ticket", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
	item, err := apiClient().EscalateSupportTicket(cmd.Context(), args[0], escalateReason, operationKey("support-escalate"))
	if err != nil {
		return err
	}
	if outFormat != output.FormatTable {
		return output.Render(cmd.OutOrStdout(), outFormat, item)
	}
	return output.Render(cmd.OutOrStdout(), outFormat, &output.Table{
		Headers: []string{"ID", "STATUS", "LINKED CHANGE", "SUBJECT"},
		Rows:    [][]string{{item.ID, item.Status, item.LinkedChangesRef, item.Subject}},
	})
}}

func init() {
	supportListCmd.Flags().StringVar(&supportStatus, "status", "", "filter by status")
	supportListCmd.Flags().StringVar(&supportPriority, "priority", "", "filter by priority")
	supportListCmd.Flags().StringVar(&supportCategory, "category", "", "filter by category")

	supportCreateCmd.Flags().StringVar(&newSupportSubject, "subject", "", "ticket subject (required)")
	supportCreateCmd.Flags().StringVar(&newSupportDescription, "description", "", "ticket description")
	supportCreateCmd.Flags().StringVar(&newSupportPriority, "priority", "", "priority (e.g. low|normal|high)")
	supportCreateCmd.Flags().StringVar(&newSupportCategory, "category", "", "category")
	supportCreateCmd.Flags().StringVar(&newSupportRequester, "requester", "", "requester (customer contact)")

	supportEscalateCmd.Flags().StringVar(&escalateReason, "reason", "", "reason for escalation")

	supportCmd.AddCommand(supportListCmd, supportGetCmd, supportCreateCmd, supportEscalateCmd)
	rootCmd.AddCommand(supportCmd)
}
