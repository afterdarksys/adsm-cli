package cmd

import (
	"fmt"

	clientapi "github.com/afterdarksys/adsm/internal/api"
	"github.com/afterdarksys/adsm/internal/output"
	"github.com/spf13/cobra"
)

var (
	pmTicketProject  string
	pmTicketStatus   string
	pmTicketPriority string

	newPMTicketProject     string
	newPMTicketSummary     string
	newPMTicketDescription string
	newPMTicketPriority    string
	newPMTicketWorkType    string
)

var pmCmd = &cobra.Command{
	Use:     "pm",
	Short:   "Manage project-management projects and tickets",
	Long:    "Manage PM Portal projects and tickets through the customer control plane.",
	GroupID: groupDelivery,
}

var pmProjectsCmd = &cobra.Command{Use: "projects", Short: "Manage PM projects"}
var pmTicketsCmd = &cobra.Command{Use: "tickets", Short: "Manage PM tickets"}

var pmProjectsListCmd = &cobra.Command{Use: "list", Aliases: []string{"ls"}, Short: "List projects", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
	items, err := apiClient().Projects(cmd.Context())
	if err != nil {
		return err
	}
	if outFormat != output.FormatTable {
		return output.Render(cmd.OutOrStdout(), outFormat, items)
	}
	table := &output.Table{Headers: []string{"ID", "NAME", "STATUS"}}
	for _, p := range items {
		table.Rows = append(table.Rows, []string{p.ID, p.Name, p.Status})
	}
	return output.Render(cmd.OutOrStdout(), outFormat, table)
}}

var pmProjectsGetCmd = &cobra.Command{Use: "get ID", Short: "Get a project", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
	item, err := apiClient().Project(cmd.Context(), args[0])
	if err != nil {
		return err
	}
	if outFormat != output.FormatTable {
		return output.Render(cmd.OutOrStdout(), outFormat, item)
	}
	return output.Render(cmd.OutOrStdout(), outFormat, &output.Table{
		Headers: []string{"ID", "NAME", "STATUS", "DESCRIPTION"},
		Rows:    [][]string{{item.ID, item.Name, item.Status, item.Description}},
	})
}}

var pmTicketsListCmd = &cobra.Command{Use: "list", Aliases: []string{"ls"}, Short: "List PM tickets", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
	items, err := apiClient().PMTickets(cmd.Context(), clientapi.PMTicketFilter{ProjectID: pmTicketProject, Status: pmTicketStatus, Priority: pmTicketPriority})
	if err != nil {
		return err
	}
	if outFormat != output.FormatTable {
		return output.Render(cmd.OutOrStdout(), outFormat, items)
	}
	table := &output.Table{Headers: []string{"ID", "NUMBER", "STATUS", "PRIORITY", "SUMMARY"}}
	for _, t := range items {
		table.Rows = append(table.Rows, []string{t.ID, t.Number, t.Status, t.Priority, t.Summary})
	}
	return output.Render(cmd.OutOrStdout(), outFormat, table)
}}

var pmTicketsGetCmd = &cobra.Command{Use: "get ID", Short: "Get a PM ticket", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
	item, err := apiClient().PMTicket(cmd.Context(), args[0])
	if err != nil {
		return err
	}
	if outFormat != output.FormatTable {
		return output.Render(cmd.OutOrStdout(), outFormat, item)
	}
	return output.Render(cmd.OutOrStdout(), outFormat, &output.Table{
		Headers: []string{"ID", "NUMBER", "STATUS", "PRIORITY", "PROJECT", "SUMMARY"},
		Rows:    [][]string{{item.ID, item.Number, item.Status, item.Priority, item.ProjectID, item.Summary}},
	})
}}

var pmTicketsCreateCmd = &cobra.Command{Use: "create", Short: "Create a PM ticket", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
	if newPMTicketSummary == "" {
		return fmt.Errorf("--summary is required")
	}
	item, err := apiClient().CreatePMTicket(cmd.Context(), clientapi.PMTicketCreate{
		ProjectID:   newPMTicketProject,
		Summary:     newPMTicketSummary,
		Description: newPMTicketDescription,
		Priority:    newPMTicketPriority,
		WorkType:    newPMTicketWorkType,
	}, operationKey("pm-ticket-create"))
	if err != nil {
		return err
	}
	if outFormat != output.FormatTable {
		return output.Render(cmd.OutOrStdout(), outFormat, item)
	}
	return output.Render(cmd.OutOrStdout(), outFormat, &output.Table{
		Headers: []string{"ID", "NUMBER", "STATUS", "SUMMARY"},
		Rows:    [][]string{{item.ID, item.Number, item.Status, item.Summary}},
	})
}}

func init() {
	pmTicketsListCmd.Flags().StringVar(&pmTicketProject, "project", "", "filter by project id")
	pmTicketsListCmd.Flags().StringVar(&pmTicketStatus, "status", "", "filter by status")
	pmTicketsListCmd.Flags().StringVar(&pmTicketPriority, "priority", "", "filter by priority")

	pmTicketsCreateCmd.Flags().StringVar(&newPMTicketProject, "project", "", "project id")
	pmTicketsCreateCmd.Flags().StringVar(&newPMTicketSummary, "summary", "", "ticket summary (required)")
	pmTicketsCreateCmd.Flags().StringVar(&newPMTicketDescription, "description", "", "ticket description")
	pmTicketsCreateCmd.Flags().StringVar(&newPMTicketPriority, "priority", "", "priority")
	pmTicketsCreateCmd.Flags().StringVar(&newPMTicketWorkType, "work-type", "", "work type")

	pmProjectsCmd.AddCommand(pmProjectsListCmd, pmProjectsGetCmd)
	pmTicketsCmd.AddCommand(pmTicketsListCmd, pmTicketsGetCmd, pmTicketsCreateCmd)
	pmCmd.AddCommand(pmProjectsCmd, pmTicketsCmd)
	rootCmd.AddCommand(pmCmd)
}
