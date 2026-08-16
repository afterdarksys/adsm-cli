package cmd

import "github.com/spf13/cobra"

var loginCmd = &cobra.Command{
	Use:     "login",
	Short:   "Authenticate with After Dark Systems",
	Long:    "Sign in through After Dark Systems single sign-on and store the session\nfor subsequent commands. Run 'adsm login' before anything else.",
	GroupID: groupIdentity,
	RunE: func(cmd *cobra.Command, args []string) error {
		return notImplemented("login")
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
