package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientSendsCanonicalOrganizationAndMutationHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Error("missing token")
		}
		if r.Header.Get("X-Organization-ID") != "tenant" {
			t.Error("missing organization")
		}
		if r.Header.Get("Idempotency-Key") != "create-bucket-key" {
			t.Error("missing idempotency key")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(202)
		_, _ = w.Write([]byte(`{"data":{"operation":{"id":"op","state":"accepted"}}}`))
	}))
	defer server.Close()
	client := New(server.URL, WithHTTPClient(server.Client()), WithOrganization("tenant"), WithTokenFunc(func(context.Context) (string, error) { return "token", nil }))
	operation, err := client.CreateBucket(context.Background(), "bucket", "region", "create-bucket-key")
	if err != nil || operation.ID != "op" {
		t.Fatalf("operation=%+v err=%v", operation, err)
	}
}
func TestClientDoesNotPutTokenInError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "proxy failure", 502) }))
	defer server.Close()
	client := New(server.URL, WithHTTPClient(server.Client()), WithTokenFunc(func(context.Context) (string, error) { return "highly-secret-token", nil }))
	err := client.Do(context.Background(), http.MethodGet, "/failure", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "highly-secret-token") {
		t.Fatal("token leaked")
	}
}
