# Plan 06 — Customer CLI and End-to-End Release Gate

Owner: `adsm-cli`; integration fixtures span all repositories

Depends on: Plans 01, 02, 04, and 05

## Task 1: Implement customer authentication and storage commands

Files:

- `internal/auth/*`
- `internal/api/*`
- `internal/cmd/login.go`
- `internal/cmd/account.go`
- `internal/cmd/services.go`
- `internal/cmd/storage.go`
- generated or validated OpenAPI models and tests
- `internal/auth/keyring.go`
- `internal/auth/oauth.go`
- `internal/api/client_test.go`
- `internal/cmd/storage_test.go`

Actions:

- Implement browser PKCE, keychain storage, refresh/revoke/logout, membership
  selection, and organization headers.
- Implement bucket list/get/create/delete, STS issue/list/revoke, usage, and
  operation polling with `--no-wait`.
- Keep structured stdout parseable and progress on stderr; use confirmation plus
  server idempotency/revision protections for deletion.

Verify:

```sh
cd /Users/ryan/development/afterdark-meta-project/adsm-cli
go test ./internal/auth ./internal/api ./internal/cmd
go test ./...
test -z "$(git ls-files adsm)"
```

Done when: a customer can complete the storage control-plane lifecycle without
an admin credential, direct database/SSH access, or upstream provider secret.

## Task 2: Run the ecosystem sandbox release gate

Files:

- `afterdarksys.com/scripts/test-adsm-darkstorage-integration.sh`
- `afterdarksys.com/tests/adsm-darkstorage/compose.yaml`
- `afterdarksys.com/tests/adsm-darkstorage/fixtures/*`
- `afterdarksys.com/docs/runbooks/adsm-darkstorage.md`

Actions:

- Start disposable Authentik, PostgreSQL, authorization/gateway, Dark Storage,
  and MinIO services with a sandbox tenant and product grant.
- Test login, selection, service discovery, create/list/get/delete, STS use and
  revocation, direct-owner convergence, inventory, usage, billing, audit,
  suspension, reconciliation, and recovery.
- Add SLOs/runbooks for auth, PDP, queues, adapter, inbox/replay, audit, meter,
  backup/restore, rotation, and break glass.

Verify:

```sh
cd /Users/ryan/development/afterdark-meta-project/afterdarksys.com
scripts/test-adsm-darkstorage-integration.sh
```

The script exits nonzero unless all negative tenant/audience/replay/revocation/
outage cases fail closed, STS use stops within 60 seconds, and retry/replay/
correction scenarios produce no duplicate resource or charge.

Done when: every definition-of-done statement in the umbrella plan is backed by
an automated sandbox test or an exercised operational runbook.
