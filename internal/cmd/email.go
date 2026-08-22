package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/afterdarksys/adsm/internal/mailmap"
	"github.com/afterdarksys/adsm/internal/output"
)

// Relay defaults. The relays run multi-instance postfix; -ext is the public
// inbound instance that owns relay_domains, virtual and access_recipient.
const (
	defaultRelayHost = "relay-a.msgs.global"
	defaultInstance  = "/etc/postfix-ext"
)

var (
	flagRelayHost string
	flagInstance  string
	flagApply     bool
	flagFreezeAs  string
	flagFreezeWhy string
	flagDomainOnly string
)

var emailCmd = &cobra.Command{
	Use:     "email",
	Short:   "Manage email domains and delivery",
	Long: `Manage the mail relay's domains and forwarding addresses.

Operates directly on the postfix lookup tables (relay_domains, virtual,
access_recipient) on the relay host. The relays are forward-only -- there are
no mailboxes -- so an address is a rewrite rule pointing somewhere else.

Read commands run immediately. Anything that changes the relay prints a plan
and does nothing until you add --apply.`,
	GroupID: groupDelivery,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func client() *mailmap.Client {
	return &mailmap.Client{Host: flagRelayHost, Instance: flagInstance}
}

// plan prints what would change and reports whether to go ahead. Keeping this
// in one place means no subcommand can accidentally mutate without the gate.
func plan(action string, detail string) bool {
	if flagApply {
		return true
	}
	fmt.Printf("PLAN  %s\n      %s\n\n", action, detail)
	fmt.Println("Nothing changed. Re-run with --apply to make it so.")
	return false
}

// ---------------------------------------------------------------- domains

var emailDomainCmd = &cobra.Command{
	Use:   "domain",
	Short: "View and manage the domains this relay accepts mail for",
	Run:   func(cmd *cobra.Command, args []string) { _ = cmd.Help() },
}

var emailDomainListCmd = &cobra.Command{
	Use:   "list",
	Short: "List domains the relay accepts mail for",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client()
		domains, err := c.Read(mailmap.MapDomains)
		if err != nil {
			return err
		}
		access, err := c.Read(mailmap.MapAccess)
		if err != nil {
			return err
		}

		type row struct {
			Domain string `json:"domain"`
			Status string `json:"status"`
		}
		var rows []row
		t := &output.Table{Headers: []string{"DOMAIN", "STATUS"}}
		for _, d := range domains {
			status := "active"
			if e, ok := mailmap.Find(access, "@"+d.Key); ok && e.Frozen() {
				status = "FROZEN"
			}
			rows = append(rows, row{d.Key, status})
			t.Rows = append(t.Rows, []string{d.Key, status})
		}
		if outFormat == output.FormatTable {
			return output.Render(os.Stdout, outFormat, t)
		}
		return output.Render(os.Stdout, outFormat, rows)
	},
}

var emailDomainAddCmd = &cobra.Command{
	Use:   "add <domain>",
	Short: "Accept mail for a domain",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := strings.ToLower(args[0])
		if err := mailmap.ValidateDomain(domain); err != nil {
			return err
		}
		c := client()
		domains, err := c.Read(mailmap.MapDomains)
		if err != nil {
			return err
		}
		if _, exists := mailmap.Find(domains, domain); exists {
			return fmt.Errorf("%s is already accepted", domain)
		}
		if !plan("add domain "+domain, "relay_domains += "+domain+"  OK") {
			return nil
		}
		domains = mailmap.Upsert(domains, domain, "OK")
		if err := c.Write(mailmap.MapDomains, domains, "add domain "+domain); err != nil {
			return err
		}
		fmt.Printf("added %s\n", domain)
		fmt.Printf("note: publish MX, SPF, DKIM and DMARC for %s or mail from it will fail authentication\n", domain)
		return nil
	},
}

var emailDomainRemoveCmd = &cobra.Command{
	Use:   "remove <domain>",
	Short: "Stop accepting mail for a domain",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := strings.ToLower(args[0])
		c := client()
		domains, err := c.Read(mailmap.MapDomains)
		if err != nil {
			return err
		}
		if _, exists := mailmap.Find(domains, domain); !exists {
			return fmt.Errorf("%s is not in relay_domains", domain)
		}
		// Surface orphans before they happen: aliases for a removed domain stop
		// working, and finding out later from a bounce is a bad way to learn.
		virtual, err := c.Read(mailmap.MapVirtual)
		if err != nil {
			return err
		}
		var orphans []string
		for _, v := range virtual {
			if strings.HasSuffix(strings.ToLower(v.Key), "@"+domain) {
				orphans = append(orphans, v.Key)
			}
		}
		detail := "relay_domains -= " + domain
		if len(orphans) > 0 {
			detail += fmt.Sprintf("\n      WARNING: %d alias(es) still point into this domain and will stop resolving: %s",
				len(orphans), strings.Join(orphans, ", "))
		}
		if !plan("remove domain "+domain, detail) {
			return nil
		}
		domains, _ = mailmap.Remove(domains, domain)
		if err := c.Write(mailmap.MapDomains, domains, "remove domain "+domain); err != nil {
			return err
		}
		fmt.Printf("removed %s\n", domain)
		return nil
	},
}

