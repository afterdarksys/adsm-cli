package api

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// Ticket is a ticket in the After Dark change/ticket system (the `changes`
// service), fronted by the ADSM façade at /v1/adsm/tickets. Identifier carries
// the type prefix (PRB/CHG/SUP/TKT); ID is the stable internal id.
type Ticket struct {
	ID          string    `json:"id"`
	Identifier  string    `json:"identifier,omitempty"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	State       string    `json:"state"`
	Priority    string    `json:"priority,omitempty"`
	Category    string    `json:"category,omitempty"`
	Industry    string    `json:"industry,omitempty"`
	Visibility  string    `json:"visibility,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TicketCreate is the create-ticket request body. Only Title is required; Type
// defaults server-side to Generic (TKT) when empty.
type TicketCreate struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
	Priority    string `json:"priority,omitempty"`
	Category    string `json:"category,omitempty"`
	Industry    string `json:"industry,omitempty"`
	Visibility  string `json:"visibility,omitempty"`
}

// TicketFilter narrows a ticket list. Empty fields are omitted from the query.
type TicketFilter struct {
	Type  string
	State string
}

// Tickets lists tickets, optionally filtered by type and state.
func (c *Client) Tickets(ctx context.Context, f TicketFilter) ([]Ticket, error) {
	var response envelope[struct {
		Tickets []Ticket `json:"tickets"`
	}]
	path := "/v1/adsm/tickets"
	q := url.Values{}
	if f.Type != "" {
		q.Set("type", f.Type)
	}
	if f.State != "" {
		q.Set("state", f.State)
	}
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	err := c.Do(ctx, http.MethodGet, path, nil, &response)
	return response.Data.Tickets, err
}

// Ticket fetches a single ticket by its id or identifier.
func (c *Client) Ticket(ctx context.Context, id string) (Ticket, error) {
	var response envelope[struct {
		Ticket Ticket `json:"ticket"`
	}]
	err := c.Do(ctx, http.MethodGet, "/v1/adsm/tickets/"+url.PathEscape(id), nil, &response)
	return response.Data.Ticket, err
}

// CreateTicket creates a ticket. key is an idempotency key so a retried create
// does not open duplicate tickets.
func (c *Client) CreateTicket(ctx context.Context, req TicketCreate, key string) (Ticket, error) {
	var response envelope[struct {
		Ticket Ticket `json:"ticket"`
	}]
	headers := http.Header{"Idempotency-Key": {key}}
	err := c.DoHeaders(ctx, http.MethodPost, "/v1/adsm/tickets", req, &response, headers)
	return response.Data.Ticket, err
}
