# ADSM Customer Infrastructure Backend Integration Plan

Plan date: 2026-09-17

## Goal

Prepare `api.afterdarksys.com` to be the customer control plane used by the
`adsm` CLI for hosted infrastructure, and integrate the existing
`api.darkstorage.io` service as the authoritative storage owner.

The finished system must let a customer authenticate once, select an
organization, inspect their purchased capabilities, and manage only the hosted
resources that organization is entitled to use. Storage operations must produce
the same resources, policy decisions, usage, and audit records whether they are
initiated through ADSM or a supported Dark Storage client.

## Architecture decision

```text
adsm CLI
  |
  | OAuth2 authorization code + PKCE
  | Authorization: Bearer ...
  | X-Organization-ID: <tenant UUID>
  v
api.afterdarksys.com/v1/adsm                 Customer control plane
  |-- principal and membership resolution
  |-- entitlement/capability decision
  |-- resource + operation state
  |-- idempotency + audit + metering
  |-- owner-service adapters
  |
  | audience-bound service credential
  | initiating-actor + tenant + correlation context
  v
api.darkstorage.io/v1                       Storage owner service
  |-- bucket/object metadata
  |-- storage policy and quota enforcement
  |-- STS credential issuance
  |-- MinIO/R2 provider bindings
  |-- storage audit and usage events
  v
/s3 or upstream data plane                  Direct bulk data transfer
```

Rules:

- `api.afterdarksys.com` is the only general customer control-plane endpoint
  used by ADSM.
- `darkstorage.io` remains the source of truth for storage behavior. Do not
  recreate buckets, objects, STS, storage policy, or storage-provider logic in
  the gateway.
- The gateway may return a Dark Storage endpoint, presigned operation, or
  short-lived scoped STS credentials. It must not proxy bulk object data.
- Never forward the customer's gateway access token unchanged to Dark Storage.
  Use OAuth token exchange or an audience-bound service credential that carries
  tenant, initiating actor, operation, correlation, and entitlement context.
- `darkstorage-admin` and `/v1/admin/*` remain operator-only and are never
  exposed through customer ADSM routes.
- After Dark hosted resources use `provider=afterdark`. Physical placement such
  as MinIO, R2, bare metal, AWS, or OCI is private `upstream_provider` or provider
  binding metadata.
- Authentik is the identity authority. `login.afterdarksys.com` may remain the
  public issuer/facade, but no service may create an independent customer
  identity authority.

## Scope

This plan delivers the shared backend needed by the CLI plus one complete
storage vertical slice. It establishes reusable patterns for hosts, containers,
Kubernetes, DNS, CDN, email, telco, secrets, billing, and status without claiming
those owner-service implementations are complete.

In scope:

- CLI OAuth registration, login, token refresh, logout, and organization choice
- canonical principal, tenant, entitlement, request, and error contracts
- entitlement repair and operation-specific capability decisions
- durable resource and asynchronous operation records
- Dark Storage service registration and delegated authentication
- bucket lifecycle, usage/quota, and scoped STS access through ADSM
- inventory projection, metering, billing correlation, audit, and reconciliation
- generated/validated client models and end-to-end tests

Out of scope for this plan:

- exposing Dark Storage administrative APIs to customers
- proxying S3 payloads through `api.afterdarksys.com`
- duplicating the specialist `darkstorage` CLI feature set in ADSM
- implementing every non-storage owner service
- direct database, SSH, OPC-session, or provider-credential access from ADSM

## Required contracts

### Principal and tenant

Every authenticated request resolves this server-owned context:

```json
{
  "subject_id": "uuid-or-stable-provider-subject",
  "tenant_id": "organization-uuid",
  "actor_type": "human|service|service_on_behalf_of_human",
  "initiating_actor": { "subject_id": "...", "actor_type": "human" },
  "effective_service_actor": null,
  "scopes": ["services:read"],
  "operations": ["storage.bucket.read"],
  "policy_revision": "...",
  "entitlement_revision": "...",
  "correlation_id": "uuid"
}
```

- `X-Organization-ID` is only a selection hint. Active membership in the
  selected organization must be proven from the platform database.
- Email address, group name, role label, and request body fields are never tenant
  authority.
