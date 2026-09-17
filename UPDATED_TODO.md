# ADSM Ecosystem Integration TODO

## Implementation update — 2026-09-17

This section supersedes the original audit snapshot below. The storage vertical
slice is now implemented through source and automated component/integration
tests, but it is **not yet production-complete** because deployment identities,
live Secret Server material, and the multi-service sandbox release gate are
external work that cannot be proven from these repositories alone.

### Completed in source

- [x] Published `/v1/adsm`, internal authorization, delegation JWT, and owner
  event contracts with compatibility validation.
- [x] Repaired billing entitlement joins and introduced canonical
  tenant/customer links, stable DarkStorage capabilities, monotonic revisions,
  immutable decisions, and atomic reservation/commit/release accounting.
- [x] Added the durable ADSM resources/operations/provider-bindings/outbox
  control plane with tenant RLS and idempotent mutation acceptance.
- [x] Added the DarkStorage adapter and operation worker with bounded timeouts,
  retry classification, reservation finalization, and Secret Server-backed
  Ed25519 delegation signing.
- [x] Added DarkStorage canonical provider-subject mapping without email
  inference, tenant-pinned API keys/sessions, migration quarantine, rollback,
  and a cutover validator.
- [x] Replaced production local-JWT trust with Authentik audience verification
  plus central entitlement decisions for direct customer mutations.
- [x] Added exact request-bound delegation verification (audience, tenant,
  operation, resource, method, route, body digest, idempotency key, and a
  maximum 60-second lifetime), persistent replay binding, and owner-side
  response idempotency.
- [x] Added transactional bucket and STS owner events, ordered pull/snapshot/ack
  APIs, durable-before-ack gateway ingestion, inventory projection, and
  correlation back to ADSM operations.
- [x] Added customer bucket and scoped STS routes through ADSM; no upstream
  storage credential is stored in ADSM configuration or operation records.
- [x] Implemented `adsm login/logout/account/services`, OS-keychain credential
  storage, refresh/revoke, canonical organization selection, bucket lifecycle,
  STS issue/list/revoke, operation polling, `--no-wait`, revision checks, and
  structured output.
- [x] Enforced separate operator API-key scope on `/v1/admin/*`; customer OIDC,
  ordinary customer keys, and delegation capabilities fail closed.
- [x] Kept the repository-local `adsm` binary ignored and untracked. Verified
  with `test -z "$(git ls-files adsm)"`.
- [x] Added a disposable PostgreSQL migration gate applying all DarkStorage
  migrations through 035, testing subject reconciliation and credential tenant
  pinning, rolling 033–035 back, reapplying them, and running cutover checks.

### Verified

- [x] AfterDark API: 31 suites / 293 tests pass.
- [x] DarkStorage API: `go test ./...` passes.
- [x] ADSM CLI: `go test ./...` passes, including PKCE and request-contract
  tests.
- [x] ADSM OpenAPI/delegation contract lint passes.
- [x] DarkStorage canonical tenant forward/rollback/reapply gate passes.

### Required before calling the ecosystem completely integrated

- [ ] Register the `adsm-cli` public OAuth client and reviewed loopback redirect
  policy in the deployed Authentik tenant; confirm issuer/audience/scope claims
  against real tokens.
- [ ] Provision the two confidential workload clients and their narrow scopes
  (`authorization.decide`, `entitlement.reserve`, `owner.events.read`).
- [ ] Create/rotate the delegation Ed25519 key and both OAuth client secrets in
  Secret Server, mount only their runtime token/secret files, and register the
  public JWK metadata in the control database.
- [ ] Apply billing migration 006, AfterDark API migrations 011–014, and
  DarkStorage migrations 033–035 in staging; run the tenant cutover validator
  and assign a support owner to every quarantine record before enforcement.
- [ ] Deploy network policy allowing only the named gateway and owner workloads
  to reach internal authorization and owner-event routes.
- [ ] Build the live sandbox Compose fixture with Authentik, both PostgreSQL
  databases, AfterDark API/worker, DarkStorage, Secret Server fixture, and
  MinIO. Prove login, bucket lifecycle, STS use/revocation, direct-owner
  convergence, retry/replay, suspension, metering, and backup/restore.
- [ ] Publish storage usage/correction and suspension events for all supported
  object/data-plane paths. Bucket and STS lifecycle events are implemented;
  byte-hour, egress, request-class, and correction meters remain.
- [ ] Replace the legacy telco-named DarkStorage billing client path or remove
  it after all storage billing reads/writes use the canonical entitlement and
  meter projection.
- [ ] Migrate the specialist `darkstorage-cli` from query-parameter bearer
  callbacks/YAML credentials to authorization-code PKCE, canonical organization
  selection, refresh/revoke, and OS-keychain storage. The ADSM client is fixed;
  the specialist client is still compatibility work.
