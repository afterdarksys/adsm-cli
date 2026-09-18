package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSupportTicketsHitsFacadeWithFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/adsm/support/tickets" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("category") != "billing" {
			t.Errorf("category filter = %q", r.URL.Query().Get("category"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"tickets":[{"id":"s1","number":"SUP-1","subject":"cannot log in","status":"open","priority":"high"}]}}`))
	}))
	defer server.Close()

	client := New(server.URL, WithHTTPClient(server.Client()))
	items, err := client.SupportTickets(context.Background(), SupportTicketFilter{Category: "billing"})
	if err != nil || len(items) != 1 || items[0].Number != "SUP-1" {
		t.Fatalf("items=%+v err=%v", items, err)
	}
}

func TestCreateSupportTicketSendsBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/adsm/support/tickets" {
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Idempotency-Key") == "" {
			t.Error("missing idempotency key")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body["subject"] != "cannot log in" {
			t.Errorf("subject = %v", body["subject"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"ticket":{"id":"s2","number":"SUP-2","subject":"cannot log in","status":"open"}}}`))
	}))
	defer server.Close()

	client := New(server.URL, WithHTTPClient(server.Client()))
	item, err := client.CreateSupportTicket(context.Background(), SupportTicketCreate{Subject: "cannot log in"}, "support-key")
	if err != nil || item.Number != "SUP-2" {
		t.Fatalf("item=%+v err=%v", item, err)
	}
}

func TestEscalateSupportTicketPostsToEscalatePathAndReturnsLink(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/adsm/support/tickets/SUP-2/escalate" {
			t.Errorf("%s %s, want POST /v1/adsm/support/tickets/SUP-2/escalate", r.Method, r.URL.Path)
		}
		if r.Header.Get("Idempotency-Key") == "" {
			t.Error("escalation must be idempotent")
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["reason"] != "needs infra change" {
			t.Errorf("reason = %v", body["reason"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"ticket":{"id":"s2","number":"SUP-2","subject":"cannot log in","status":"escalated","linked_changes_id":"c9","linked_changes_ref":"CHG-9"}}}`))
	}))
	defer server.Close()

	client := New(server.URL, WithHTTPClient(server.Client()))
	item, err := client.EscalateSupportTicket(context.Background(), "SUP-2", "needs infra change", "esc-key")
	if err != nil {
		t.Fatalf("EscalateSupportTicket: %v", err)
	}
	if item.Status != "escalated" || item.LinkedChangesRef != "CHG-9" {
		t.Fatalf("item=%+v", item)
	}
}
