# Plan 05 — Dark Storage Customer and Operator Client Compatibility

Owners: `darkstorage-cli` and `darkstorage-admin`

Depends on: Plans 00, 01, and 03

## Task 1: Migrate the customer CLI to canonical identity

Files:

- `darkstorage-cli/cmd/login.go`
- `darkstorage-cli/cmd/login_test.go`
- `darkstorage-cli/cmd/client.go`
- `darkstorage-cli/cmd/client_test.go`
- `darkstorage-cli/cmd/config.go`
- `darkstorage-cli/internal/auth/keyring.go`
- `darkstorage-cli/internal/auth/keyring_test.go`

Actions:

- Replace query-parameter bearer-token callback handling with authorization code
  + PKCE for audience `api.darkstorage.io`; add refresh/revoke support.
- Enumerate/select canonical organizations and send the canonical organization
  header. Migrate valid legacy tokens/API keys through an explicit compatibility
  flow without email matching.
- Store OAuth credentials in the OS keychain and remove them from YAML. Preserve
  API-key login only for scoped machine use.
- Update the client for the canonical error/tenant contract while preserving
  bucket, object, and STS command behavior.

Verify:

```sh
cd /Users/ryan/development/darkstorage-cli
go test ./cmd ./internal/auth
go test ./...
```

Required tests: PKCE verifier/challenge, state mismatch, code exchange, refresh,
revoke, wrong audience, organization selection, keychain migration, direct
bucket create/delete, STS issue/use/revoke, and no token in config output.

Done when: the supported direct client uses the same identity, tenant, PDP,
quota, audit, and event paths as an ADSM-mediated storage operation.

## Task 2: Prove operator separation

Files:

- `darkstorage-admin/cmd/client_test.go`
- `darkstorage.io/api/internal/middleware/auth_test.go`
- `darkstorage.io/api/internal/middleware/admin_test.go`

Actions:

- Keep the admin CLI on a separately issued operator credential and audience.
- Require an explicit admin/superadmin operation at `/v1/admin/*`; customer
  tokens, customer API keys, gateway delegation capabilities, and Dark Storage
  customer tokens must not satisfy it.
- Ensure admin credentials cannot be used as customer tenant credentials without
  a separately authorized impersonation/break-glass flow.

Verify:

```sh
cd /Users/ryan/development/darkstorage-admin
go test ./...
cd /Users/ryan/development/afterdark-meta-project/darkstorage.io/api
go test ./internal/middleware ./internal/server
```

Done when: positive operator tests pass and every customer/delegated credential
receives 401/403 on admin routes.