var emailDomainFreezeCmd = &cobra.Command{
	Use:   "freeze <domain>",
	Short: "Suspend mail for a whole domain without deleting its config",
	Args:  cobra.ExactArgs(1),
	RunE:  func(cmd *cobra.Command, args []string) error { return freeze(args[0], true) },
}

var emailDomainUnfreezeCmd = &cobra.Command{
	Use:   "unfreeze <domain>",
	Short: "Resume mail for a frozen domain",
	Args:  cobra.ExactArgs(1),
	RunE:  func(cmd *cobra.Command, args []string) error { return unfreeze(args[0], true) },
}

// ------------------------------------------------------------------ users

var emailUserCmd = &cobra.Command{
	Use:   "user",
	Short: "View and manage forwarding addresses",
	Run:   func(cmd *cobra.Command, args []string) { _ = cmd.Help() },
}

var emailUserListCmd = &cobra.Command{
	Use:   "list",
	Short: "List forwarding addresses",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client()
		virtual, err := c.Read(mailmap.MapVirtual)
		if err != nil {
			return err
		}
		access, err := c.Read(mailmap.MapAccess)
		if err != nil {
			return err
		}

		type row struct {
			Address  string `json:"address"`
			Forwards string `json:"forwards_to"`
			Status   string `json:"status"`
		}
		var rows []row
		t := &output.Table{Headers: []string{"ADDRESS", "FORWARDS TO", "STATUS"}}
		for _, v := range virtual {
			if flagDomainOnly != "" && !strings.HasSuffix(strings.ToLower(v.Key), "@"+strings.ToLower(flagDomainOnly)) {
				continue
			}
			status := "active"
			if e, ok := mailmap.Find(access, v.Key); ok && e.Frozen() {
				status = "FROZEN"
			}
			rows = append(rows, row{v.Key, v.Value, status})
			t.Rows = append(t.Rows, []string{v.Key, v.Value, status})
		}
		if outFormat == output.FormatTable {
			return output.Render(os.Stdout, outFormat, t)
		}
		return output.Render(os.Stdout, outFormat, rows)
	},
}

