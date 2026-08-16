// Package api is the HTTP client shell for the After Dark Systems API.
//
// Scaffold status: transport, error decoding and request plumbing are real.
// Nothing calls it yet — commands are stubs. Endpoint methods (Account,
// Services, DNS, ...) get added alongside the commands that need them.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// UserAgent identifies this client to the API. Keep the version in sync with
// the build-time version in internal/cmd.
const UserAgent = "adsm-cli"

// Client talks to the After Dark Systems API.
type Client struct {
	BaseURL string
	HTTP    *http.Client

	// tokenFn supplies the bearer token per request rather than holding it on
	// the struct, so a refreshed token is picked up without rebuilding the
	// client — and so a token is never accidentally logged with the Client.
	tokenFn func(context.Context) (string, error)
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient overrides the default HTTP client (useful in tests).
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.HTTP = h }
}

// WithTokenFunc sets the bearer-token provider.
func WithTokenFunc(fn func(context.Context) (string, error)) Option {
	return func(c *Client) { c.tokenFn = fn }
}

// New builds a Client for the given base URL.
func New(baseURL string, opts ...Option) *Client {
	c := &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Error is a structured API error.
//
// The API's documented error shape is {error, message, docs}, so surface all
// three: 'docs' is the difference between a user fixing it themselves and
// filing a ticket.
type Error struct {
	StatusCode int    `json:"-"`
	Code       string `json:"error"`
	Message    string `json:"message"`
	Docs       string `json:"docs,omitempty"`
}

func (e *Error) Error() string {
	msg := e.Message
	if msg == "" {
		msg = e.Code
	}
	if msg == "" {
		msg = http.StatusText(e.StatusCode)
	}
	if e.Docs != "" {
		return fmt.Sprintf("%s (%d) — see %s", msg, e.StatusCode, e.Docs)
	}
	return fmt.Sprintf("%s (%d)", msg, e.StatusCode)
}

// Do issues a request against the API and decodes a JSON response into out.
//
// body may be nil. out may be nil for calls with no response payload.
func (c *Client) Do(ctx context.Context, method, path string, body, out any) error {
	var rdr io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request: %w", err)
		}
		rdr = bytes.NewReader(buf)
	}

	url := c.BaseURL + "/" + strings.TrimLeft(path, "/")
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.tokenFn != nil {
		tok, err := c.tokenFn(ctx)
		if err != nil {
			return fmt.Errorf("resolving credentials: %w", err)
		}
		if tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("calling %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		apiErr := &Error{StatusCode: resp.StatusCode}
		// A non-JSON error body (proxy HTML, gateway timeout) must not mask the
		// status code, so decode failures are ignored on purpose.
		_ = json.NewDecoder(resp.Body).Decode(apiErr)
		return apiErr
	}

	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding response from %s: %w", url, err)
	}
	return nil
}
