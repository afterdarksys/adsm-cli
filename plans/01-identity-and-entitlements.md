# Plan 01 — Customer Identity, Tenant Context, and Entitlements

Owner: `afterdarksys.com`; consumer: `adsm-cli`

Depends on: Plan 00

## Task 1: Complete CLI identity and membership

Files:

- `subdomains/api/middleware/auth.js`
- `subdomains/api/routes/memberships.js`
- `subdomains/api/routes/adsm.js`
- `subdomains/api/migrations/012_adsm_identity_context.sql`
- `subdomains/api/routes/adsm.test.js`
- `subdomains/api/routes/memberships.test.js`

Actions:

- Register authorization-code + PKCE, refresh rotation, revocation, and loopback
  redirects for a public client.
- Finish canonical provider-subject membership backfill.
- Require the ADSM audience and treat `X-Organization-ID` only as a selection
  that must match active membership.
- Add shared request/correlation ID and error-envelope middleware.
- Consume the validated ADSM/gateway/Dark Storage client blueprint from Plan 00
  while implementing membership and customer login behavior.

Verify:

```sh
cd /Users/ryan/development/afterdark-meta-project/afterdarksys.com/subdomains/api
npm test -- --runInBand routes/memberships.test.js routes/adsm.test.js middleware/auth.test.js
```

The sandbox E2E in the final plan owns real login/refresh/revoke browser flow.

Done when: `/v1/adsm/context` returns one proven principal/tenant context and no
email, role, or request-body value can select a tenant.

## Task 2: Repair the entitlement model and expose one PDP

Files:

- `subdomains/billing/src/services/entitlements.ts`
- billing entitlement migrations/schema/tests
- `subdomains/api/routes/internal-authorization.js`
- `subdomains/api/services/entitlement-reservations.js`

Actions:

- Migrate grants to canonical UUID tenant/subject/customer/product/actor keys and
  correct subscription/order joins.
- Replace display feature matching with stable operation keys and typed limits.
- Implement decision plus reserve/commit/release/delete-consume APIs with
  monotonic revisions and immutable audit.
- Set revocation behavior: new operations deny immediately; affected STS use
  denies within 60 seconds.

Verify:

Use exact additions:

- `subdomains/billing/migrations/006_entitlement_authority.sql`
- `subdomains/billing/src/services/entitlements.test.ts`
- `subdomains/api/services/entitlement-reservations.test.js`
- `subdomains/api/tests/contracts/authorization-vectors.test.js`

```sh
cd /Users/ryan/development/afterdark-meta-project/afterdarksys.com/subdomains/billing
npm test -- --runInBand src/services/entitlements.test.ts
cd ../api
npm test -- --runInBand services/entitlement-reservations.test.js tests/contracts/authorization-vectors.test.js
```

The authorization vectors are provider-neutral fixtures. Plan 03 proves the real
Dark Storage client produces the same results.

Done when: no customer mutation can proceed without a current central decision
and, where applicable, a live quota reservation lease.
