package api

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// Project is a PM Portal project, fronted by the ADSM façade at
// /v1/adsm/pm/projects.
type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Status      string    `json:"status,omitempty"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
}

// PMTicket is a PM Portal work item (case), fronted by /v1/adsm/pm/tickets. The
// façade normalizes PM Portal's native work-item fields (ticket_id, summary)
// into this ADSM shape so the CLI stays consistent with the rest of the surface.
type PMTicket struct {
	ID        string    `json:"id"`
	Number    string    `json:"number,omitempty"`
	Summary   string    `json:"summary"`
	Status    string    `json:"status,omitempty"`
	Priority  string    `json:"priority,omitempty"`
	ProjectID string    `json:"project_id,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// PMTicketCreate is the create-PM-ticket request body. Summary is required.
type PMTicketCreate struct {
	ProjectID   string `json:"project_id,omitempty"`
	Summary     string `json:"summary"`
	Description string `json:"description,omitempty"`
	Priority    string `json:"priority,omitempty"`
	WorkType    string `json:"work_type,omitempty"`
}

// PMTicketFilter narrows a PM ticket list. Empty fields are omitted.
type PMTicketFilter struct {
	ProjectID string
	Status    string
	Priority  string
}

// Projects lists PM Portal projects.
func (c *Client) Projects(ctx context.Context) ([]Project, error) {
	var response envelope[struct {
		Projects []Project `json:"projects"`
	}]
	err := c.Do(ctx, http.MethodGet, "/v1/adsm/pm/projects", nil, &response)
	return response.Data.Projects, err
}

// Project fetches one PM Portal project.
func (c *Client) Project(ctx context.Context, id string) (Project, error) {
	var response envelope[struct {
		Project Project `json:"project"`
	}]
	err := c.Do(ctx, http.MethodGet, "/v1/adsm/pm/projects/"+url.PathEscape(id), nil, &response)
	return response.Data.Project, err
}

// PMTickets lists PM Portal tickets, optionally filtered.
func (c *Client) PMTickets(ctx context.Context, f PMTicketFilter) ([]PMTicket, error) {
	var response envelope[struct {
		Tickets []PMTicket `json:"tickets"`
	}]
	path := "/v1/adsm/pm/tickets"
	q := url.Values{}
	if f.ProjectID != "" {
		q.Set("project_id", f.ProjectID)
	}
	if f.Status != "" {
		q.Set("status", f.Status)
	}
	if f.Priority != "" {
		q.Set("priority", f.Priority)
	}
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	err := c.Do(ctx, http.MethodGet, path, nil, &response)
	return response.Data.Tickets, err
}

// PMTicket fetches one PM Portal ticket.
func (c *Client) PMTicket(ctx context.Context, id string) (PMTicket, error) {
	var response envelope[struct {
		Ticket PMTicket `json:"ticket"`
	}]
	err := c.Do(ctx, http.MethodGet, "/v1/adsm/pm/tickets/"+url.PathEscape(id), nil, &response)
	return response.Data.Ticket, err
}

// CreatePMTicket creates a PM Portal ticket. key is an idempotency key.
func (c *Client) CreatePMTicket(ctx context.Context, req PMTicketCreate, key string) (PMTicket, error) {
	var response envelope[struct {
		Ticket PMTicket `json:"ticket"`
	}]
	headers := http.Header{"Idempotency-Key": {key}}
	err := c.DoHeaders(ctx, http.MethodPost, "/v1/adsm/pm/tickets", req, &response, headers)
	return response.Data.Ticket, err
}
