package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	clientapi "github.com/afterdarksys/adsm/internal/api"
	"github.com/afterdarksys/adsm/internal/auth"
	"github.com/afterdarksys/adsm/internal/config"
	"github.com/afterdarksys/adsm/internal/output"
)

// Build metadata, overridden at link time by build.sh:
//
//	go build -ldflags "-X .../internal/cmd.version=1.2.3"
var (
	version = "dev"
	commit  = "none"
)

// Global flags, bound in init and readable by every subcommand.
var (
	flagProfile string
	flagOutput  string
	flagAPIURL  string
	flagVerbose bool
	flagNoColor bool
)

// cfg is loaded once in PersistentPreRunE and shared by all subcommands.
var cfg *config.Config

// outFormat is the validated --output value, resolved in PersistentPreRunE so
// commands never have to re-parse the flag.
var outFormat output.Format

// Command group IDs. Seventeen top-level commands is too many for a flat help
// listing, so they are grouped by what the user is trying to do.
const (
	groupIdentity = "identity"
	groupPlatform = "platform"
	groupCompute  = "compute"
	groupDelivery = "delivery"
	groupAccount  = "account"
)

var rootCmd = &cobra.Command{
	Use:   "adsm",
	Short: "After Dark Systems CLI",
	Long: `adsm - the After Dark Systems command line interface.

Manage your account, services, and infrastructure across the After Dark
Systems platform. Authenticate once with 'adsm login'; every other command
reuses that session.`,
	SilenceUsage: true,
	// Execute prints errors itself; without this cobra prints them too and
	// every failure appears twice.
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// 'adsm login' must work before a config exists, so a missing config
		// is not an error here — Load returns defaults.
		c, err := config.Load(flagProfile)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		if flagAPIURL != "" {
			c.APIURL = flagAPIURL
		}
		cfg = c

		// Reject a bad --output before the command runs, rather than letting
		// it fail at render time after the work is already done.
		format := flagOutput
		if !cmd.Flags().Changed("output") && c.Output != "" {
			format = c.Output
		}
		f, err := output.ParseFormat(format)
		if err != nil {
			return err
		}
		outFormat = f
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	pf := rootCmd.PersistentFlags()
	pf.StringVar(&flagProfile, "profile", "", "configuration profile to use (default \"default\")")
	pf.StringVarP(&flagOutput, "output", "o", "table", "output format: table, json, yaml")
	pf.StringVar(&flagAPIURL, "api-url", "", "override the API endpoint")
	pf.BoolVarP(&flagVerbose, "verbose", "v", false, "verbose output")
	pf.BoolVar(&flagNoColor, "no-color", false, "disable coloured output")

	rootCmd.AddGroup(
		&cobra.Group{ID: groupIdentity, Title: "Identity & Access:"},
		&cobra.Group{ID: groupPlatform, Title: "Platform:"},
		&cobra.Group{ID: groupCompute, Title: "Compute:"},
		&cobra.Group{ID: groupDelivery, Title: "Data & Delivery:"},
		&cobra.Group{ID: groupAccount, Title: "Account & Usage:"},
	)

	rootCmd.SetVersionTemplate("adsm {{.Version}}\n")
	rootCmd.Version = fmt.Sprintf("%s (%s)", version, commit)
}

// Execute runs the root command. Called from cmd/adsm/main.go.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// notImplemented is the single contract every scaffolded command uses.
//
// It deliberately exits non-zero. A stub that prints a friendly message and
// exits 0 is indistinguishable from success to a shell script, which is how
// half-built CLIs end up silently wired into automation.
func notImplemented(name string) error {
	return fmt.Errorf("'adsm %s' is not implemented yet", name)
}

func oauthConfig() auth.OAuth {
	return auth.OAuth{AuthorizationURL: "https://login.afterdarksys.com/oauth/authorize", TokenURL: "https://login.afterdarksys.com/oauth/token", RevokeURL: "https://login.afterdarksys.com/oauth/revoke", ClientID: "adsm-cli", Audience: "api.afterdarksys.com"}
}
func apiClient() *clientapi.Client {
	manager := &auth.TokenManager{Store: auth.NewKeyring(), OAuth: oauthConfig(), Profile: cfg.Profile}
	return clientapi.New(cfg.APIURL, clientapi.WithOrganization(cfg.OrganizationID), clientapi.WithTokenFunc(func(ctx context.Context) (string, error) { return manager.Token(ctx) }))
}