- [ ] Exercise and document the direct DarkStorage client compatibility path
  against the same central PDP, quota, audit, and owner-event flows.
- [ ] Add the canonical `afterdark` provider to the broader inventory registry
  and project ADSM resources there. ADSM's own resource/provider binding is
  implemented; the legacy ecosystem-wide inventory registry is not yet
  migrated.
- [ ] Complete the Secret Server customer-resource adapter. Secret Server is
  currently used for workload/delegation credentials, but ADSM customer
  `secrets` commands and entitlement/event integration are not implemented.
- [ ] Implement the non-storage ADSM owner services and commands listed in the
  historical backlog below; they remain intentionally fail-closed stubs.

### Release blockers that require an explicit product/operations decision

- [ ] Decide whether the specialist DarkStorage CLI remains a supported direct
  customer surface after ADSM GA; if yes, its PKCE/keychain migration is P0.
- [ ] Decide the retention and billing semantics for storage byte-hours,
  request classes, egress, corrections, and late-arriving owner events.
- [ ] Decide whether ADSM exposes only the common Secret Server subset or hands
  off advanced workflows to the `ss` CLI.

---

Executable backend plan: [BACKEND_INTEGRATION_PLAN.md](BACKEND_INTEGRATION_PLAN.md)

Audit date: 2026-09-17
Repositories reviewed:

- `adsm-cli` at commit `c382c44`
- sibling `afterdarksys.com` ecosystem, especially `subdomains/login`, `api`, `billing`, `inventory`, `opc`, `status`, `cdn`, and `sip`
- `darkstorage-cli` at commit `595d803`
- `darkstorage-admin` at commit `88d632b`
- `darkstorage.io` at commit `f4606c8`
- `secretserver.io` at commit `086b727`

## Verdict

**ADSM is a useful CLI shell, but it is not integrated end to end with the After Dark Systems ecosystem or entitlement system yet.**

The intended architecture is sound: ADSM should be the customer/operator CLI for resources provided by After Dark Systems. In inventory and API contracts, After Dark Systems must be represented as a first-class infrastructure provider in the same way that `aws` and `oci` represent external clouds.

Canonical provider decision for this backlog:

- Provider slug: `afterdark`
- Human name: `After Dark Systems`
- `adsm` is the client/product name, not the provider slug.
- `dartnode`, `on_premise`, and individual upstream vendors remain implementation/location metadata. A customer-owned resource delivered by us is still provider `afterdark`.

Do not connect CLI commands directly to internal databases, SSH targets, vendor credentials, or admin-only OPC routes. The CLI should call versioned customer-facing APIs at `https://api.afterdarksys.com`; those APIs authenticate the principal, select a tenant, enforce entitlements, invoke an owning service/provider adapter, and write an audit event.

## What exists today

- The Go CLI has configuration profiles, output rendering, HTTP transport, bearer-token injection, and a fail-closed stub contract.
- Sixteen top-level commands are stubs. `email` has source code for relay-domain and forwarding operations.
- `login.afterdarksys.com` has an OAuth2 authorization-code + PKCE implementation, refresh tokens, revocation, introspection, and `offline_access`, but no registered ADSM public client or CLI login contract was found. Device authorization is not implemented.
- `api.afterdarksys.com` has `/v1/platform/memberships`, `/v1/platform/services`, and `/v1/platform/context`, including canonical subject/tenant context and entitlement revision projection.
- The platform API documentation explicitly says resource adapters, membership backfills, operational contracts, and consumer integration are unfinished.
- Billing has entitlement routes and a product catalog, but its entitlement query layer is not compatible with its current billing schema and is user/customer oriented rather than tenant/resource oriented.
- Inventory has a tenant-scoped infrastructure hierarchy and provider values including `aws` and `oci`, but not `afterdark`. OPC separately uses provider labels including `oci`, `cloudflare`, and `dartnode`, so provider naming is already inconsistent.
- Status, CDN, SIP, Vault/OPC, Kubernetes/OPC, inventory, and billing expose pieces of functionality, but they do not share one customer-facing OAuth audience, entitlement contract, error envelope, tenant-selection contract, or audit contract.
- Dark Storage already has all three major layers: `darkstorage-cli` is the customer client, `darkstorage-admin` is the operator client, and `darkstorage.io` implements the API/backend, S3 gateway, STS, tenant-aware bucket/object services, policy, quota, audit outbox, and MinIO/R2 providers. ADSM must integrate this service rather than create a second storage product.
- Secret Server already has a substantial tenant-aware Go API, Vault-backed storage, OpenAPI document, clients, agents, device authorization, tests, and infrastructure-secret types. Its current platform assessment identifies remaining contract, authorization-path, audit durability, and operational recovery work. This is the owning secrets platform; ADSM must not duplicate it with direct OPC Vault access.
- The ecosystem repository contains no ADSM registration, integration reference, API contract, or end-to-end test.

