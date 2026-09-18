package api

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// SupportTicket is a customer-helpdesk ticket in the standalone `support` app,
// fronted by the ADSM façade at /v1/adsm/support/tickets. When escalated, the
// support→backend adapter creates and links an internal `changes` ticket, whose
// reference is surfaced in LinkedChangesRef/LinkedChangesID.
type SupportTicket struct {
	ID               string    `json:"id"`
	Number           string    `json:"number,omitempty"`
	Subject          string    `json:"subject"`
	Description      string    `json:"description,omitempty"`
	Status           string    `json:"status"`
	Priority         string    `json:"priority,omitempty"`
	Category         string    `json:"category,omitempty"`
	Requester        string    `json:"requester,omitempty"`
	LinkedChangesID  string    `json:"linked_changes_id,omitempty"`
	LinkedChangesRef string    `json:"linked_changes_ref,omitempty"`
	CreatedAt        time.Time `json:"created_at,omitempty"`
	UpdatedAt        time.Time `json:"updated_at,omitempty"`
}

// SupportTicketCreate is the create-support-ticket request body. Subject is
// required.
type SupportTicketCreate struct {
	Subject     string `json:"subject"`
	Description string `json:"description,omitempty"`
	Priority    string `json:"priority,omitempty"`
	Category    string `json:"category,omitempty"`
	Requester   string `json:"requester,omitempty"`
}

// SupportTicketFilter narrows a support ticket list. Empty fields are omitted.
type SupportTicketFilter struct {
	Status   string
	Priority string
	Category string
}

// SupportTickets lists support tickets, optionally filtered.
func (c *Client) SupportTickets(ctx context.Context, f SupportTicketFilter) ([]SupportTicket, error) {
	var response envelope[struct {
		Tickets []SupportTicket `json:"tickets"`
	}]
	path := "/v1/adsm/support/tickets"
	q := url.Values{}
	if f.Status != "" {
		q.Set("status", f.Status)
	}
	if f.Priority != "" {
		q.Set("priority", f.Priority)
	}
	if f.Category != "" {
		q.Set("category", f.Category)
	}
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	err := c.Do(ctx, http.MethodGet, path, nil, &response)
	return response.Data.Tickets, err
}

// SupportTicket fetches one support ticket, including any linked changes ticket.
func (c *Client) SupportTicket(ctx context.Context, id string) (SupportTicket, error) {
	var response envelope[struct {
		Ticket SupportTicket `json:"ticket"`
	}]
	err := c.Do(ctx, http.MethodGet, "/v1/adsm/support/tickets/"+url.PathEscape(id), nil, &response)
	return response.Data.Ticket, err
}

// CreateSupportTicket opens a support ticket. key is an idempotency key.
func (c *Client) CreateSupportTicket(ctx context.Context, req SupportTicketCreate, key string) (SupportTicket, error) {
	var response envelope[struct {
		Ticket SupportTicket `json:"ticket"`
	}]
	headers := http.Header{"Idempotency-Key": {key}}
	err := c.DoHeaders(ctx, http.MethodPost, "/v1/adsm/support/tickets", req, &response, headers)
	return response.Data.Ticket, err
}

// EscalateSupportTicket manually escalates a support ticket: the adapter creates
// and links an internal `changes` ticket and returns the support ticket with the
// linked reference populated. key is an idempotency key so a retried escalation
// does not open duplicate internal tickets.
func (c *Client) EscalateSupportTicket(ctx context.Context, id, reason, key string) (SupportTicket, error) {
	var response envelope[struct {
		Ticket SupportTicket `json:"ticket"`
	}]
	headers := http.Header{"Idempotency-Key": {key}}
	err := c.DoHeaders(ctx, http.MethodPost, "/v1/adsm/support/tickets/"+url.PathEscape(id)+"/escalate", map[string]any{"reason": reason}, &response, headers)
	return response.Data.Ticket, err
}