- Machine clients use tenant-bound, scope-limited credentials. Human and service
  actors remain separately attributable.

### Capabilities

Initial common operations:

- `platform.context.read`
- `platform.services.read`
- `infrastructure.resource.read`
- `infrastructure.operation.read`
- `storage.bucket.read`
- `storage.bucket.create`
- `storage.bucket.delete`
- `storage.object.read`
- `storage.object.write`
- `storage.object.delete`
- `storage.credentials.issue`
- `storage.credentials.revoke`
- `storage.usage.read`

Capability evaluation input must include principal, tenant, operation, resource
scope, requested quantity, and current entitlement revision. The result must
include allow/deny, reason code, applicable limits, policy revision, entitlement
revision, expiry, and decision ID.

The gateway performs the customer-facing decision. Dark Storage also enforces
the delegated capability and its local resource policy/quota. Either denial wins.
Dependency failure denies mutations and returns an observable service error; it
must not be converted into an empty entitlement list.

### API envelope

Success:

```json
{
  "data": {},
  "meta": {
    "api_version": "v1",
    "request_id": "uuid",
    "correlation_id": "uuid",
    "next_cursor": null
  }
}
```

Error:

```json
{
  "error": {
    "code": "entitlement_denied",
    "message": "This organization cannot create another bucket.",
    "request_id": "uuid",
    "correlation_id": "uuid",
    "docs_url": "https://docs.afterdarksys.com/errors/entitlement-denied",
    "details": {}
  }
}
```

All mutations require `Idempotency-Key`. Mutable resources use revision/ETag
preconditions. Long-running work returns `202 Accepted` with an operation
resource rather than holding the request open.

### Delegated Dark Storage call

Decision: use a short-lived After Dark delegation JWT minted by the central
authorization service after an allow decision. Do not make delivery depend on
Authentik supporting RFC 8693 token exchange. Authentik remains the human and
service identity authority; the delegation JWT is an authorization capability,
not a second login identity.

The gateway authenticates to the authorization service using its Authentik
service identity. The authorization service signs delegation JWTs with a
dedicated, rotatable Ed25519 key and publishes current/overlap public keys at
`/internal/.well-known/delegation-jwks.json`. Dark Storage trusts only this
issuer and JWKS for delegated calls.

Required claims:

- issuer: canonical After Dark issuer
- audience: `api.darkstorage.io`
- subject: gateway service identity
- initiating subject: customer subject
- tenant/organization UUID
- exact operations and optional resource IDs/prefixes
- decision ID and entitlement revision
- correlation ID
- HTTP method, normalized route template, canonical request SHA-256 digest, and
  idempotency key for mutations
- unique `jti`, issued-at, not-before, and expiry of at most 60 seconds

Dark Storage rejects a capability whose audience, tenant, operation, resource,
method, route, body digest, decision revision, or time window does not match the
request. It records `jti` with the owner operation/idempotency record. A replay
is accepted only as retrieval of the identical idempotent result; a changed
request is rejected.

Direct `darkstorage` customer clients obtain an Authentik access token with
audience `api.darkstorage.io`. Dark Storage validates that identity token and
calls the same central authorization/reservation APIs used by the gateway. A
direct customer token is never treated as a delegated gateway capability.

Static shared API keys are permitted only as a temporary migration mechanism and
must be tenant-independent, rotated, stored in Secret Server, restricted at the
network layer, and unable to authorize an operation without signed request
context.

### Internal authorization and reservation API

There is one authoritative policy decision point in `afterdarksys.com`:

- `POST /internal/v1/authorization/decisions`
- `POST /internal/v1/entitlement-reservations`
- `POST /internal/v1/entitlement-reservations/{id}/commit`
- `POST /internal/v1/entitlement-reservations/{id}/release`
- `POST /internal/v1/entitlement-reservations/{id}/consume-delete`

Both the ADSM gateway path and direct Dark Storage path call it. Reservation
requests carry tenant, subject, capability, quantity/unit, resource scope,
idempotency key, and entitlement revision. A reservation has a unique lease ID,
expires automatically if not committed, and can be committed or released only
once. Delete consumption releases resource-count quota only after the owner
confirms deletion. Stale entitlement revisions are rejected and retried after a
fresh decision. Decision and reservation dependency failures deny mutations.