## Critical defects found in existing components

- Billing entitlement code queries `subscriptions.user_id` and `subscriptions.product_code`, while the current schema defines `subscriptions.customer_id` and `subscriptions.product_id`.
- Billing entitlement code queries `orders.user_id` and `orders.status`, while the orders migration defines `orders.customer_id` and `orders.phase`.
- `user_entitlement_grants.user_id` and `granted_by` are `BIGINT`, but the current billing customer and canonical identity IDs are UUIDs. The authenticated route passes a billing-customer UUID into those queries.
- `user_entitlement_grants` has no foreign keys to canonical customer/subject/product records.
- Entitlement query failures are caught and converted to an empty list, creating a false “no access” result instead of a visible service failure.
- Feature checks use case-insensitive substring matching against display strings instead of stable capability keys.
- Entitlement administration is deliberately fail-closed: `authenticateAdmin` always returns 403. Grants, revocations, freezes, and audit administration are therefore not operational.
- The platform service directory is static, not entitlement-filtered, and omits infrastructure, inventory, status, DNS, CDN, email, storage, SIP/telco, secrets, and Kubernetes owner services.
- Inventory rejects `provider: "afterdark"` because that provider is absent from its allowlist.
- The implemented `email` source bypasses OAuth, tenant selection, entitlements, central audit, and API contracts by SSHing directly to a fixed relay host.
- Dark Storage currently has its own login/token lifecycle and local JWT trust rather than canonical After Dark issuer, audience, tenant, and entitlement decisions.
- The Dark Storage console callback returns a bearer token in a URL query parameter, and `darkstorage-cli` persists credentials in `~/.darkstorage/config.yaml`; this must converge on PKCE/device authorization and keychain-backed storage.
- The Dark Storage billing client targets a telco-named internal billing route. Storage usage, quotas, and billing require a service-specific canonical contract.
- The local `adsm` executable is correctly ignored and is **not tracked by Git**, but it is stale relative to source. Build/release workflows must never depend on a repository-local binary.
- Repository hygiene is inconsistent elsewhere: `darkstorage-admin` tracks a root `darkstorage-admin` executable, and `darkstorage.io` tracks `api/cmd/darkstorage/darkstorage`, `api/darkstorage`, `api/darkstoraged`, and `cli/darkstorage`. These generated binaries should be removed from the Git index and covered by explicit ignore rules without disturbing source or release artifacts.
- The repository pins Go `1.25.7`, which the configured asdf environment does not currently provide. Verification through the default `go` shim fails before compilation.
- There are no CLI tests, generated API models, OpenAPI conformance tests, release workflows, or end-to-end ecosystem tests.

## P0 — establish the contracts before wiring commands

### 1. Make `afterdark` a first-class infrastructure provider

- [ ] Add `afterdark` to the canonical provider registry used by inventory, API schemas, filters, UI choices, tests, telemetry, billing meters, and provisioning jobs.
- [ ] Replace per-service hard-coded provider arrays with one versioned provider contract or generated package.
- [ ] Define provider semantics:
  - `provider=afterdark` means the resource is sold/managed by After Dark Systems.
  - `upstream_provider` records `aws`, `oci`, `dartnode`, `on_premise`, or another implementation venue when relevant.
  - `location`, `region`, `zone`, and `external_id` remain separate fields.
- [ ] Migrate existing customer-facing resources currently labeled only `dartnode`, `on_premise`, `oci`, or `other` when After Dark Systems is the actual provider of record.
- [ ] Add database constraints and tests ensuring every provisioned ADSM resource has `tenant_id`, provider, stable resource ID, owner service, lifecycle state, and entitlement/billing references.
- [ ] Add `afterdark` provider fixtures covering host, VM, container, Kubernetes cluster, storage, CDN, DNS, email, and telco resources.

Acceptance criteria:

- `GET` inventory queries accept `provider=afterdark`.
- A customer sees After Dark resources independently of their upstream hosting venue.
- An upstream migration from OCI to AWS does not change the customer-facing resource identity or provider.

### 2. Define and publish the ADSM API contract

