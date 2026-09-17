package cmd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	clientapi "github.com/afterdarksys/adsm/internal/api"
	"github.com/afterdarksys/adsm/internal/auth"
	"github.com/afterdarksys/adsm/internal/config"
	"github.com/spf13/cobra"
)

var loginOrganization string
var loginNoBrowser bool

var loginCmd = &cobra.Command{
	Use:     "login",
	Short:   "Authenticate with After Dark Systems",
	Long:    "Sign in through After Dark Systems single sign-on and store the session\nfor subsequent commands. Run 'adsm login' before anything else.",
	GroupID: groupIdentity,
	RunE: func(cmd *cobra.Command, args []string) error {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return err
		}
		defer listener.Close()
		oauth := oauthConfig()
		oauth.RedirectURL = fmt.Sprintf("http://%s/callback", listener.Addr().String())
		verifier, challenge, err := auth.PKCE()
		if err != nil {
			return err
		}
		state, _, err := auth.PKCE()
		if err != nil {
			return err
		}
		result := make(chan struct {
			code string
			err  error
		}, 1)
		server := &http.Server{ReadHeaderTimeout: 5 * time.Second}
		server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/callback" {
				http.NotFound(w, r)
				return
			}
			if r.URL.Query().Get("state") != state {
				result <- struct {
					code string
					err  error
				}{err: fmt.Errorf("OAuth state mismatch")}
				http.Error(w, "Invalid login state", 400)
				return
			}
			code := r.URL.Query().Get("code")
			if code == "" {
				result <- struct {
					code string
					err  error
				}{err: fmt.Errorf("authorization failed: %s", r.URL.Query().Get("error"))}
				http.Error(w, "Authorization failed", 400)
				return
			}
			result <- struct {
				code string
				err  error
			}{code: code}
			_, _ = fmt.Fprintln(w, "After Dark Systems login complete. You can close this window.")
		})
		go func() { _ = server.Serve(listener) }()
		defer server.Shutdown(context.Background())
		authorizeURL := oauth.AuthorizeURL(state, challenge)
		fmt.Fprintln(cmd.ErrOrStderr(), "Open this URL to authenticate:", authorizeURL)
		if !loginNoBrowser {
			_ = openBrowser(authorizeURL)
		}
		var callback struct {
			code string
			err  error
		}
		select {
		case callback = <-result:
		case <-cmd.Context().Done():
			return cmd.Context().Err()
		case <-time.After(5 * time.Minute):
			return fmt.Errorf("login timed out")
		}
		if callback.err != nil {
			return callback.err
		}
		credential, err := oauth.Exchange(cmd.Context(), callback.code, verifier)
		if err != nil {
			return err
		}
		store := auth.NewKeyring()
		if err := store.Put(cmd.Context(), cfg.Profile, credential); err != nil {
			return fmt.Errorf("storing credentials in OS keychain: %w", err)
		}
		client := clientapi.New(cfg.APIURL, clientapi.WithTokenFunc(func(context.Context) (string, error) { return credential.AccessToken, nil }))
		memberships, err := client.Memberships(cmd.Context())
		if err != nil {
			_ = store.Delete(context.Background(), cfg.Profile)
			return err
		}
		selected, err := selectMembership(memberships, loginOrganization)
		if err != nil {
			_ = store.Delete(context.Background(), cfg.Profile)
			return err
		}
		cfg.OrganizationID = selected.OrganizationID
		cfg.Account = selected.Name
		if err := config.Save(cfg); err != nil {
			return err
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Logged in to %s (%s)\n", selected.Name, selected.OrganizationID)
		return err
	},
}

func init() {
	loginCmd.Flags().StringVar(&loginOrganization, "organization", "", "organization UUID or slug to select")
	loginCmd.Flags().BoolVar(&loginNoBrowser, "no-browser", false, "print the authorization URL without opening a browser")
	rootCmd.AddCommand(loginCmd)
}

func selectMembership(items []clientapi.Membership, want string) (clientapi.Membership, error) {
	if len(items) == 0 {
		return clientapi.Membership{}, fmt.Errorf("no active organization memberships")
	}
	if want == "" {
		if len(items) == 1 {
			return items[0], nil
		}
		return clientapi.Membership{}, fmt.Errorf("multiple organizations available; rerun with --organization <uuid-or-slug>")
	}
	for _, item := range items {
		if item.OrganizationID == want || item.Slug == want {
			return item, nil
		}
	}
	return clientapi.Membership{}, fmt.Errorf("organization %q is not an active membership", want)
}
func openBrowser(target string) error {
	var command string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		command = "open"
		args = []string{target}
	case "linux":
		command = "xdg-open"
		args = []string{target}
	case "windows":
		command = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", target}
	default:
		return fmt.Errorf("unsupported browser platform")
	}
	return exec.Command(command, args...).Start()
}
