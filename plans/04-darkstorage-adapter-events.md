# Plan 04 — Dark Storage Adapter, Events, and Metering

Owners: `afterdarksys.com` and `darkstorage.io`

Depends on: Plans 02 and 03

## Task 1: Implement the bounded gateway adapter

Files:

- `afterdarksys.com/subdomains/api/adapters/darkstorage.js`
- `afterdarksys.com/subdomains/api/workers/adsm-operations.js`
- `afterdarksys.com/subdomains/api/adapters/darkstorage.test.js`
- `afterdarksys.com/subdomains/api/workers/adsm-operations.test.js`

Actions:

- Acquire request-bound delegation capabilities and map ADSM bucket/STS
  operations to Dark Storage.
- Apply bounded timeouts, safe retry classification, circuit breaking, error
  normalization, quota reservation commit/release, and stable operation keys.
- Store only owner stable identifiers in provider bindings and expose ADSM
  resource IDs to customers.

Verify:

```sh
cd /Users/ryan/development/afterdark-meta-project/afterdarksys.com/subdomains/api
npm test -- --runInBand adapters/darkstorage.test.js workers/adsm-operations.test.js
```

Done when: an adapter retry cannot duplicate a bucket, reservation, or billable
event.

## Task 2: Publish and consume lifecycle/meter events

Files:

- `darkstorage.io/database/migrations/035_owner_event_outbox.sql`
- `darkstorage.io/api/internal/events/outbox.go`
- `darkstorage.io/api/internal/events/outbox_test.go`
- `darkstorage.io/api/internal/handlers/owner_events.go`
- `darkstorage.io/api/internal/handlers/owner_events_test.go`
- `darkstorage.io/api/internal/billing/client.go`
- `afterdarksys.com/subdomains/api/workers/darkstorage-events.js`
- `afterdarksys.com/subdomains/api/workers/darkstorage-events.test.js`

Actions:

- Emit versioned bucket, STS, usage, suspension, and correction events in the
  same database transaction as owner state changes.
- Expose pull-only `/internal/v1/owner-events` and snapshot endpoints. Require the
  gateway confidential workload identity with audience `api.darkstorage.io` and
  scope `owner.events.read`; acknowledge durable cursors only after inbox commit.
- Store and rotate both workload credentials through Secret Server. Network
  policy permits only the named services to reach internal routes. Refresh once
  on 401; never retry 401/403; bound retries for transport/429/retryable 5xx.
- Replace the telco-named billing path with the canonical storage meter contract.
- Deliver/replay events to the gateway and reconcile snapshots for resources
  created directly through supported Dark Storage clients.

Verify:

```sh
cd /Users/ryan/development/afterdark-meta-project/darkstorage.io/api
go test ./internal/events ./internal/handlers ./internal/billing
cd /Users/ryan/development/afterdark-meta-project/afterdarksys.com/subdomains/api
npm test -- --runInBand workers/darkstorage-events.test.js services/reconciliation.test.js
```

The integration test owns real direct-create/delete convergence; fixture-only
projection behavior remains in Plan 02.

Done when: no supported owner-side mutation is invisible to the customer control
plane.
