package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPKCEChallenge(t *testing.T) {
	verifier, challenge, err := PKCE()
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(verifier))
	if challenge != base64.RawURLEncoding.EncodeToString(sum[:]) {
		t.Fatal("challenge does not bind verifier")
	}
	if strings.ContainsAny(verifier, "=+/") {
		t.Fatal("verifier is not URL safe")
	}
}
func TestExchangeAndRefresh(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, _ := io.ReadAll(r.Body)
		values := string(body)
		if calls == 1 && !strings.Contains(values, "code_verifier=verifier") {
			t.Error("missing verifier")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"access","refresh_token":"refresh","token_type":"Bearer","expires_in":3600}`))
	}))
	defer server.Close()
	oauth := OAuth{TokenURL: server.URL, ClientID: "client", RedirectURL: "http://127.0.0.1/callback", HTTP: server.Client()}
	credential, err := oauth.Exchange(context.Background(), "code", "verifier")
	if err != nil || credential.AccessToken != "access" || !credential.Expiry.After(time.Now()) {
		t.Fatalf("credential=%+v err=%v", credential, err)
	}
	if _, err := oauth.Refresh(context.Background(), "refresh"); err != nil {
		t.Fatal(err)
	}
}