var emailUserAddCmd = &cobra.Command{
	Use:   "add <address> <forwards-to>",
	Short: "Create a forwarding address",
	Long: `Create a forwarding address.

The relay has no mailboxes, so every address forwards somewhere. Use
"@example.com" as the address for a catch-all.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, target := strings.ToLower(args[0]), args[1]
		if err := mailmap.ValidateAddress(addr); err != nil {
			return err
		}
		if err := mailmap.ValidateAddress(target); err != nil {
			return fmt.Errorf("forwarding target: %w", err)
		}
		c := client()
		virtual, err := c.Read(mailmap.MapVirtual)
		if err != nil {
			return err
		}
		if existing, ok := mailmap.Find(virtual, addr); ok {
			return fmt.Errorf("%s already forwards to %s (remove it first, or use a different address)", addr, existing.Value)
		}
		// An alias in a domain we do not accept mail for is dead on arrival.
		if at := strings.LastIndex(addr, "@"); at >= 0 {
			domain := addr[at+1:]
			domains, err := c.Read(mailmap.MapDomains)
			if err != nil {
				return err
			}
			if _, ok := mailmap.Find(domains, domain); !ok {
				return fmt.Errorf("this relay does not accept mail for %s -- run 'adsm email domain add %s' first", domain, domain)
			}
		}
		if !plan(fmt.Sprintf("add %s -> %s", addr, target), "virtual += "+addr+"  "+target) {
			return nil
		}
		virtual = mailmap.Upsert(virtual, addr, target)
		if err := c.Write(mailmap.MapVirtual, virtual, "add "+addr); err != nil {
			return err
		}
		fmt.Printf("added %s -> %s\n", addr, target)
		return nil
	},
}

var emailUserRemoveCmd = &cobra.Command{
	Use:   "remove <address>",
	Short: "Delete a forwarding address",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		addr := strings.ToLower(args[0])
		c := client()
		virtual, err := c.Read(mailmap.MapVirtual)
		if err != nil {
			return err
		}
		existing, ok := mailmap.Find(virtual, addr)
		if !ok {
			return fmt.Errorf("%s is not a forwarding address here", addr)
		}
		if !plan("remove "+addr, "virtual -= "+addr+"  (was: "+existing.Value+")") {
			return nil
		}
		virtual, _ = mailmap.Remove(virtual, addr)
		if err := c.Write(mailmap.MapVirtual, virtual, "remove "+addr); err != nil {
			return err
		}
		fmt.Printf("removed %s\n", addr)
		return nil
	},
}

var emailUserFreezeCmd = &cobra.Command{
	Use:   "freeze <address>",
	Short: "Suspend an address without deleting it",
	Args:  cobra.ExactArgs(1),
	RunE:  func(cmd *cobra.Command, args []string) error { return freeze(args[0], false) },
}

var emailUserUnfreezeCmd = &cobra.Command{
	Use:   "unfreeze <address>",
	Short: "Resume a frozen address",
	Args:  cobra.ExactArgs(1),
	RunE:  func(cmd *cobra.Command, args []string) error { return unfreeze(args[0], false) },
}

// ----------------------------------------------------------------- freeze

// freeze adds a check_recipient_access entry. That restriction is already live
// in smtpd_recipient_restrictions on the relays, so this needs no config change
// -- only a map update and a reload.
func freeze(target string, isDomain bool) error {
	key := strings.ToLower(target)
	if isDomain {
		if err := mailmap.ValidateDomain(key); err != nil {
			return err
		}
		key = "@" + key
	} else if err := mailmap.ValidateAddress(key); err != nil {
		return err
	}

	action, err := mailmap.FreezeAction(mailmap.FreezeMode(flagFreezeAs), flagFreezeWhy)
	if err != nil {
		return err
	}
	c := client()
	access, err := c.Read(mailmap.MapAccess)
	if err != nil {
		return err
	}
	if e, ok := mailmap.Find(access, key); ok {
		if e.Frozen() {
			return fmt.Errorf("%s is already frozen", key)
		}
		// Refuse to clobber a rule a human wrote by hand.
		return fmt.Errorf("%s already has an access rule (%q) that adsm did not create; resolve it by hand", key, e.Value)
	}

	effect := "mail will be parked in the hold queue and released on unfreeze"
	if mailmap.FreezeMode(flagFreezeAs) == mailmap.FreezeReject {
		effect = "mail will be REJECTED with a permanent error and lost"
	}
	if !plan("freeze "+key, "access_recipient += "+key+"  "+action+"\n      "+effect) {
		return nil
	}
	access = mailmap.Upsert(access, key, action)
	if err := c.Write(mailmap.MapAccess, access, "freeze "+key); err != nil {
		return err
	}
	fmt.Printf("froze %s (%s)\n", key, flagFreezeAs)
	return nil
}

func unfreeze(target string, isDomain bool) error {
	key := strings.ToLower(target)
	if isDomain {
		key = "@" + key
	}
	c := client()
	access, err := c.Read(mailmap.MapAccess)
	if err != nil {
		return err
	}
	e, ok := mailmap.Find(access, key)
	if !ok {
		return fmt.Errorf("%s is not frozen", key)
	}
	if !e.Frozen() {
		return fmt.Errorf("%s has an access rule (%q) that adsm did not create; leaving it alone", key, e.Value)
	}
	if !plan("unfreeze "+key, "access_recipient -= "+key) {
		return nil
	}
	access, _ = mailmap.Remove(access, key)
	if err := c.Write(mailmap.MapAccess, access, "unfreeze "+key); err != nil {
		return err
	}
	fmt.Printf("unfroze %s\n", key)
	if strings.Contains(e.Value, "HOLD") {
		fmt.Printf("held mail is still queued; release it with:  ssh %s postsuper -c %s -H ALL\n", flagRelayHost, flagInstance)
	}
	return nil
}

func init() {
	pf := emailCmd.PersistentFlags()
	pf.StringVar(&flagRelayHost, "relay", defaultRelayHost, "relay host (ssh destination)")
	pf.StringVar(&flagInstance, "instance", defaultInstance, "postfix instance config dir")
	pf.BoolVar(&flagApply, "apply", false, "actually make the change (default: print the plan only)")
	pf.StringVar(&flagFreezeAs, "mode", string(mailmap.FreezeHold), "freeze mode: hold (keep mail) or reject (bounce it)")
	pf.StringVar(&flagFreezeWhy, "reason", "", "reason recorded in the access rule")

	emailUserListCmd.Flags().StringVar(&flagDomainOnly, "domain", "", "only show addresses in this domain")

	emailDomainCmd.AddCommand(emailDomainListCmd, emailDomainAddCmd, emailDomainRemoveCmd,
		emailDomainFreezeCmd, emailDomainUnfreezeCmd)
	emailUserCmd.AddCommand(emailUserListCmd, emailUserAddCmd, emailUserRemoveCmd,
		emailUserFreezeCmd, emailUserUnfreezeCmd)
	emailCmd.AddCommand(emailDomainCmd, emailUserCmd)
	rootCmd.AddCommand(emailCmd)
}
