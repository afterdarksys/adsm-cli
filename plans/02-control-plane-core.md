# Plan 02 — Durable ADSM Control-Plane Core

Owner: `afterdarksys.com/subdomains/api`

Depends on: Plans 00 and 01

## Task 1: Add resources, operations, and strict idempotency

Files:

- `migrations/013_adsm_control_plane.sql`
- `routes/adsm.js`
- `services/resources.js`
- `services/operations.js`
- `services/idempotency.js`
- `services/outbox.js`
- `services/idempotency.test.js`
- `services/operations.test.js`

Actions:

- Add tenant-scoped resources, provider bindings, operations, events, outbox,
  usage correlation, idempotency, and owner-inbox tables.
- Separate desired from observed state and run provider work from durable jobs.
- Atomically bind idempotency to tenant, operation, key, canonical request
  digest, resource, and expected revision. Return 409 for mismatched reuse.

Verify:

```sh
cd /Users/ryan/development/afterdark-meta-project/afterdarksys.com/subdomains/api
npm test -- --runInBand services/idempotency.test.js services/operations.test.js
```

Done when: public handlers never own long-running side effects and every
mutation can be retried safely.

## Task 2: Add the owner-event inbox and projections

Files:

- `services/owner-event-inbox.js`
- `services/reconciliation.js`
- `services/projections/inventory.js`
- `services/projections/metering.js`
- `services/projections/audit.js`
- `migrations/014_owner_event_inbox.sql`
- `services/owner-event-inbox.test.js`
- `services/reconciliation.test.js`

Actions:

- Authenticate delivery, persist before processing, deduplicate by owner +
  event ID, and advance replay cursors transactionally.
- Project lifecycle and meter events into resources/bindings, inventory, audit,
  and usage/billing.
- Add snapshot/backfill reconciliation for missed and pre-existing owner state.

Verify:

```sh
cd /Users/ryan/development/afterdark-meta-project/afterdarksys.com/subdomains/api
npm test -- --runInBand services/owner-event-inbox.test.js services/reconciliation.test.js
```

Use contract fixtures to prove duplicate, reordered, late, replayed, correction,
and deletion projection behavior. Real Dark Storage-originated convergence is
owned by Plan 04.

Done when: versioned owner-event fixtures converge to one control-plane state;
Plan 04 supplies the real producer/consumer integration.