- [ ] Create an OpenAPI 3.1 contract for `/v1/adsm` (or explicitly extend `/v1/platform`) at `api.afterdarksys.com`.
- [ ] Use one response envelope, pagination model, correlation ID, error schema, optimistic-concurrency model, and idempotency contract across owner services.
- [ ] Standardize errors to at least: `code`, `message`, `request_id`, `correlation_id`, optional `docs_url`, and safe structured `details`.
- [ ] Require `Idempotency-Key` for every create, update, action, and delete that can reach a provider.
- [ ] Require revision/ETag preconditions for mutable resources where lost updates are possible.
- [ ] Define async operation resources for provisioning, deletion, restore, transfer, purge, and other long-running actions.
- [ ] Generate Go request/response models or add contract tests so CLI types cannot silently drift from the server.
- [ ] Add API compatibility/versioning policy and deprecation headers.

### 3. Build CLI-native OAuth and tenant selection

- [ ] Register a first-party public OAuth client named `adsm-cli`; do not give it a client secret.
- [ ] Implement Authorization Code + PKCE S256 using an RFC 8252 loopback callback and system browser.
- [ ] Update provider redirect-URI validation to safely support the loopback port chosen by the CLI, or allocate and document a fixed callback port with collision handling.
- [ ] Request only reviewed scopes, initially `openid profile email offline_access platforms services:read` plus narrowly defined ADSM resource scopes.
- [ ] Store refresh/access credentials in the OS keychain; never in `~/.adsm/config.yaml`.
- [ ] Implement refresh-token rotation, expiry handling, revocation, `adsm logout`, and `adsm account whoami`.
- [ ] Use `/v1/platform/memberships` to enumerate organizations and persist only the selected organization ID per profile.
- [ ] Send `X-Organization-ID` on tenant-bound calls and fail with a clear selection prompt when membership is ambiguous.
- [ ] Finish canonical provider-subject membership backfill/enrollment; no email or group inference.
- [ ] Add non-browser/headless login only after a reviewed device-authorization endpoint exists. Do not emulate device flow with copied bearer tokens.

Acceptance criteria:

- A clean machine can log in, select an organization, refresh silently, call context, log out, and revoke its session.
- Cross-tenant organization IDs fail closed.
- Tokens for `billing`, `platforms`, or another audience cannot be replayed against an unrelated owner service.

### 4. Repair and redesign entitlement authority

- [ ] Choose one authoritative entitlement read model. Billing may own commercial facts, but authorization decisions must consume a stable tenant/subject/capability projection rather than query billing tables ad hoc.
- [ ] Replace legacy `BIGINT user_id` assumptions with canonical UUID `tenant_id`, `subject_id`, and billing `customer_id` links.
- [ ] Correct subscription/order joins to use `customer_id`, `product_id`, and `phase`; add migration and data reconciliation tests.
- [ ] Add foreign keys for entitlement grants, product/capability mappings, grant actor, and tenant/customer ownership.
- [ ] Replace display-string feature matching with stable capability keys such as `compute.host.read`, `compute.host.create`, `dns.zone.write`, and `email.alias.write`.
- [ ] Model limits as typed meters with unit, window, hard/soft behavior, and atomic reservation/commit/release operations.
- [ ] Define precedence for subscription, purchase, free tier, admin grant, suspension, revocation, expiry, and support override.
- [ ] Stop converting entitlement database failures into empty entitlement lists. Return a fail-closed, observable dependency error.
- [ ] Implement central, operation-specific administrative authority for grants/revocations; remove the permanent-403 placeholder only when that authority is connected.
- [ ] Emit a monotonic `entitlement_revision` whenever effective access changes and invalidate API/session caches.
- [ ] Provide an internal authorization endpoint that evaluates principal + tenant + operation + resource scope + requested quantity and returns a short-lived decision/capability.
- [ ] Add immutable audit events for decisions and mutations, including initiating actor, effective service actor, tenant, resource, operation, decision reason, policy revision, entitlement revision, correlation ID, and outcome.

Acceptance criteria:

- Every ADSM operation is denied without an explicit capability, even if the user has a broad role name.
- Suspension/revocation becomes effective within a documented maximum propagation time.
- Concurrent quota reservations cannot exceed a hard limit.
- Entitlement dependency failure never becomes accidental access or a misleading empty success.

## P0 — build the backend control plane

### 5. Add an ADSM API façade and owner-service adapters

- [ ] Add authenticated `/v1/adsm` routes to `api.afterdarksys.com` rather than exposing OPC admin sessions or vendor APIs directly.
- [ ] For every route, resolve canonical principal and tenant, evaluate entitlement, call the owning service with a narrow delegated service capability, project safe fields, and append audit.
- [ ] Never forward the user's gateway token unchanged to a different OAuth audience.
- [ ] Add service-to-service identity with audience-bound credentials and rotation. Replace broad shared service keys over time.
- [ ] Add adapter health/readiness that distinguishes API health, entitlement dependency health, and provider connectivity.
- [ ] Add per-tenant rate limits and per-operation abuse controls.
- [ ] Add reconciliation jobs so API state, provider state, inventory state, and billing/metering state converge after partial failures.

