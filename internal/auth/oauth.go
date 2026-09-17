package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type OAuth struct {
	AuthorizationURL, TokenURL, RevokeURL, ClientID, RedirectURL, Audience string
	HTTP                                                                   *http.Client
}

func randomURLSafe(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func PKCE() (verifier, challenge string, err error) {
	verifier, err = randomURLSafe(48)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(sum[:]), nil
}
func (o OAuth) AuthorizeURL(state, challenge string) string {
	u, _ := url.Parse(o.AuthorizationURL)
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", o.ClientID)
	q.Set("redirect_uri", o.RedirectURL)
	q.Set("scope", "openid profile email offline_access services:read")
	q.Set("audience", o.Audience)
	q.Set("state", state)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	u.RawQuery = q.Encode()
	return u.String()
}
func (o OAuth) token(ctx context.Context, values url.Values) (Credential, error) {
	client := o.HTTP
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.TokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return Credential{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := client.Do(req)
	if err != nil {
		return Credential{}, err
	}
	defer res.Body.Close()
	var payload struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int64  `json:"expires_in"`
		Error        string `json:"error"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return Credential{}, err
	}
	if res.StatusCode >= 400 || payload.AccessToken == "" {
		return Credential{}, fmt.Errorf("OAuth token exchange failed: %s", payload.Error)
	}
	return Credential{AccessToken: payload.AccessToken, RefreshToken: payload.RefreshToken, TokenType: payload.TokenType, Expiry: time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second)}, nil
}
func (o OAuth) Exchange(ctx context.Context, code, verifier string) (Credential, error) {
	return o.token(ctx, url.Values{"grant_type": {"authorization_code"}, "client_id": {o.ClientID}, "redirect_uri": {o.RedirectURL}, "code": {code}, "code_verifier": {verifier}})
}
func (o OAuth) Refresh(ctx context.Context, refresh string) (Credential, error) {
	return o.token(ctx, url.Values{"grant_type": {"refresh_token"}, "client_id": {o.ClientID}, "refresh_token": {refresh}})
}
func (o OAuth) Revoke(ctx context.Context, token string) error {
	if o.RevokeURL == "" || token == "" {
		return nil
	}
	client := o.HTTP
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.RevokeURL, strings.NewReader(url.Values{"client_id": {o.ClientID}, "token": {token}}.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return fmt.Errorf("OAuth revocation failed (%d)", res.StatusCode)
	}
	return nil
}

type TokenManager struct {
	Store   Store
	OAuth   OAuth
	Profile string
}

func (m *TokenManager) Token(ctx context.Context) (string, error) {
	credential, err := m.Store.Get(ctx, m.Profile)
	if err != nil {
		return "", err
	}
	if credential.Expiry.After(time.Now().Add(30 * time.Second)) {
		return credential.AccessToken, nil
	}
	if credential.RefreshToken == "" {
		return "", errors.New("session expired; run adsm login")
	}
	updated, err := m.OAuth.Refresh(ctx, credential.RefreshToken)
	if err != nil {
		return "", err
	}
	if updated.RefreshToken == "" {
		updated.RefreshToken = credential.RefreshToken
	}
	if err := m.Store.Put(ctx, m.Profile, updated); err != nil {
		return "", err
	}
	return updated.AccessToken, nil
}
