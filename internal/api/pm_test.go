package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProjectsHitsFacadePath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/adsm/pm/projects" {
			t.Errorf("path = %s, want /v1/adsm/pm/projects", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"projects":[{"id":"p1","name":"Platform","status":"active"}]}}`))
	}))
	defer server.Close()

	client := New(server.URL, WithHTTPClient(server.Client()))
	items, err := client.Projects(context.Background())
	if err != nil || len(items) != 1 || items[0].Name != "Platform" {
		t.Fatalf("items=%+v err=%v", items, err)
	}
}

func TestPMTicketsSendsFiltersToFacade(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/adsm/pm/tickets" {
			t.Errorf("path = %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("project_id") != "p1" || q.Get("status") != "open" || q.Get("priority") != "2" {
			t.Errorf("filters = %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"tickets":[{"id":"wt1","number":"PM-1","summary":"do the thing","status":"open","priority":"2","project_id":"p1"}]}}`))
	}))
	defer server.Close()

	client := New(server.URL, WithHTTPClient(server.Client()))
	items, err := client.PMTickets(context.Background(), PMTicketFilter{ProjectID: "p1", Status: "open", Priority: "2"})
	if err != nil || len(items) != 1 || items[0].Number != "PM-1" {
		t.Fatalf("items=%+v err=%v", items, err)
	}
}

func TestCreatePMTicketSendsBodyAndIdempotencyKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/adsm/pm/tickets" {
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Idempotency-Key") != "pm-key" {
			t.Error("missing idempotency key")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body["summary"] != "do the thing" || body["project_id"] != "p1" {
			t.Errorf("body = %v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"ticket":{"id":"wt2","number":"PM-2","summary":"do the thing","status":"open","project_id":"p1"}}}`))
	}))
	defer server.Close()

	client := New(server.URL, WithHTTPClient(server.Client()))
	item, err := client.CreatePMTicket(context.Background(), PMTicketCreate{ProjectID: "p1", Summary: "do the thing"}, "pm-key")
	if err != nil || item.Number != "PM-2" {
		t.Fatalf("item=%+v err=%v", item, err)
	}
}