### 6. Create the provisioning and operation model

- [ ] Create durable `resources`, `operations`, `provider_bindings`, and `resource_events` contracts (or extend inventory with clearly separated desired/observed state).
- [ ] Define lifecycle states such as `requested`, `provisioning`, `active`, `suspending`, `suspended`, `deleting`, `deleted`, `failed`, and `needs_attention`.
- [ ] Store desired state separately from observed provider state.
- [ ] Implement an outbox/worker pattern for provider side effects; HTTP request handlers must not be the sole owner of long-running work.
- [ ] Make provider adapters idempotent and reconcile by stable external ID.
- [ ] Record cost, billable usage, quota reservation, and entitlement reference on each customer resource.
- [ ] Support compensating actions and operator-visible recovery for partial provisioning failures.
- [ ] Publish resource and operation events to the existing ecosystem event backbone.

## P1 — backend required for each CLI command

| CLI area | Existing reusable pieces | Required backend/API work before connection |
|---|---|---|
| `login` | Login OAuth2 authorization code, PKCE, refresh, revoke; platform membership API | Register public client, loopback redirect policy, scopes, canonical membership backfill, keychain-oriented session contract |
| `account` | `/v1/platform/memberships` and `/context` | Account/profile endpoint, organization selection/default, safe settings mutations, entitlement summary |
| `services` | Static `/v1/platform/services`; billing ecosystem catalog | Entitlement-filtered service catalog with deployment/readiness state and owner-service links |
| `endpoints` | Inventory resource URLs and static service origins | Tenant-scoped endpoint registry, DNS/TLS state, exposure policy, secret-safe projections |
| `status` | Public status PHP endpoints and service checks | Versioned JSON API, account/resource impact projection, incidents/maintenance/history, consistent auth and errors |
| `api` | API-key control tables/routes exist for admin/control use | Customer self-service scoped API keys or OAuth service clients, one-time secret return, rotation/revoke, tenant binding, audit |
| `billing` | Customer, subscription, invoice, payment, order, and public catalog routes | Repair auth/schema ownership, add CLI-safe customer endpoints, entitlement summary, usage/cost breakdown, Stripe portal handoff |
| `stats` | Billing usage tables, inventory cost fields, analytics/status data | Unified tenant usage/meter API with time ranges, dimensions, pagination, late-data semantics, and billing reconciliation |
| `hosts` | Tenant inventory hierarchy and telemetry host data | `afterdark` resources, allocation/provision/deprovision actions, capacity projections, ownership joins, operations API |
| `containers` | Inventory types, Harbor, Kubernetes/OPC fragments | Tenant runtime service for list/inspect/logs/start/stop/restart/deploy, workload isolation, entitlement and quota enforcement |
| `k3s` | OPC Kubernetes routes and inventory cluster hierarchy | Customer-safe cluster adapter, scoped kube access issuance, namespace/workload policy, async operations, no OPC session dependency |
| `storage` | `darkstorage.io` implements `api.darkstorage.io/v1` bucket/object operations, S3 data plane, scoped STS credentials, policy, quota, audit, and admin APIs; dedicated customer/admin CLIs already exist | Validate and harden the existing backend, converge it on canonical After Dark OAuth/tenant identity and entitlements, then expose a thin ADSM adapter. Reuse Dark Storage; do not create a parallel storage backend. |
| `cdn` | CDN site has file upload/list/delete and stats | Replace legacy/static upload auth with canonical OAuth/tenant model; add distributions, origins, domains, purge jobs, entitlement, audit, `afterdark` inventory |
| `dns` | DNS Science ecosystem and provider tooling exist | Define authoritative DNS owner API for zones/records/enrollment; optimistic concurrency, DNSSEC, ownership verification, provider adapters, entitlement/audit |
| `email` | Go source manages Postfix maps over SSH; OPC notification/email pieces | Build tenant-scoped mail-domain/alias service and relay agent/API; move SSH and Postfix credentials server-side; verify domain ownership; quota, audit, DKIM/DNS state, idempotency |
| `telco` | SIP service and Telnyx/OPC messaging fragments | Migrate to canonical OAuth/tenant identity; create customer trunks/numbers/routes API; regulatory controls, usage meters, spend limits, webhook reconciliation |
| `secrets` | `secretserver.io` has a tenant-aware Go API, OpenAPI, Vault-backed secret CRUD, agents/device flow, SDKs, audit and many secret types; OPC also has internal Vault routes | Make Secret Server the owner service. Repair/verify its remaining platform contract and authorization/audit issues, add After Dark canonical identity + entitlement exchange, then expose a thin ADSM adapter. Do not route customers through OPC Vault or create a second secret store. |