Suspension, membership removal, or removal of
`storage.credentials.issue` blocks new sessions immediately and places all
existing sessions for the affected tenant/subject on the revocation queue.
Dark Storage must deny their S3 use within 60 seconds. Bucket-operation grant
removal blocks new control-plane operations immediately but does not delete
customer data automatically.

### Internal service and event contracts

Phase 0 produces machine-readable contracts in addition to the public OpenAPI:

- `subdomains/api/openapi/darkstorage-internal-v1.yaml` for gateway/owner calls
- `subdomains/api/openapi/authorization-internal-v1.yaml` for decisions and
  reservations
- `subdomains/api/asyncapi/owner-events-v1.yaml` for lifecycle and usage events
- a JSON Schema for delegation JWT claims

Every owner event carries `event_id`, `event_type`, `schema_version`,
`occurred_at`, `tenant_id`, `owner_service`, `owner_resource_id`, optional ADSM
`resource_id`, `operation_id`, `decision_id`, `entitlement_revision`,
`correlation_id`, and an event-specific payload.

Dark Storage writes lifecycle and usage events to its transactional outbox. The
gateway pulls them from `GET /internal/v1/owner-events` using its confidential
Authentik workload identity with audience `api.darkstorage.io` and scope
`owner.events.read`. The gateway persists them into a durable inbox keyed by
owner service + event ID, then acknowledges a cursor. Dark Storage never pushes
events into a public gateway route. Inbox handlers are idempotent and
update resource/provider-binding, inventory, audit, usage, and billing
projections. A replay cursor and snapshot/backfill endpoint repair missed
events, including resources created directly through supported Dark Storage
clients.

Dark Storage uses a separate confidential Authentik workload identity with
audience `api.afterdarksys.com` and scopes `authorization.decide` and
`entitlement.reserve` when calling the central PDP. Client credentials are
stored and rotated through Secret Server, never configuration committed to Git.
Private network policy allows only the named workloads to reach internal routes.
401/403 responses are never retried; token refresh occurs once on 401, while
bounded retries apply only to transport failure, 429, and retryable 5xx results.

Meter events use a stable `meter_event_id`, capability meter key, integer
quantity, unit, measurement window, source resource, and correction reference.
Consumers deduplicate by meter event ID, accept ordered or late delivery within
the declared billing window, and apply corrections as compensating events
rather than overwriting billed history.

## Customer API surface

The first published OpenAPI contract should contain:

| Method and path | Purpose |
|---|---|
| `GET /v1/platform/memberships` | Enumerate organizations before selection |
| `GET /v1/adsm/context` | Return selected tenant, principal, revisions, and safe account summary |
| `GET /v1/adsm/services` | Return entitlement-filtered available services and readiness |
| `GET /v1/adsm/resources` | List customer resources across owner services |
| `GET /v1/adsm/resources/{id}` | Inspect one projected resource |
| `GET /v1/adsm/operations/{id}` | Poll an asynchronous operation |
| `GET /v1/adsm/storage/buckets` | List tenant buckets through the storage adapter |
| `POST /v1/adsm/storage/buckets` | Reserve quota and create a bucket idempotently |
| `GET /v1/adsm/storage/buckets/{name}` | Inspect bucket and storage binding state |
| `DELETE /v1/adsm/storage/buckets/{name}` | Start idempotent bucket deletion |
| `POST /v1/adsm/storage/credentials` | Issue short-lived, bucket/prefix/action-scoped STS credentials |
| `GET /v1/adsm/storage/credentials` | List safe STS session metadata |
| `DELETE /v1/adsm/storage/credentials/{accessKey}` | Revoke a session |
| `GET /v1/adsm/storage/usage` | Return quota and metered usage |

Object listing and small metadata operations may be added to ADSM if needed.
Object upload/download should use S3-compatible tooling, a presigned operation,
or the specialist Dark Storage CLI after credentials are issued.

## Delivery phases

### Phase 0 — contract and ownership freeze

Primary repository: `afterdarksys.com`

Tasks:

- Add `docs/adsm/ARCHITECTURE.md` containing the architecture and trust
  boundaries from this plan.
- Add `subdomains/api/openapi/adsm-v1.yaml` as the authoritative public contract.
- Add the internal authorization, Dark Storage adapter, AsyncAPI owner-event,
  and delegation-claim schemas listed under Required contracts.
- Define the provider schema with `afterdark` plus private
  `upstream_provider`, `region`, `zone`, and `external_id` fields.
- Create a service registry entry for Dark Storage with owner, base URL,
  audience, health endpoint, supported operations, and data-plane endpoint.
- Record an ADR that Dark Storage owns storage and the gateway owns customer
  orchestration.
- Record an ADR selecting the 60-second Ed25519 delegation JWT, including issuer,
  audience, JWKS ownership/rotation, actor/subject semantics, request binding,
  replay behavior, and direct-client authentication.
- Add an Authentik blueprint/configuration artifact for the `adsm-cli`, gateway
  service, and direct Dark Storage audiences; validate it in a disposable realm.
- Add contract linting and breaking-change checks to CI.

Exit gate:

- OpenAPI validates and covers every endpoint in the initial table.
- Provider and service ownership decisions have one canonical definition.
- No route or schema requires a customer token to be forwarded to an owner
  service.
- The reference signer/validator and contract fixtures prove issuance and
  rejection for wrong audience, tenant, operation, resource, request digest,
  expired key, and replayed `jti`. Real Dark Storage validation is a Phase 4
  integration gate.

### Phase 1 — canonical CLI identity and organization context

Primary repository: `afterdarksys.com`; consumer: `adsm-cli`

Backend tasks:

- Register `adsm-cli` as a public OAuth client with authorization code + PKCE
  S256, loopback redirect URIs, refresh rotation, revocation, and
  `offline_access`.
- Keep Authentik as the identity authority and document the public issuer,
  authorization, token, JWKS, introspection, and revocation endpoints.
- Finish provider-subject membership backfill and reject ambiguous tenant
  selection.
- Reuse and harden `createAuthMiddleware` for `/v1/adsm`; require an explicit
  ADSM audience and stop relying on locally synthesized entitlement revisions.
- Standardize request/correlation ID middleware and error envelopes before
  adding resource routes.

CLI tasks:

- Add browser PKCE login with a loopback listener and a separately tested
  device flow only if the issuer supports it.
- Store tokens in the OS keychain. Store only selected organization ID, account
  display value, endpoint, and output preference in YAML.
- Send `X-Organization-ID` on tenant-bound calls and implement `account whoami`,
  organization list/select, refresh, and logout.

Exit gate:

- A new customer can log in, enumerate only their memberships, select one,
  refresh, call `/v1/adsm/context`, revoke, and log out.
- Forged organization IDs, wrong audiences, expired tokens, and disabled
  memberships fail closed.

### Phase 2 — entitlement authority and service catalog

Primary repository: `afterdarksys.com`

Tasks:

- Repair billing entitlement joins to use current `customer_id`, `product_id`,
  and order `phase` fields.
- Migrate legacy integer user grant identifiers to canonical UUID tenant,
  subject, customer, product/capability, and actor references with foreign keys.
- Replace display-string feature matching with the stable capability keys in
  this plan.
- Implement typed limits and reservation/commit/release for bucket count,
  stored bytes, egress, request classes, and STS sessions.
- Build the concrete `/internal/v1/authorization/decisions` and entitlement
  reservation APIs defined above, backed by immutable decision/reservation
  audit records.
- Emit monotonic entitlement revisions and define cache invalidation and maximum
  revocation propagation time.
- Make `/v1/adsm/services` entitlement-filtered and include Dark Storage health
  separately from customer authorization.

Exit gate:

- Tests cover subscription, one-time purchase, explicit grant, suspension,
  revocation, expiry, hard quota, concurrent reservations, and dependency
  failure.
- Provider-neutral authorization vectors define the result for identical
  principal, tenant, capability, resource, quantity, and revision inputs. Real
  gateway/direct Dark Storage parity is a Phase 4 integration gate.
- A customer without `storage.bucket.create` cannot reserve or create a bucket.
- Service availability does not imply authorization.

