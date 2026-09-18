package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTicketsListHitsFacadeWithFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/adsm/tickets" {
			t.Errorf("path = %s, want /v1/adsm/tickets", r.URL.Path)
		}
		if got := r.URL.Query().Get("type"); got != "support" {
			t.Errorf("type filter = %q, want support", got)
		}
		if got := r.URL.Query().Get("state"); got != "open" {
			t.Errorf("state filter = %q, want open", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"tickets":[{"id":"t1","identifier":"SUP-1","type":"support","state":"open","priority":"normal","title":"help"}]}}`))
	}))
	defer server.Close()

	client := New(server.URL, WithHTTPClient(server.Client()))
	items, err := client.Tickets(context.Background(), TicketFilter{Type: "support", State: "open"})
	if err != nil {
		t.Fatalf("Tickets: %v", err)
	}
	if len(items) != 1 || items[0].Identifier != "SUP-1" || items[0].Type != "support" {
		t.Fatalf("items = %+v", items)
	}
}

func TestTicketsListOmitsEmptyFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query string, got %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"tickets":[]}}`))
	}))
	defer server.Close()

	client := New(server.URL, WithHTTPClient(server.Client()))
	if _, err := client.Tickets(context.Background(), TicketFilter{}); err != nil {
		t.Fatalf("Tickets: %v", err)
	}
}

func TestTicketGetUsesEscapedPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/adsm/tickets/CHG-9" {
			t.Errorf("path = %s, want /v1/adsm/tickets/CHG-9", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"ticket":{"id":"t9","identifier":"CHG-9","type":"change","state":"pending_approval","title":"deploy"}}}`))
	}))
	defer server.Close()

	client := New(server.URL, WithHTTPClient(server.Client()))
	item, err := client.Ticket(context.Background(), "CHG-9")
	if err != nil {
		t.Fatalf("Ticket: %v", err)
	}
	if item.Type != "change" || item.State != "pending_approval" {
		t.Fatalf("item = %+v", item)
	}
}

func TestCreateTicketSendsBodyAndIdempotencyKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/adsm/tickets" {
			t.Errorf("%s %s, want POST /v1/adsm/tickets", r.Method, r.URL.Path)
		}
		if r.Header.Get("Idempotency-Key") != "ticket-create-key" {
			t.Error("missing idempotency key")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["title"] != "help" {
			t.Errorf("title = %v, want help", body["title"])
		}
		// An empty type must be omitted so the server can default it to Generic.
		if _, present := body["type"]; present {
			t.Errorf("empty type should be omitted, got %v", body["type"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"ticket":{"id":"t2","identifier":"TKT-2","type":"generic","state":"submitted","title":"help"}}}`))
	}))
	defer server.Close()

	client := New(server.URL, WithHTTPClient(server.Client()))
	item, err := client.CreateTicket(context.Background(), TicketCreate{Title: "help"}, "ticket-create-key")
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	if item.Identifier != "TKT-2" || item.Type != "generic" {
		t.Fatalf("item = %+v", item)
	}
}
