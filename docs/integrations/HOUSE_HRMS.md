# House HRMS integration

Status: Planned; no adapter implemented

Last observed: 2026-07-30

## 1. Purpose

Allow the house tenant to use the existing standalone HRMS through the ERP's
canonical HRMS API without replacing the standalone system's authentication,
roles, or source-of-truth responsibility.

This is one narrow ERP integration, not a generic integration platform.

## 2. Systems and ownership

| System | Responsibility |
|---|---|
| ERP | ERP login, tenant context, RBAC, stable workforce identity, encrypted connection state, canonical API |
| House HRMS | Employee/HR/Admin identities, provider roles, provider sessions, HRMS-owned data |
| House adapter | Transport, response validation, persona-specific authentication, translation, canonical errors |
| PostgreSQL in ERP | Provider binding, workforce mapping, encrypted session state, audit records |
| Deployment configuration | Provider base URL, allowed host, timeouts, TLS policy |

The local development checkout is conventionally a sibling repository at
`..\HRMS-2.0`. That path is not a production configuration value.

## 3. Observed implementation facts

Read-only inspection of the local checkout found:

- Node.js;
- Fastify;
- Prisma;
- PostgreSQL;
- separate Employee, HR, and Admin authentication paths;
- database-backed active-session records;
- Bearer authentication for employee requests;
- optional employee 2FA.

Observed employee login behavior:

```text
assignedEmail + password
-> employee lookup and employment-state check
-> password verification
-> settings lookup
-> either session creation
   or OTP creation and delivery
```

Successful employee sessions are effectively seven-day sessions.

## 4. Observed employee 2FA inconsistency

The employee login method can return a result indicating that 2FA is required.
The `/employee/login` route still assumes that the result contains a token and
constructs a successful token response from it.

The `/employee/login/2fa` route performs the second-factor operation and returns
the eventual Bearer credential through the response authorization header.

Consequences for the adapter:

- never accept HTTP success alone as proof of a valid session;
- validate the credential's presence and format;
- recognize the actual 2FA-required response deliberately;
- fail closed on contradictory or malformed responses;
- expose a canonical challenge/reconnect state instead of leaking provider
  response shapes.

This document does not authorize a fix in the standalone HRMS.

## 5. Planned ERP flow

```text
User logs into ERP
-> ERP derives trusted tenant and ERP user
-> ERP authorizes HRMS module access
-> ERP resolves the tenant's house-HRMS provider
-> user links one external persona
-> adapter sends credentials directly to the provider
-> adapter handles normal or 2FA-required response
-> adapter validates the resulting provider session
-> ERP encrypts and stores the session server-side
-> ERP persists the provider-person/workforce mapping
-> future calls use the server-side session
-> adapter translates the result into the canonical ERP contract
```

The browser never receives the provider session.

## 6. Persona scope

Connection state is scoped by:

```text
tenant + ERP user + provider + persona
```

Employee, HR, and Admin sessions must not be collapsed into one ambiguous
connection. ERP authorization and provider authorization are both required;
neither silently replaces the other.

The first implementation slice covers Employee only. HR and Admin require
separate inspection of their schemas, payloads, session lifecycles, and failure
paths.

## 7. Source of truth and data movement

The standalone house HRMS remains authoritative for its HRMS-owned records in
the first adapter version.

ERP stores only the state necessary to integrate safely:

- tenant/provider binding;
- ERP user and stable workforce mapping;
- provider subject identifier;
- persona;
- connection status;
- encrypted provider session and expiry metadata;
- security and privileged-action audit records.

There is no general HRMS replication or dual-write requirement in the first
version.

## 8. Authentication and sensitive data

- External passwords and OTPs are transient request inputs only.
- Never persist or log them.
- Never include them in audit metadata or error details.
- Encrypt provider sessions before PostgreSQL storage.
- Keep encryption keys outside the database and repository.
- Never return provider sessions to the browser.
- Rate-limit login, OTP, resend, and reconnect commands.
- Redact upstream bodies and headers that may contain credentials.

The encryption envelope and rotation rules must be approved before Step 05.

## 9. Network configuration

The adapter accepts provider endpoints only from trusted deployment
configuration.

Required safeguards:

- allowlisted host;
- TLS for remote traffic;
- explicit connect and request timeouts;
- bounded response size;
- no arbitrary URL supplied by an end user or tenant API request;
- actionable fail-fast startup validation.

Local mock verification must bind to loopback only.

## 10. Canonical failure behavior

| Provider condition | Canonical ERP behavior |
|---|---|
| Wrong password | Authentication failure without upstream details |
| 2FA required | Challenge-required state |
| Wrong, expired, or consumed OTP | Challenge failure with safe retry guidance |
| Missing or malformed token | Invalid provider response; no connection stored |
| Expired provider session | Reconnect-required state |
| Timeout or downtime | HRMS provider unavailable; unrelated ERP remains available |
| Logout failure | Report outcome as uncertain or failed; do not claim success |
| Malformed JSON or oversized body | Invalid provider response |
| Cross-user or cross-tenant access | Denied before provider session use |

Canonical error codes and payload schemas are finalized with the implementing
OpenAPI slice.

## 11. Restart and recovery

Valid encrypted sessions and connection mappings are restored from PostgreSQL
after ERP restart. Process memory and browser state are not authoritative.

If decryption fails, the key is unavailable, or the provider rejects the
session, the ERP must fail closed and require an explicit reconnect. It must not
silently create or elevate a provider identity.

## 12. External write operations

The employee connection/profile slice is primarily authentication and query
behavior.

Before adding an external command that mutates house-HRMS state, document and
verify:

- the provider's actual idempotency behavior;
- timeout and unknown-outcome handling;
- duplicate detection;
- retry policy;
- partial-failure recording;
- reconciliation;
- audit semantics.

Do not blindly retry non-idempotent provider writes.

## 13. Verification plan

Use a loopback-only mock provider for deterministic coverage of:

- login without 2FA;
- login with 2FA;
- wrong password and OTP;
- expired and consumed challenges;
- missing and malformed credentials;
- expired sessions;
- malformed and oversized responses;
- timeout and provider downtime;
- logout failure;
- restart restoration;
- secret redaction;
- cross-user and cross-tenant denial.

A live smoke test requires separately approved nonproduction credentials and
must not store them in the repository, documentation, logs, or chat history.

## 14. Known limitations

- The adapter does not exist yet.
- The exact canonical link/2FA/disconnect/reconnect routes are undecided.
- HR and Admin behavior has not been sufficiently revalidated for
  implementation.
- The standalone HRMS does not currently provide an authoritative OpenAPI
  contract for this adapter to consume.
- General synchronization and provider migration are deferred.