### Phase 3 — durable ADSM control-plane primitives

Primary repository: `afterdarksys.com/subdomains/api`

Suggested files:

- `routes/adsm.js`
- `middleware/request-context.js`
- `middleware/authorize-operation.js`
- `services/resources.js`
- `services/operations.js`
- `services/idempotency.js`
- `services/outbox.js`
- `adapters/darkstorage.js`
- new migrations after the current migration sequence

Tasks:

- Add tenant-scoped `resources`, `operations`, `provider_bindings`,
  `idempotency_records`, `resource_events`, `outbox_events`, and
  `usage_correlations` tables, plus a durable `owner_event_inbox` and replay
  cursor per owner service.
- Store desired state separately from observed owner/provider state.
- Enforce tenant IDs and unique idempotency constraints in the database.
- Implement worker-owned asynchronous operations and retry-safe adapter calls.
- Bind idempotency records to tenant + route operation + key + canonical request
  digest + resource + expected revision. Claim them atomically; return the
  stored result only for an identical request, return 409 for a changed request,
  return the existing operation for in-progress work, retain mutation records
  for at least the resource recovery window, and propagate the stable operation
  key to the owner service.
- Add resource/operation list and inspect routes with cursor pagination.
- Add immutable gateway audit events containing initiating actor, effective
  service actor, tenant, operation, resource, decision, revisions, correlation,
  and outcome.
- Project resources to inventory with `provider=afterdark`; keep physical
  storage placement in the provider binding.
- Add reconciliation for timeout, partial success, duplicate response, owner
  service outage, and metering lag.
- Consume owner-event contract fixtures idempotently and update ADSM resources,
  provider bindings, inventory, usage/billing, and audit projections. Add
  snapshot/backfill reconciliation for Dark Storage resources created outside
  ADSM.

Exit gate:

- Repeating a mutation with the same idempotency key returns the same operation
  and never duplicates an owner resource.
- Reusing a key with a different payload, resource, operation, or expected
  revision returns 409 and cannot reach an adapter.
- A fake-adapter success followed by gateway failure is recovered by
  reconciliation; real Dark Storage recovery is proved in Phase 4.
- All list/get/update queries include an enforced tenant predicate and negative
  cross-tenant tests.

### Phase 4 — Dark Storage identity, policy, and adapter convergence

Primary repository: `darkstorage.io`; adapter in `afterdarksys.com`

Dark Storage tasks:

- Inventory every schema/query/event path that treats `user_id` as tenant
  authority, including accounts, organization memberships, API keys, buckets,
  objects, STS sessions, audit, analytics, quotas, affinity, and event payloads.
- Add canonical subject and tenant mapping tables plus tenant foreign keys.
  Import linkage only from the authenticated platform membership reconciliation
  API, keyed by `provider=authentik` + immutable provider subject from
  `provider_membership_links`. Match Dark Storage `oauth_provider/oauth_id` to
  that provider subject; never match email. For legacy local-only identities,
  require an Authentik login plus proof of the existing Dark Storage API key or
  live legacy session in a one-time account-claim flow. Accounts with neither
  proof path require audited operator remediation. Personal organizations are
  linked to or replaced by a canonical organization only after that claim is
  approved. Quarantine ambiguity, validate counts/ownership, then
  dual-read/dual-write during a bounded compatibility release before making
  tenant columns mandatory. Supply forward and rollback migrations and
  post-migration validation queries. Cutover requires 100% of active
  resource-owning accounts to be mapped or explicitly quarantined with a support
  owner; no mapping may be inferred from email.
- Replace the protected-route dependency on locally signed customer JWTs with
  canonical issuer/audience verification and delegated service capabilities.
- Stop mapping `tenant_id` to `user_id`; resolve the canonical organization
  membership or validate a signed gateway tenant delegation.
- Accept the canonical organization header/context while retaining
  `X-DarkStorage-Tenant` only during a documented compatibility period.
- Map delegated operations to the existing policy evaluator and enforce bucket,
  prefix, action, byte, expiry, and STS-session limits.
- Replace the telco-named billing URL with the canonical storage metering and
  billing contract.
