# CLI service integration: tickets, project management, support (+ email remediation)

Status: proposed (2026-09-18)
Scope: `adsm-cli` (new commands + api client methods), and the ADSM façade
adapters in `afterdarksys.com/api` (backend, separate repo)
Depends on: the P0 auth work in `UPDATED_TODO.md` (OAuth client registration,
delegation signing) and one façade adapter per owning service.

## Goal

Make the existing intranet services reachable from `adsm`: tickets (the
`changes` app), project management (`pmportal`), and support (the `support`
subdomain). None of these needs a new backend — they exist. The work is
integration through the ADSM façade, plus retiring the one command that
currently bypasses it (`email`).

## Architecture (non-negotiable)

The CLI calls ONLY the versioned customer API at `https://api.afterdarksys.com`.
Each owning service stays behind an ADSM façade adapter that authenticates the
ADSM principal, selects the tenant, enforces entitlements, calls the owning
service, and writes an audit event. This mirrors the working `storage` vertical
(`/v1/adsm/...`). The CLI must NOT call `api.changes.afterdarksys.com` or SSH a
host directly — that is the `email` anti-pattern this plan also fixes.

```
adsm CLI ──bearer──> api.afterdarksys.com /v1/adsm/{tickets,pm,support,email}
                         │  (façade: authN, tenant, entitlement, audit)
                         └──> owning service API (changes /v1, pmportal, support, relay)
```

## Contract mapping

### Tickets  ←→  changes app
- Owning service API: `changes` exposes `https://api.changes.afterdarksys.com/v1`
  with `X-API-Key` auth; documented endpoints `/v1/tickets` (GET list, POST
  create), `/v1/auth/me`. Ticket types PRB/CHG/SUP/TKT (default Generic/TKT),
  visibility Organization|Private, fields: title, description, priority,
  category, industry. Only Change tickets enter the approval queue.
- Façade: `/v1/adsm/tickets` (GET list, POST create), `/v1/adsm/tickets/{id}`
  (GET), later `/v1/adsm/tickets/{id}/comments` and `.../link` (PM Portal
  universal-links). The façade holds the changes API key server-side; the CLI
  presents only its ADSM bearer token.
- CLI: `adsm tickets list [--type] [--state]`, `adsm tickets get ID`,
  `adsm tickets create --title --description [--type] [--priority] [--category]
  [--industry] [--visibility]`.

### Project management  ←→  pmportal
- Owning service: `pmportal` is a Next.js + Prisma app with `routes/` and API
  handlers; confirm its `/api` surface and auth. (pmportal had env/registry
  defects in the Sept intranet review — verify health first.)
- Façade: `/v1/adsm/pm/cases` (+ get/create), plus the universal-links relation
  `pmportal:ticket` ↔ `changes:change` already defined in the changes app.
- CLI: `adsm pm cases list|get|create`, and `adsm tickets link` for cross-links.

### Support  ←→  support subdomain (customer-facing) → links into changes (internal)
- RESOLVED: `support` is its OWN standalone app (own `schema.sql`, `migrations/`,
  `cron/`, `workers/`, `bin/`, and an `api/` under `public/` — a PHP-style app,
  not Next.js/Prisma). It and `changes` are DELIBERATELY two systems: `support`
  is the customer-facing helpdesk, `changes` is the internal ticket system.
- Design intent (Ryan, 2026-09-18): `support` is the real customer helpdesk AND
  the FILTER that keeps noise out of `changes`. `changes` is the ESCALATION
  system — it only receives tickets that support escalates, so it stays a clean
  internal work/problem/change record rather than a dumping ground. Flow:
  customer → support (triage/filter) → escalate qualifying tickets → changes
  (internal), with the two linked and moving together thereafter.
- The linking trigger is ESCALATION, not open. A support ticket becomes a linked
  `changes` ticket when it is escalated — by a support agent's explicit escalate
  action, and/or by automatic rules (e.g. category/severity). Most support
  tickets never escalate; that is the point of the filter.
- Mechanism: reuse the existing universal-links service (Login-owned; it already
  backs `pmportal:ticket` ↔ `changes:change`). Register a `support:ticket`
  provider and publish a relationship `support:ticket` ↔ `changes:change`, and
  provision link/discovery/read/unlink entitlements, as the PM Portal linking
  already requires. The escalate-and-link logic lives in the support→backend
  adapter, not the CLI.