For every row above:

- [ ] Add OpenAPI operations and stable capability keys.
- [ ] Add provider adapter and inventory synchronization where resources are created.
- [ ] Add entitlement decision, quota reservation, metering, and billing linkage.
- [ ] Add tenant isolation, negative authorization, idempotency, audit, and provider-failure tests.
- [ ] Only then replace the corresponding CLI `notImplemented` body.

## P1 — fix the email exception before shipping

- [ ] Do not ship the current direct-SSH implementation as a public end-user command.
- [ ] Move Postfix map reads/writes and reloads behind an authenticated owner service or relay-side agent.
- [ ] Remove public CLI flags that allow arbitrary relay host and Postfix instance selection, or restrict them to a separately named break-glass operator tool.
- [ ] Bind every domain/alias to a tenant and entitlement.
- [ ] Require domain ownership verification before activation.
- [ ] Add API-side plan/apply semantics, idempotency keys, revisions, durable audit, safe rollback, and reconciliation.
- [ ] Ensure mutation output respects `--output json|yaml`; human plan/status text must not corrupt structured stdout.
- [ ] Add integration tests against an ephemeral Postfix fixture, not a production relay.

## P1 — integrate the existing Dark Storage ecosystem

- [x] Identify the existing topology: customer CLI at `/Users/ryan/development/darkstorage-cli`, operator CLI at `/Users/ryan/development/darkstorage-admin`, and API/backend at `/Users/ryan/development/afterdark-meta-project/darkstorage.io`.
- [ ] Treat `darkstorage.io` and its API/S3 contracts as the source of truth for storage behavior and S3 compatibility.
- [ ] Keep `darkstorage-admin` and `/v1/admin/*` operator-only. Never surface them through customer ADSM commands without a separately authenticated staff role and explicit audit policy.
- [ ] Register Dark Storage as the authoritative storage adapter in the After Dark control plane; do not duplicate bucket/object implementation in `afterdarksys.com`.
- [ ] Implement/verify the API-first routes already consumed by Dark Storage: bucket CRUD/list, object list/get/put/delete/copy/move, multipart/recursive support, and scoped STS issue/list/revoke.
- [ ] Replace Dark Storage's standalone/local JWT validation with canonical issuer verification, audience checks, JWKS rotation, and the shared subject/tenant model.
- [ ] Replace long-lived token/API-key persistence in `~/.darkstorage/config.yaml` with the same canonical OAuth/keychain model used by ADSM, or document a narrowly scoped compatibility migration.
- [ ] Replace the console callback that returns a bearer token in the URL query with authorization code + PKCE or a reviewed device flow.
- [ ] Bind Dark Storage accounts and buckets to canonical After Dark `tenant_id` and `subject_id`; do not infer identity from email.
- [ ] Map billing products to stable storage capabilities and meters: bucket count, stored byte-hours, egress bytes, request classes, retention, replication/DR, and STS permissions.
- [ ] Replace the telco-named billing endpoint in the Dark Storage billing client with the canonical storage usage/metering contract.
- [ ] Implement atomic quota reservation and usage publication to billing/entitlements.
- [ ] Project every bucket as `provider=afterdark`, `type=storage` in inventory while retaining the physical S3/MinIO/upstream placement as private binding metadata.
- [ ] Add delegated service authentication between `api.afterdarksys.com` and `api.darkstorage.io`; never forward an ADSM bearer token to an audience that did not issue it.
- [ ] Add ADSM storage routes that adapt the Dark Storage contract without forking storage semantics.
- [ ] Keep advanced Dark Storage features in the dedicated `darkstorage` CLI; ADSM should cover the common ecosystem lifecycle and provide a documented handoff for specialist operations.

Acceptance criteria:

- An ADSM-created bucket appears identically through ADSM, Dark Storage, S3 STS, inventory, usage, and billing views.
- Deleting or suspending through either supported control surface converges without orphaned credentials or double metering.
- Cross-tenant buckets and STS sessions are never enumerable or usable.

## P1 — integrate the existing Secret Server ecosystem