- Publish storage lifecycle and usage events with stable tenant, resource,
  operation, decision, and correlation identifiers.
- Publish those events transactionally, expose replay/snapshot cursors, and
  preserve compatibility for existing accounts, buckets, API keys, STS
  sessions, audit, analytics, and alternate lookup paths throughout migration.
- Preserve the current audit outbox, tenant-qualified bucket queries, storage
  affinity, S3 gateway, and MinIO/R2 provider behavior.

Gateway adapter tasks:

- Implement bounded timeouts, retry classification, circuit breaking, and
  service-token acquisition without logging credentials.
- Translate the ADSM contract to existing Dark Storage bucket and STS routes.
- Normalize owner errors into the public error envelope without leaking
  upstream details.
- Reserve entitlement quota before creation, commit it after owner success, and
  release it after definitive failure.
- Persist the Dark Storage stable ID/physical name only in provider bindings;
  return the ADSM resource ID to customers.

Exit gate:

- Direct Dark Storage access and ADSM-mediated access resolve the same tenant,
  bucket, policy, limits, audit chain, and usage record.
- A gateway token cannot be replayed directly against Dark Storage.
- A delegated token cannot be used for another tenant, operation, bucket, or
  prefix.
- `darkstorage-admin` continues to require an operator/admin principal.
- Existing migrated customers retain the same buckets and usable authorized
  access; every ambiguous legacy row is quarantined rather than guessed.
- A bucket created or deleted directly through a supported Dark Storage client
  converges into ADSM resources, inventory, usage/billing, and audit exactly
  once.

### Phase 5 — storage vertical slice in the ADSM CLI

Primary repository: `adsm-cli`

Tasks:

- Generate or validate Go types against `adsm-v1.yaml`.
- Update the client for the canonical success/error envelope, organization
  header, request/correlation IDs, idempotency keys, pagination, and ETags.
- Implement:
  - `adsm storage bucket list`
  - `adsm storage bucket get NAME`
  - `adsm storage bucket create NAME`
  - `adsm storage bucket delete NAME`
  - `adsm storage credentials issue`
  - `adsm storage credentials list`
  - `adsm storage credentials revoke ACCESS_KEY`
  - `adsm storage usage`
- Poll asynchronous operations with bounded backoff and support `--no-wait`.
- Keep JSON/YAML stdout parseable; send progress and warnings to stderr.
- Require confirmation or `--yes` for destructive actions without weakening
  server-side idempotency and revision checks.
- Direct users to S3 or `darkstorage` for bulk/specialist object operations.

Exit gate:

- The CLI contains no token in YAML, no embedded service/admin credential, and
  no checked-in binary.
- Golden tests cover table, JSON, and YAML output for success, denial, conflict,
  async progress, partial outage, and quota exhaustion.

### Phase 6 — ecosystem verification and production readiness

Repositories: all three implementation repositories

Tasks:

- Build an isolated sandbox organization, billing customer, product grant,
  storage backend, and test issuer/client.
- Add end-to-end tests for login, organization selection, service discovery,
  create/list/get/delete bucket, STS issue/use/revoke, usage, billing, inventory,
  suspension, entitlement revocation, and reconciliation.
- Add contract tests from ADSM to gateway and gateway to Dark Storage.
- Add negative tests for every cross-tenant identifier and alternate retrieval
  path.
- Add duplicate, reordered, late, replayed, and correction meter-event tests;
  verify deletion quota release and prevent double billing.
- Add SLOs and alerts for auth, entitlement decisions, operation queue age,
  Dark Storage adapter health, reconciliation lag, audit lag, and metering lag.
- Document credential rotation, owner-service outage, stuck operations,
  reconciliation, backup/restore, and break-glass procedures.
- Remove tracked generated binaries from affected repositories and publish only
  checksummed, signed release artifacts with SBOM/provenance.

Production gate:

- A bucket created through ADSM is visible through Dark Storage, inventory,
  usage, billing, and audit with the same stable tenant/resource correlation.
- Retrying every timed-out mutation is safe.
- Suspension or entitlement revocation blocks new access within the documented
  propagation window and prevents affected STS sessions from S3 use within 60
  seconds.
