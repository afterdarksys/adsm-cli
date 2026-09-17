# Plan 00 — Contracts and Delegated Authorization

Owner: `afterdarksys.com`

Depends on: none

## Task 1: Publish the contract set

Files:

- `docs/adsm/ARCHITECTURE.md`
- `docs/adr/ADSM_CONTROL_PLANE.md`
- `subdomains/api/openapi/adsm-v1.yaml`
- `subdomains/api/openapi/authorization-internal-v1.yaml`
- `subdomains/api/openapi/darkstorage-internal-v1.yaml`
- `subdomains/api/asyncapi/owner-events-v1.yaml`
- `subdomains/api/schemas/delegation-claims-v1.json`
- `subdomains/api/scripts/validate-adsm-contracts.js`
- `subdomains/api/tests/contracts/adsm-contracts.test.js`

Actions:

- Encode the public endpoints, internal authorization/reservation endpoints,
  Dark Storage adapter operations, lifecycle/meter events, and canonical error
  envelope from the umbrella plan.
- Define `afterdark` as provider of record and make upstream placement private.
- Record ownership boundaries: gateway orchestration, Dark Storage storage/S3,
  central authorization decisions, Authentik identity.

Verify:

```sh
cd /Users/ryan/development/afterdark-meta-project/afterdarksys.com/subdomains/api
npm run contract:lint
npm test -- --runInBand tests/contracts/adsm-contracts.test.js
```

Add `contract:lint` to `package.json`; it validates all schemas and compares the
public OpenAPI to the checked-in release baseline.

Done when: all later plans reference versioned schemas and no cross-service
payload remains prose-only.

## Task 2: Prove delegated authorization

Files:

- `subdomains/api/services/delegation-tokens.js`
- `subdomains/api/services/delegation-tokens.test.js`
- `subdomains/api/routes/internal-authorization.js`
- `subdomains/api/routes/internal-authorization.test.js`
- `subdomains/api/migrations/011_adsm_delegation_keys.sql`
- `infrastructure/authentik/blueprints/adsm-clients.yaml`
- `infrastructure/authentik/validate-blueprint.sh`

Actions:

- Create and validate the Authentik client blueprint for the public ADSM client,
  gateway-service workload, and Dark Storage workload/audiences in a disposable
  Authentik environment. Production application uses the existing
  `infrastructure/authentik/apply-blueprint.sh` through change management;
  automated tests must not mutate the live issuer.
- Implement a reference signer/validator for dedicated Ed25519 capability
  signing, overlap-key JWKS publication,
  60-second expiry, canonical request binding, and `jti` persistence.
- Prove gateway service authentication without making the gateway a human IdP.

Verify:

```sh
cd /Users/ryan/development/afterdark-meta-project/afterdarksys.com/subdomains/api
npm test -- --runInBand services/delegation-tokens.test.js routes/internal-authorization.test.js
cd ../../infrastructure/authentik
./validate-blueprint.sh blueprints/adsm-clients.yaml
```

The reference validator covers allow plus wrong issuer, audience, tenant,
operation, resource, request digest, expired key, future `nbf`, changed-request
replay, and signing-key rotation. Real Dark Storage middleware validation belongs
to Plan 03.

Done when: contract fixtures and the reference validator prove the capability
format without depending on future Dark Storage middleware.
