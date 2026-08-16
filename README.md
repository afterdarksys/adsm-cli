# adsm

The public end-user CLI for the After Dark Systems ecosystem.

> **Status: scaffolding.** The command tree, configuration, API transport and
> output rendering exist. **No command is implemented** — every one exits
> non-zero with "not implemented yet". This is deliberate; see
> [Stub contract](#stub-contract).

## Build

```bash
./build.sh          # or: go build -o adsm ./cmd/adsm
./adsm --help
```

Requires Go 1.25+. Run `go mod tidy` first to populate `go.sum` — the module
is declared but dependencies have not been vendored yet.

## Command surface

Seventeen top-level commands, grouped by intent so `--help` stays readable.

| Group | Commands |
|---|---|
| Identity & Access | `login` `account` `secrets` |
| Platform | `status` `services` `endpoints` `api` |
| Compute | `hosts` `containers` `k3s` |
| Data & Delivery | `storage` `cdn` `dns` `email` `telco` |
| Account & Usage | `billing` `stats` |

Global flags, available on every command:

| Flag | Default | Purpose |
|---|---|---|
| `--profile` | `default` | Configuration profile to use |
| `-o, --output` | `table` | Output format: `table`, `json`, `yaml` |
| `--api-url` | from config | Override the API endpoint |
| `-v, --verbose` | `false` | Verbose output |
| `--no-color` | `false` | Disable coloured output |

## Layout

```
cmd/adsm/main.go         entrypoint
internal/cmd/            root command, global flags, one file per command
internal/config/         ~/.adsm/config.yaml load & save, profile handling
internal/api/            HTTP transport, auth header injection, error decoding
internal/output/         table / json / yaml rendering
```

Structure mirrors [`adpm`](https://github.com/afterdarksys/adpm) so the two
CLIs read the same way.

## Configuration

`~/.adsm/config.yaml`, overridable with `ADSM_CONFIG_DIR`:

```yaml
current_profile: default
profiles:
  default:
    api_url: https://api.afterdarksys.com
    account: you@example.com
    output: table
```

Written `0600`, directory `0700`.

**Credentials are not stored here.** `config.Config` has no token field, and
adding one would be a regression rather than a shortcut — session tokens
belong in the OS keychain behind `internal/auth`, which is not yet written.
`internal/api` therefore takes a token *provider function* rather than a
token, so a refreshed credential is picked up without rebuilding the client
and the secret never sits on a long-lived struct.

## Stub contract

Every unimplemented command returns via a single helper:

```go
func notImplemented(name string) error {
	return fmt.Errorf("'adsm %s' is not implemented yet", name)
}
```

It **exits non-zero on purpose**. A stub that prints a friendly notice and
exits `0` is indistinguishable from success to a shell script, which is how
half-built CLIs get silently wired into automation and then quietly do
nothing. Implementing a command means replacing its `RunE` body; if you see a
zero exit, the command really ran.

## Implementing a command

1. Add the API call to `internal/api` (a method on `Client` using `Do`).
2. Replace the command's `RunE` to call it.
3. Render through `internal/output` — never `fmt.Print` to stdout directly,
   or `-o json` stops being machine-parseable for that command.
4. Add subcommands with `cobra.Command.AddCommand` on the group's parent
   (`adsm dns list`, `adsm dns add`, ...).

## Not yet built

- `internal/auth` — SSO flow and keychain-backed token storage
- `go.sum` — run `go mod tidy`
- Subcommand trees under each top-level noun
- Shell completions, man pages, release packaging
- Tests