- [ ] Treat `secretserver.io` as the source of truth for customer secrets, keys, certificates, agents, variables, and credential lifecycle.
- [ ] Complete the P0 findings already recorded in `secretserver.io/docs/PLATFORM_ASSESSMENT_2026-09-16.md`: consumer/server contract consistency, alternate retrieval-path authorization, real versus placeholder features, durable audit, and recovery evidence.
- [ ] Select the supported Secret Server API version and generate/validate ADSM models from `secretserver.io/docs/openapi.yaml`.
- [ ] Add canonical After Dark identity federation: explicit subject/tenant linkage, audience-bound exchange or delegation, and no email/default-tenant guessing.
- [ ] Map billing products to stable Secret Server capabilities and limits such as `secret.read`, `secret.write`, `secret.rotate`, `certificate.issue`, agent count, secret count, and audit retention.
- [ ] Make entitlement revision changes invalidate Secret Server sessions/capabilities within a documented bound.
- [ ] Add an audience-bound service credential for the ADSM façade and project only customer-safe operations; do not reuse OPC session-held Vault tokens.
- [ ] Decide whether the ADSM `secrets` command is a thin common subset or an install/handoff to the specialist `ss` CLI. Avoid implementing two divergent secret-management CLIs.
- [ ] Reuse Secret Server's device/agent infrastructure only for its intended workload/device use cases; ADSM human login remains the central After Dark OAuth flow.
- [ ] Add correlated audit linkage across ADSM request, entitlement decision, Secret Server audit event, and any Vault/HSM operation without logging secret material.
- [ ] Add integration tests for cross-tenant denial, expired/revoked grants, alternate retrieval paths, secret versioning, rotation failure, suspension, and dependency outage.

Acceptance criteria:

- A secret created through the supported ADSM subset is visible through Secret Server with the same tenant, stable ID, policy, version history, and audit trail.
- ADSM never stores a secret value or Vault token in its config or logs.
- Revocation/suspension blocks both direct Secret Server and ADSM-mediated access within the declared propagation window.

## P1 — API security and operational requirements

- [ ] Enforce tenant ownership at the database layer where possible, not only in route filters.
- [ ] Return 404 for foreign resource identifiers when existence must not be disclosed.
- [ ] Separate user-readable secrets from credentials used by provider adapters.
- [ ] Redact tokens, provider errors, secret values, payment data, phone data, and customer content from logs.
- [ ] Add per-operation scopes and resource ceilings; never authorize from a role/group string alone.
- [ ] Add audit retention, export, integrity controls, and operator search.
- [ ] Add request timeout, retry, circuit-breaker, and idempotency policies per provider.
- [ ] Define SLOs and alerts for identity, entitlement decisions, provisioning queues, reconciliation lag, and metering lag.
- [ ] Add disaster-recovery procedures for control-plane databases and provider bindings.
- [ ] Add a sandbox organization/provider mode so CLI integration tests never mutate production resources.

## P2 — wire and harden the CLI after APIs are ready

- [ ] Add `internal/auth` with OS-keychain storage and injectable test implementation.
- [ ] Construct the API client in the root command with the current profile token provider and organization selection.
- [ ] Add request IDs, correlation IDs, idempotency keys, retry classification, pagination, timeouts, and cancellation.
- [ ] Generate or validate client models against the published OpenAPI contract.
- [ ] Define actual subcommands and flags for every top-level noun; top-level nouns should show help rather than perform ambiguous mutations.
- [ ] Keep stdout machine-readable. Send progress, warnings, and plans to stderr when structured output is selected.
- [ ] Add `--yes`/confirmation rules for destructive actions while retaining server-side idempotency and revision checks.
- [ ] Add shell completion from live/static API schemas without leaking credentials.
- [ ] Add unit tests for config, API errors, output, token refresh, tenant selection, and all command argument validation.
- [ ] Add contract tests using mock owner services and golden JSON/YAML output.
- [ ] Add end-to-end tests against the ecosystem sandbox for login, tenant selection, entitlement denial, provision/list/delete, usage, suspension, and revocation.

## P2 — build and release hygiene

- [ ] Keep `/adsm`, `/adsm_cli`, and `/dist` ignored. The current local `adsm` file is ignored and untracked; do not add it with force.
- [ ] Remove the tracked `darkstorage-admin` executable from the `darkstorage-admin` Git index and add a root-anchored ignore rule for it.
- [ ] Remove tracked generated executables from `darkstorage.io` (`api/cmd/darkstorage/darkstorage`, `api/darkstorage`, `api/darkstoraged`, and `cli/darkstorage`) and add precise ignore rules that do not hide Go source directories.
- [ ] Remove any binary from Git history only if repository policy requires history rewriting; normal forward cleanup does not currently need this because `git ls-files adsm` is empty.
- [ ] Resolve the Go version policy: install the pinned `1.25.7` toolchain or pin a supported version that CI and developers can obtain.
- [ ] Make `build.sh` fail with an explicit version/help message when the required compiler is unavailable.
- [ ] Add CI for `go test ./...`, `go vet ./...`, formatting, vulnerability scanning, OpenAPI compatibility, and cross-compilation.
- [ ] Publish checksummed, signed release artifacts for macOS/Linux/Windows and generate an SBOM/provenance record.
- [ ] Embed semantic version, commit, build date, and API compatibility version.
- [ ] Add install/upgrade/uninstall documentation and a package-manager path.

