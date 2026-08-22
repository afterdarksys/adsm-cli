// Package mailmap manages the postfix lookup tables that actually drive mail
// on the AfterDark relays.
//
// Why postfix maps and not LDAP: the relays are forward-only. There are no
// mailboxes (relay-a's /var/mail/vhosts is empty and always has been) — mail
// for a served domain is rewritten by an alias and forwarded on. That whole
// job is done by three flat hash tables:
//
//	relay_domains     which domains we accept mail for
//	virtual           address -> forwarding target
//	access_recipient  per-recipient access control (already wired into
//	                  smtpd_recipient_restrictions via check_recipient_access)
//
// The LDAP directory these once pointed at died with the OCI migration and was
// authenticating nobody. Managing the tables directly removes that dependency
// instead of restoring it.
package mailmap

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Map names, relative to the instance's maps/ directory.
const (
	MapDomains = "relay_domains"
	MapVirtual = "virtual"
	MapAccess  = "access_recipient"
)

// FreezeMode decides what happens to mail for a frozen recipient.
type FreezeMode string

const (
	// FreezeHold parks mail in the hold queue. Nothing is lost and unfreezing
	// can release it, at the cost of queue growth while frozen.
	FreezeHold FreezeMode = "hold"
	// FreezeReject bounces mail with a permanent error. The sender finds out
	// immediately; the mail is gone.
	FreezeReject FreezeMode = "reject"
)

// freezeMarker tags entries this tool added, so unfreeze only ever removes
// its own work and never a hand-written access rule.
const freezeMarker = "adsm-freeze"

// Client talks to one postfix instance on one relay host over ssh.
type Client struct {
	// Host is an ssh destination (alias or user@host).
	Host string
	// Instance is the postfix config dir, e.g. /etc/postfix-ext. The relays run
	// multi-instance postfix, so this is not optional — plain postconf reads
	// /etc/postfix and reports a different server's settings.
	Instance string
}

// Entry is one line of a lookup table.
type Entry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Frozen reports whether an access_recipient action came from this tool.
func (e Entry) Frozen() bool { return strings.Contains(e.Value, freezeMarker) }

func (c *Client) mapPath(name string) string {
	return strings.TrimRight(c.Instance, "/") + "/maps/" + name
}

// ssh runs a command on the relay and returns stdout.
func (c *Client) ssh(script string) (string, error) {
	cmd := exec.Command("ssh",
		"-o", "ConnectTimeout=15",
		"-o", "BatchMode=yes",
		c.Host, script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("ssh %s: %w: %s", c.Host, err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// Read parses a lookup table. Comments and blank lines are skipped; they are
// preserved on write by Write, which only ever rewrites the managed region.
func (c *Client) Read(name string) ([]Entry, error) {
	out, err := c.ssh("cat " + c.mapPath(name) + " 2>/dev/null || true")
	if err != nil {
		return nil, err
	}
	return parse(out), nil
}

func parse(s string) []Entry {
	var entries []Entry
	for _, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 2 {
			// A key with no value is malformed for every map type we manage.
			continue
		}
		entries = append(entries, Entry{Key: fields[0], Value: strings.Join(fields[1:], " ")})
	}
	return entries
}

// Write replaces a lookup table, backs up the previous version, rebuilds the
// .db with postmap, and reloads the instance.
//
// The file is shipped base64-encoded so an address or action containing shell
// metacharacters cannot break out of the remote command.
func (c *Client) Write(name string, entries []Entry, reason string) error {
	sort.Slice(entries, func(i, j int) bool { return entries[i].Key < entries[j].Key })

	var b strings.Builder
	fmt.Fprintf(&b, "# %s -- managed by 'adsm email'. Hand edits are preserved on\n", name)
	fmt.Fprintf(&b, "# read but rewritten on the next write; prefer the CLI.\n")
	fmt.Fprintf(&b, "# last change: %s\n#\n", reason)
	width := 0
	for _, e := range entries {
		if len(e.Key) > width {
			width = len(e.Key)
		}
	}
	for _, e := range entries {
		fmt.Fprintf(&b, "%-*s  %s\n", width, e.Key, e.Value)
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(b.String()))
	path := c.mapPath(name)
	stamp := time.Now().UTC().Format("20060102-150405")

	// Backup, write, postmap, reload -- and stop at the first failure so a bad
	// postmap never gets a reload on top of it.
	script := strings.Join([]string{
		"set -e",
		fmt.Sprintf("cp %s %s.bak-%s 2>/dev/null || true", path, path, stamp),
		fmt.Sprintf("printf %%s %s | base64 -d > %s", encoded, path),
		fmt.Sprintf("postmap hash:%s", path),
		fmt.Sprintf("postfix -c %s reload", c.Instance),
	}, "\n")

	_, err := c.ssh(script)
	return err
}

// Reload asks postfix to re-read its tables without restarting.
func (c *Client) Reload() error {
	_, err := c.ssh(fmt.Sprintf("postfix -c %s reload", c.Instance))
	return err
}

var (
	domainRe  = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$`)
	addressRe = regexp.MustCompile(`^[^@\s]+@[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$`)
)

// ValidateDomain rejects input that would produce a map postfix silently
// ignores. A typo'd domain in relay_domains is not an error to postfix, it just
// means mail for the real domain is refused.
func ValidateDomain(d string) error {
	if !domainRe.MatchString(strings.ToLower(strings.TrimSpace(d))) {
		return fmt.Errorf("%q is not a valid domain name", d)
	}
	return nil
}

// ValidateAddress accepts a full address, or a bare domain for catch-alls
// (postfix virtual accepts "@example.com" for that).
func ValidateAddress(a string) error {
	a = strings.ToLower(strings.TrimSpace(a))
	if strings.HasPrefix(a, "@") {
		return ValidateDomain(a[1:])
	}
	if !addressRe.MatchString(a) {
		return fmt.Errorf("%q is not a valid email address", a)
	}
	return nil
}

// FreezeAction renders the access_recipient value for a freeze.
func FreezeAction(mode FreezeMode, note string) (string, error) {
	if note == "" {
		note = "suspended"
	}
	switch mode {
	case FreezeHold:
		return fmt.Sprintf("HOLD %s: %s", freezeMarker, note), nil
	case FreezeReject:
		return fmt.Sprintf("550 5.7.1 %s: %s", freezeMarker, note), nil
	default:
		return "", fmt.Errorf("unknown freeze mode %q: want hold or reject", mode)
	}
}

// Upsert sets key to value, replacing any existing entry.
func Upsert(entries []Entry, key, value string) []Entry {
	for i := range entries {
		if strings.EqualFold(entries[i].Key, key) {
			entries[i].Value = value
			return entries
		}
	}
	return append(entries, Entry{Key: key, Value: value})
}

// Remove deletes key. The bool reports whether anything was removed, so callers
// can tell "removed" from "was never there" instead of reporting a false success.
func Remove(entries []Entry, key string) ([]Entry, bool) {
	for i := range entries {
		if strings.EqualFold(entries[i].Key, key) {
			return append(entries[:i:i], entries[i+1:]...), true
		}
	}
	return entries, false
}

// Find returns the entry for key, if present.
func Find(entries []Entry, key string) (Entry, bool) {
	for _, e := range entries {
		if strings.EqualFold(e.Key, key) {
			return e, true
		}
	}
	return Entry{}, false
}