- No test or command needs production SSH, a database shell, an OPC browser
  session, a customer-visible upstream credential, or an admin API key.

## Implementation work packages

Repository/file-level execution plans:

- [`plans/00-contracts-and-delegation.md`](plans/00-contracts-and-delegation.md)
- [`plans/01-identity-and-entitlements.md`](plans/01-identity-and-entitlements.md)
- [`plans/02-control-plane-core.md`](plans/02-control-plane-core.md)
- [`plans/03-darkstorage-tenant-migration.md`](plans/03-darkstorage-tenant-migration.md)
- [`plans/04-darkstorage-adapter-events.md`](plans/04-darkstorage-adapter-events.md)
- [`plans/05-darkstorage-cli-compatibility.md`](plans/05-darkstorage-cli-compatibility.md)
- [`plans/06-adsm-cli-and-e2e.md`](plans/06-adsm-cli-and-e2e.md)

Each execution plan has two bounded tasks with explicit file ownership,
automated verification, and done conditions. The umbrella phases define the
architecture and release gates; the execution plans are the units to implement
and review.

| Package | Depends on | Repositories | Completion evidence |
|---|---|---|---|
| A. OpenAPI/provider/service contracts | none | `afterdarksys.com` | linted contract and ADR |
| B. CLI OAuth and tenant context | A | `afterdarksys.com`, `adsm-cli` | login/membership/revoke E2E |
| C. Entitlement repair and PDP | A | `afterdarksys.com` | migration and decision tests |
| D. Resource/operation/outbox core | B, C | `afterdarksys.com` | idempotency/reconciliation tests |
| E. Dark Storage delegated auth | B, C | `darkstorage.io`, `afterdarksys.com` | token replay/scope negative tests |
| F. Storage adapter and metering | D, E | `afterdarksys.com`, `darkstorage.io` | vertical API integration test |
| G. Dark Storage client compatibility | B, E | `darkstorage-cli`, `darkstorage-admin`, `darkstorage.io` | direct-client and operator-boundary tests |
| H. Customer CLI storage commands | B, F | `adsm-cli` | CLI contract/golden/E2E tests |
| I. Production readiness | all | all | complete production gate |

Packages B and C can proceed in parallel after A. Package D can begin while E
is developed, but F cannot complete until both converge on the same contracts.

## Verification commands

Use the repository-native commands in CI and local development:

```sh
# afterdarksys.com API gateway
cd subdomains/api
npm test -- --runInBand

# Dark Storage API
cd api
go test ./...
go vet ./...

# ADSM CLI
go test ./...
go vet ./...
```

Add a cross-repository sandbox harness that starts the issuer fixture, gateway,
entitlement service, PostgreSQL, Dark Storage API, and MinIO, then runs the ADSM
binary built into a temporary directory. The harness must never rely on a binary
stored in Git.

## Definition of done

The backend is prepared for customer CLI tooling only when:

- the public API contract is versioned and generated/validated by the CLI;
- Authentik-backed login, explicit organization selection, and audience checks
  work end to end;
- entitlements use stable capabilities and typed, concurrency-safe limits;
- resources and mutations are tenant-scoped, idempotent, asynchronous where
  necessary, audited, metered, and reconcilable;
- Dark Storage is reached through delegated service identity and independently
  enforces tenant, capability, policy, and quota;
- object data uses the S3/data plane rather than the general API gateway;
- storage state agrees across ADSM, Dark Storage, inventory, billing, usage, and
  audit views;
- cross-tenant, revoked, suspended, expired, wrong-audience, replay, and outage
  tests fail closed; and
- customer tooling contains no admin credential, provider credential, direct
  production SSH path, or checked-in executable.

## Repository access required for execution

This plan can be maintained from `adsm-cli`, but implementation requires write
access to:

- `/Users/ryan/development/afterdark-meta-project/afterdarksys.com`
- `/Users/ryan/development/afterdark-meta-project/darkstorage.io`
- `/Users/ryan/development/afterdark-meta-project/adsm-cli`

The dedicated `darkstorage-cli` and `darkstorage-admin` repositories are
compatibility/test consumers for this phase; they need changes only where the
canonical identity or contract migration affects them.
