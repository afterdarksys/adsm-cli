# Plan 03 — Dark Storage Canonical Tenant Migration

Owner: `darkstorage.io`

Depends on: Plans 00 and 01

## Task 1: Inventory and migrate tenant authority

Files:

- `api/internal/middleware/auth.go`
- `api/internal/middleware/tenant.go`
- `database/migrations/033_canonical_tenants.sql`
- `database/rollback/033_canonical_tenants.down.sql`
- `api/cmd/validate-tenant-migration/main.go`
- `api/internal/tenant/migration_test.go`
- `scripts/test-canonical-tenant-migration.sh`

Actions:

- Inventory every use of `user_id` as tenant authority across tables, queries,
  handlers, jobs, events, analytics, and alternate lookup paths.
- Add canonical subject and organization mappings plus tenant foreign keys.
- Import mappings from the authenticated platform membership reconciliation API
  keyed by Authentik provider subject. Match only `oauth_provider/oauth_id`;
  never email.
- Add a one-time account claim requiring Authentik login plus a current legacy
  API key/session. Route identities lacking both proofs to audited operator
  remediation. Map personal organizations only after a verified claim.
- Require 100% of active resource-owning accounts mapped or explicitly
  quarantined with a support owner before cutover.
- Dual-read/dual-write for one bounded compatibility release, then require the
  canonical tenant columns. Preserve a tested rollback path until cutover.

Verify:

```sh
cd /Users/ryan/development/afterdark-meta-project/darkstorage.io
scripts/test-canonical-tenant-migration.sh
cd api
go test ./internal/tenant -run 'TestCanonicalTenantMigration|TestAccountClaim'
```

The script starts disposable PostgreSQL, applies canonical forward migrations
through 033, runs preflight/postflight validation, applies the rollback artifact
from `database/rollback` explicitly, and repeats the forward migration. The
forward runner never scans `database/rollback`. The fixture reconciles accounts,
buckets, API keys, STS, audit, and analytics and asserts that no email-derived
mapping exists.

Done when: no middleware or data lookup equates customer subject with tenant and
existing customer resources remain attached to verified organizations.

## Task 2: Replace local customer auth paths

Files:

- `api/internal/middleware/auth.go`
- `api/internal/auth/*`
- `api/internal/middleware/policy.go`
- `api/internal/middleware/delegation.go`
- `api/internal/middleware/delegation_test.go`
- `api/internal/authorization/client.go`
- `api/internal/authorization/client_test.go`
- `database/migrations/034_delegated_authorization.sql`

Actions:

- Validate direct Authentik tokens only for the Dark Storage audience.
- Validate gateway delegation JWTs through the dedicated authorization JWKS and
  bind claims to the exact request.
- Provision the Dark Storage confidential workload identity with audience
  `api.afterdarksys.com` and scopes `authorization.decide` and
  `entitlement.reserve`; retrieve/rotate its credential through Secret Server.
- Call the central PDP/reservation API for direct-client operations; validate
  the signed decision for gateway operations; retain local policy/quota as an
  additional deny layer.

Verify:

```sh
cd /Users/ryan/development/afterdark-meta-project/darkstorage.io/api
go test ./internal/middleware ./internal/authorization ./internal/sts
```

Tests include shared authorization vectors, wrong audience/tenant/operation/
resource/digest/replay, workload-token refresh, non-retryable 401/403, PDP
outage denial, and the 60-second STS revocation bound.

Done when: local JWT tier/role claims cannot independently authorize customer
storage access.
