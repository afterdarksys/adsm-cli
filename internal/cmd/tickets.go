package cmd

import (
	"fmt"

	clientapi "github.com/afterdarksys/adsm/internal/api"
	"github.com/afterdarksys/adsm/internal/output"
	"github.com/spf13/cobra"
)

var (
	ticketFilterType  string
	ticketFilterState string

	newTicketTitle       string
	newTicketDescription string
	newTicketType        string
	newTicketPriority    string
	newTicketCategory    string
	newTicketIndustry    string
	newTicketVisibility  string
)

// ticketRef prefers the human identifier (PRB/CHG/SUP/TKT-...) and falls back to
// the internal id.
func ticketRef(t clientapi.Ticket) string {
	if t.Identifier != "" {
		return t.Identifier
	}
	return t.ID
}

var ticketsCmd = &cobra.Command{
	Use:     "tickets",
	Short:   "Manage tickets (Problem, Change, Support, Generic)",
	Long:    "Manage tickets in the After Dark change/ticket system through the customer control plane.",
	GroupID: groupDelivery,
}

var ticketsListCmd = &cobra.Command{Use: "list", Aliases: []string{"ls"}, Short: "List tickets", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
	items, err := apiClient().Tickets(cmd.Context(), clientapi.TicketFilter{Type: ticketFilterType, State: ticketFilterState})
	if err != nil {
		return err
	}
	if outFormat != output.FormatTable {
		return output.Render(cmd.OutOrStdout(), outFormat, items)
	}
	table := &output.Table{Headers: []string{"ID", "TYPE", "STATE", "PRIORITY", "TITLE"}}
	for _, t := range items {
		table.Rows = append(table.Rows, []string{ticketRef(t), t.Type, t.State, t.Priority, t.Title})
	}
	return output.Render(cmd.OutOrStdout(), outFormat, table)
}}

var ticketsGetCmd = &cobra.Command{Use: "get ID", Short: "Get a ticket", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
	item, err := apiClient().Ticket(cmd.Context(), args[0])
	if err != nil {
		return err
	}
	if outFormat != output.FormatTable {
		return output.Render(cmd.OutOrStdout(), outFormat, item)
	}
	return output.Render(cmd.OutOrStdout(), outFormat, &output.Table{
		Headers: []string{"ID", "TYPE", "STATE", "PRIORITY", "VISIBILITY", "TITLE"},
		Rows:    [][]string{{ticketRef(item), item.Type, item.State, item.Priority, item.Visibility, item.Title}},
	})
}}

var ticketsCreateCmd = &cobra.Command{Use: "create", Short: "Create a ticket", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
	if newTicketTitle == "" {
		return fmt.Errorf("--title is required")
	}
	item, err := apiClient().CreateTicket(cmd.Context(), clientapi.TicketCreate{
		Title:       newTicketTitle,
		Description: newTicketDescription,
		Type:        newTicketType,
		Priority:    newTicketPriority,
		Category:    newTicketCategory,
		Industry:    newTicketIndustry,
		Visibility:  newTicketVisibility,
	}, operationKey("ticket-create"))
	if err != nil {
		return err
	}
	if outFormat != output.FormatTable {
		return output.Render(cmd.OutOrStdout(), outFormat, item)
	}
	return output.Render(cmd.OutOrStdout(), outFormat, &output.Table{
		Headers: []string{"ID", "TYPE", "STATE", "TITLE"},
		Rows:    [][]string{{ticketRef(item), item.Type, item.State, item.Title}},
	})
}}

func init() {
	ticketsListCmd.Flags().StringVar(&ticketFilterType, "type", "", "filter by type (problem|change|support|generic)")
	ticketsListCmd.Flags().StringVar(&ticketFilterState, "state", "", "filter by state")

	ticketsCreateCmd.Flags().StringVar(&newTicketTitle, "title", "", "ticket title (required)")
	ticketsCreateCmd.Flags().StringVar(&newTicketDescription, "description", "", "ticket description")
	ticketsCreateCmd.Flags().StringVar(&newTicketType, "type", "", "ticket type (problem|change|support|generic); defaults to generic")
	ticketsCreateCmd.Flags().StringVar(&newTicketPriority, "priority", "", "priority (e.g. low|normal|high)")
	ticketsCreateCmd.Flags().StringVar(&newTicketCategory, "category", "", "category (e.g. infrastructure)")
	ticketsCreateCmd.Flags().StringVar(&newTicketIndustry, "industry", "", "industry (e.g. it)")
	ticketsCreateCmd.Flags().StringVar(&newTicketVisibility, "visibility", "", "visibility (organization|private)")

	ticketsCmd.AddCommand(ticketsListCmd, ticketsGetCmd, ticketsCreateCmd)
	rootCmd.AddCommand(ticketsCmd)
}