- Façade + CLI: `/v1/adsm/support/tickets` (list/get/create) plus an
  `escalate` action (`POST /v1/adsm/support/tickets/{id}/escalate`) that creates
  and links the internal `changes` ticket. `adsm support get` surfaces the linked
  `changes` ticket; `adsm support escalate ID` performs a manual escalation.
  Confirm the support app's actual routes under `public/api` when building it.
- RESOLVED (Ryan): escalations are BOTH manual (an agent/human hits escalate) and
  rule-based (auto-escalate on category, severity, and/or SLA breach). The adapter
  needs a configurable rule set for auto-escalation plus an explicit escalate
  action; both create + link the internal `changes` ticket the same way.

## Agent-operated tickets (design constraint, forward-looking)

Ryan's intent: software agents (Claude, Codex, others) will handle tickets, not
just humans. The ticket/API model must not preclude this:
- **Machine identities.** Agents authenticate as service principals with scoped
  entitlements, distinct from human OIDC sessions. The platform already separates
  operator API-key scope (`/v1/admin/*`) from customer OIDC; agent identities fit
  that model with their own narrow scopes (read tickets, comment, transition,
  resolve — never approve their own Change tickets).
- **Actionable API, not just read.** Beyond list/get/create, the façade needs
  comment, state-transition, assign, and resolve actions so an agent can work a
  ticket end to end. Approval of Change tickets stays gated to a separate human/
  entitlement path (agents must not self-approve).
- **Auditability.** Every agent action is attributed to its principal and written
  to the central audit event (already a façade requirement). This is what makes
  agent-handled tickets reviewable.
- **Idempotency + optimistic revision.** Agents retry; every mutation already
  carries an idempotency key and If-Match revision (see storage), so concurrent
  agent+human edits fail closed on a stale revision rather than clobbering.
This is a constraint on the contract, not this session's build scope.

## Email remediation (fix the bypass without breaking the tool)

`internal/cmd/email.go` operates directly on the postfix relay
(`relay-a.msgs.global`), bypassing OAuth, tenant, entitlements, and audit. Do
NOT rip it out — it works and is in use. Instead:
1. Build a `/v1/adsm/email` façade adapter that owns the relay-table operations
   (relay_domains, virtual, access_recipient) behind authN/tenant/entitlement/
   audit.
2. Repoint the CLI command at the façade.
3. Until the adapter ships, mark the direct path clearly as an unmanaged/legacy
   operator escape hatch (a visible warning + an explicit `--unmanaged` flag),
   so it is never mistaken for a governed, audited path.

## Sequencing (what blocks what)

1. **Blocked on you/ops (P0, cannot do from here):** register the `adsm-cli`
   OAuth client in the deployed Authentik; provision delegation key + client
   secrets in Secret Server. Nothing authenticates until this lands.
2. **Backend (afterdarksys.com/api):** build one façade adapter per service
   (tickets, pm, support, email). Each needs the owning service deployed and
   healthy — `changes` and `pmportal` both had defects in the Sept intranet
   review, so verify/repair them first.
3. **CLI (this repo, buildable now):** the command + api-client code can be
   written ahead of the adapters against the contract above, with mock-server
   tests, exactly as the `storage` vertical was. It simply returns an API error
   until its adapter exists — same status as every other pending command.

## Prerequisites to verify (cheap probes)

- Is `https://api.changes.afterdarksys.com/v1/auth/me` actually deployed and
  responding? (Docs describe it; the app had a login defect.)
- pmportal's real `/api` routes and auth model.
- Whether `support` is a distinct app or SUP-type changes tickets.

## Open decisions for Ryan

- (Resolved) `support` is its own app → its own adapter, not an alias.
- Build the CLI verticals now against the contract (mock-tested, ready when
  adapters land), or wait until the façade adapters + auth are deployed?
- Priority order across tickets / pm / support once unblocked.
- The three storage GA product decisions still open in UPDATED_TODO.md:
  darkstorage-cli's future as a direct surface, storage metering/billing
  semantics, and how much of Secret Server the CLI exposes.
