package cmd

import (
	"fmt"

	"github.com/afterdarksys/adsm/internal/auth"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{Use: "logout", Short: "Revoke and remove the current session", GroupID: groupIdentity, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
	store := auth.NewKeyring()
	credential, err := store.Get(cmd.Context(), cfg.Profile)
	if err != nil {
		return err
	}
	oauth := oauthConfig()
	if err := oauth.Revoke(cmd.Context(), credential.RefreshToken); err != nil {
		return err
	}
	if err := store.Delete(cmd.Context(), cfg.Profile); err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), "Logged out")
	return err
}}

func init() { rootCmd.AddCommand(logoutCmd) }