## Integration test gate for “completely integrated”

Do not call ADSM completely integrated until an automated test proves all of the following in a non-production environment:

- [ ] Fresh install contains no repository-local credentials or binary dependency.
- [ ] Browser PKCE login succeeds and tokens land only in the OS keychain.
- [ ] Multi-organization selection is explicit and a forged organization header is denied.
- [ ] `account`, `services`, `status`, `billing`, and `stats` return tenant-correct data.
- [ ] A permitted `afterdark` resource can be created, observed in inventory, metered, billed, and deleted.
- [ ] The same operation is denied without its entitlement and when quota is exhausted.
- [ ] Retrying a timed-out mutation with the same idempotency key creates only one provider resource.
- [ ] Provider success followed by API failure is recovered by reconciliation without double billing.
- [ ] Suspension/revocation prevents new mutations and appropriately handles existing resources.
- [ ] Every mutation has a correlated identity, entitlement decision, operation, provider, inventory, metering, billing, and audit record.
- [ ] Cross-tenant resource IDs, histories, endpoints, logs, and secrets are not disclosed.
- [ ] JSON and YAML output remain parseable for successes, denials, async progress, and errors.
- [ ] No command requires direct database access, production SSH access, OPC browser session cookies, or vendor credentials on the user's machine.

## Recommended implementation order

1. Canonical `afterdark` provider contract and API/OpenAPI conventions.
2. ADSM OAuth public client, membership convergence, tenant selection, and CLI keychain auth.
3. Entitlement schema repair and central capability decision API.
4. ADSM API façade, async operations, audit, inventory, metering, and reconciliation primitives.
5. Read-only commands: `account`, `services`, `status`, `billing`, `stats`, `hosts`, `endpoints`.
6. One vertical mutation slice (recommended: create/list/delete an `afterdark` compute resource) to prove the whole control plane.
7. DNS, storage, CDN, email, containers/Kubernetes, secrets, and telco owner services.
8. Full CLI command wiring, ecosystem sandbox E2E gate, signed release pipeline.

## Audit evidence index

- CLI scaffold and explicit missing auth/tests: `README.md`
- Stub command implementations: `internal/cmd/*.go`
- Direct relay SSH implementation: `internal/cmd/email.go`, `internal/mailmap/mailmap.go`
- API transport/error assumptions: `internal/api/client.go`
- Provider-owned platform foundation and documented unfinished phases: `afterdarksys.com/subdomains/api/PLATFORM_API.md`
- Static platform directory: `afterdarksys.com/subdomains/api/routes/platform.js`
- OAuth flows/scopes: `afterdarksys.com/subdomains/login/oauth2/server.js`
- Canonical membership links: `afterdarksys.com/subdomains/api/migrations/009_provider_membership_links.sql`
- Billing entitlement schema/code mismatch: `afterdarksys.com/subdomains/billing/migrations/005_entitlements.sql`, `src/db/schema.sql`, `src/services/entitlements.ts`
- Billing admin authority placeholder: `afterdarksys.com/subdomains/billing/src/middleware/auth.ts`
- Inventory provider allowlist/hierarchy: `afterdarksys.com/subdomains/inventory/lib/resource-model.js`, `migrations/005_infrastructure_hierarchy.sql`
- Existing internal infrastructure adapters: `afterdarksys.com/subdomains/opc/routes/infrastructure.js`, `kubernetes.js`, `vault.js`
- Existing service fragments: `afterdarksys.com/subdomains/status/API.md`, `subdomains/cdn/server.js`, `subdomains/sip/src/routes/*.js`
- Dark Storage client/API contracts: `darkstorage-cli/cmd/client.go`, `cmd/storage.go`, `cmd/sts.go`, `AUTHENTICATION_ARCHITECTURE.md`, `IMPLEMENTATION_STATUS.md`
- Dark Storage operator contract: `darkstorage-admin/cmd/client.go`, `cmd/users.go`, `cmd/backends.go`, `cmd/pool.go`, `cmd/services.go`, `cmd/stats.go`
- Dark Storage API/backend: `darkstorage.io/api/internal/server/server.go`, `api/internal/bucket`, `api/internal/object`, `api/internal/sts`, `api/internal/policy`, `api/internal/quota`, `api/internal/audit`, `api/internal/billing/client.go`
- Secret Server platform/API contracts: `secretserver.io/docs/openapi.yaml`, `docs/PLATFORM_ASSESSMENT_2026-09-16.md`, `docs/VALIDATION_2026-09-17.md`, `internal/api`
