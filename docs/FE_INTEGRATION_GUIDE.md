# Frontend Integration Guide

This is the implementation guide for a React/TypeScript frontend integrating with the current `erp-go` backend. It explains the model first and then gives the exact HTTP contracts, request examples, permissions, lifecycle rules, and failure behavior.

It documents what exists in the repository now. It does not describe a future OAuth/OIDC implementation or pretend that the planned external-SaaS handoff protocol already exists.

## How to read this guide: one human, three hats

This guide is deliberately written for one person who may need to think in three different ways during the same integration:

| Hat | Question being answered | What to retain |
|---|---|---|
| Student | “What are a tenant, a control plane, and a session?” | The product and security model. |
| Consumer | “Which login do I use, and what should I see?” | The user journeys and visible behavior. |
| Developer | “Which value, endpoint, header, and failure state do I implement?” | The exact HTTP and frontend contracts. |

Read this opening section even if you are implementing the frontend immediately. Most integration mistakes in this system are not syntax mistakes; they come from confusing a platform operator with a tenant user, a tenant identifier with a secret, or a module entitlement with a user's permission.

## 1. The mental model

### 1.1 The 60-second explanation

Think of the ERP SaaS as a managed office building:

- The **platform operator** owns and operates the building.
- Each customer company rents a separate **office**. That office is a **tenant**.
- The permanent number on the office door is its **tenant ID**.
- A **platform superadmin** is the building-management employee who creates offices and decides which facilities each office has purchased.
- A **tenant user** is a person working inside one customer's office.
- A tenant's Admin, HR, Employee, Inventory Operator, or Sales Operator roles say what that person may do inside that office.

The building manager can provision Office A, but does not automatically become an employee of Office A. An HR manager in Office A can manage HR work allowed by Office A, but cannot create Office B or change Office B's subscription.

That is why the backend is split into two security planes:

1. The **control plane** manages the SaaS platform and tenant provisioning.
2. The **ERP plane** contains each tenant's business users, roles, permissions, and records.

### 1.2 Vocabulary without assumed background

| Term | Plain-language meaning | Current technical meaning |
|---|---|---|
| Platform | The ERP SaaS product operated by us. | Global control-plane data and APIs under `/control/v1`. |
| Control plane | The administration layer above all customer companies. | Tenants, plans, subscriptions, module entitlements, and platform sessions. |
| Tenant | One customer organization using the ERP. | A row in `control.tenants` with its own UUID and isolated business data. |
| Tenant ID | The stable identifier for one customer organization. | A UUID used to establish and verify tenant context. |
| ERP plane | The application area where a customer does business work. | Tenant-scoped APIs under `/api/v1`. |
| Tenant user | A person who can log in inside one tenant. | A `core.tenant_users` identity scoped to a tenant. |
| Platform superadmin | An operator of the SaaS product itself. | An active platform user with role `superadmin` and a platform session. |
| Tenant administrator | A customer-side role with tenant-level business permissions. | A tenant user/role concept; it is not the platform superadmin. |
| Module role | A business responsibility such as HR, Employee, or Inventory Operator. | Tenant-scoped roles connected to granular permissions. |
| Module entitlement | Whether the tenant has a module at all. | A control-plane decision checked before module permission. |
| Permission | Whether the current tenant user may perform one action. | A value such as `inventory.item.read`. |
| Plan | A reusable commercial offering. | A control-plane object that can include modules. |
| Subscription | A tenant's assignment to a plan and subscription state. | Control-plane metadata for the tenant. |
| Session | Server-managed proof that a login succeeded. | An opaque token whose hash and lifecycle are stored by the backend. |

An email address alone is not the identity boundary. For tenant login, think in terms of `tenant_id + email`. The same human may legitimately use the same email in multiple tenants, and one human may hold multiple module roles inside a tenant. Roles should converge on one tenant-user identity instead of creating separate login identities for “HR” and “Employee”.

### 1.3 What the consumer experiences

There are two different application journeys:

**Customer/tenant journey**

1. The person selects or arrives at their company context.
2. The login request sends `tenant_id`, `email`, and `password`.
3. The backend returns an ERP session.
4. The frontend uses that session only for the selected tenant.
5. The person sees modules enabled for the tenant and actions allowed by their role permissions.

**Platform-operator journey**

1. The platform operator opens the superadmin application area.
2. The login request sends platform `email` and `password`; it does not send a tenant ID.
3. The backend returns a platform session.
4. The operator manages tenant records, plans, subscriptions, and module entitlements.
5. That session is never used as an ERP tenant session.

If the same human performs both jobs, they still have two identities, two login requests, two session tokens, two browser-auth states, and two application areas. Logging in as one does not silently log in as the other.

### 1.4 What a platform superadmin actually is

**A platform superadmin is not a tenant administrator.** It is an operator account belonging to the company that operates this ERP SaaS platform.

In the current backend, an authenticated active superadmin may use the control-plane endpoints to:

- create and list tenants;
- create and list plans;
- assign a plan/subscription to a tenant;
- inspect available platform modules;
- enable or disable a module for a plan or tenant;
- inspect and revoke their own platform sessions.

A platform superadmin does **not** automatically:

- become a user, Admin, HR, Employee, or manager inside every tenant;
- read or mutate a tenant's Inventory, Purchase, Sales, HRMS, or other business records;
- bypass tenant roles or granular ERP permissions;
- send `X-Platform-Session` to `/api/v1` routes;
- turn `X-Platform-Session` into `X-ERP-Session`;
- impersonate a tenant user.

Support impersonation is not an available API in the current project. If support access is added later, it must be an explicit, reason-required, tenant-scoped, time-limited, audited flow—not a hidden superadmin bypass.

The current control-plane authorization is intentionally narrow: the protected handlers accept a valid platform session only when it belongs to an active platform user whose role is `superadmin`. Other platform roles may exist in storage for future use, but they are not accepted by these protected superadmin routes today.

The platform login has no `tenant_id` because it operates **above** tenants. In contrast, tenant ERP login requires `tenant_id` because the same email/password text must be evaluated inside one explicit customer boundary.

#### Superadmin, tenant admin, and HRMS Admin are different names for different scopes

| Identity/role | Belongs to | Session | Intended scope |
|---|---|---|---|
| Platform superadmin | ERP SaaS operator | `X-Platform-Session` | Provisioning, plans, subscriptions, and tenant module entitlements. |
| Tenant administrator | One customer tenant | `X-Tenant-ID` + `X-ERP-Session` | Whatever tenant-side administration permissions are assigned. |
| HRMS Admin | Existing HRMS application/module | HRMS's own current session | HRMS-local administration under that product's own model. |
| HR / Employee | Existing HRMS application/module | HRMS's own current session | HR and employee capabilities, possibly both linked to the same underlying person. |

Do not map an existing HRMS `Admin` row to the ERP platform superadmin merely because both contain the word “admin”. The former is customer/module authority; the latter is SaaS-operator authority. A future common handoff should map a person and their tenant/module roles without collapsing those authority levels.

### 1.5 The two live security planes

There are two security planes and two completely separate session types.

```text
Tenant ERP UI
  -> POST /api/v1/auth/login
  -> X-Tenant-ID + X-ERP-Session
  -> Inventory, Purchase, and Sales APIs

Platform administration UI
  -> POST /control/v1/auth/login
  -> X-Platform-Session
  -> Tenants, plans, subscriptions, and module-entitlement APIs
```

Do not combine these clients.

- A tenant user works inside one tenant and accesses ERP business data.
- A platform superadmin manages the SaaS control plane.
- A platform superadmin session is not permission to read or change tenant business data.
- If one human needs both experiences, the browser holds two distinct sessions and shows two distinct application areas.

### 1.6 Authentication, module entitlement, and permission are different checks

A successful tenant request normally passes three gates:

1. Session: is `X-ERP-Session` active and bound to `X-Tenant-ID`?
2. Module entitlement: is the requested module enabled for that tenant?
3. Permission: does the authenticated tenant user have the required permission?

The resulting errors mean different things:

- `erp_session_invalid`: clear the tenant session and return to login.
- `module_not_enabled`: hide or disable that module for this tenant.
- `permission_denied`: keep the user logged in, but show a forbidden state for that action.

Never interpret every `403` as “log out”. Inspect `error.code`.

### 1.7 The session tokens are opaque

The current ERP and platform tokens are random 32-byte values encoded as 64 hexadecimal characters. Only their SHA-256 hashes are stored in the database.

Frontend rules:

- Treat a token as an opaque string.
- Do not parse or decode it as a JWT.
- Do not infer identity, roles, tenant, or expiry from the token.
- Use `expires_at` from the login response.
- Validate a restored tenant session through `GET /api/v1/auth/me`.

Both login flows currently issue sessions with a 24-hour lifetime.

## 2. Start the backend and obtain the local handoff

From the `erp-go` repository:

```powershell
.\scripts\setup-fe-dev.ps1
```

The command:

1. Checks all configured database connections.
2. Applies every pending migration.
3. Seeds the platform superadmin.
4. Seeds the tenant and enables all registered modules.
5. Seeds an allowed tenant user and a no-access tenant user.
6. Starts the ERP and control-plane APIs.
7. Logs in with every seeded identity to verify the credentials.
8. Writes `.local/fe-integration.json`.

The generated file is local development data and is already gitignored. It contains test passwords, so do not copy it into committed frontend source or a production build.

### 2.1 Exactly where the tenant ID comes from

The setup helper does not discover a tenant ID, request one from a running API, or generate a different one on every run. It accepts a `-TenantID` parameter whose local default is:

```text
00000000-0000-0000-0000-000000000001
```

When you run the command without arguments, this is the precise lifecycle of that value:

1. PowerShell assigns the default UUID to the script's `TenantID` parameter.
2. The helper verifies that the value is a syntactically valid UUID.
3. It passes that exact value to the tenant seed and tenant-RBAC seed scripts.
4. The tenant seed upserts a control-plane tenant with that ID and enables the registered modules for it.
5. The RBAC seed creates/resets the local allowed and no-access test identities for that tenant.
6. The helper logs in using the same tenant ID and checks that `/api/v1/auth/me` and `/api/v1/modules` agree with it.
7. It prints `Tenant ID: ...` in the final terminal summary.
8. It writes the same value to `.local/fe-integration.json` as `tenant.id` and under each tenant credential as `tenant_id`.
9. It returns the whole manifest as a PowerShell object; the value is available as `.tenant.id`.

So the default run does not leave FE asking “which database row is mine?” The answer is returned explicitly and consistently.

### 2.2 Three ways to read the tenant ID after setup

**Method A — read the terminal summary**

The final output includes:

```text
Tenant ID:              00000000-0000-0000-0000-000000000001
```

This is convenient for a human copying a value once.

**Method B — capture the returned PowerShell object**

```powershell
$SetupResult = .\scripts\setup-fe-dev.ps1
$TenantID = $SetupResult.tenant.id
$TenantID
```

Use this when another local automation script needs to consume the result directly.

**Method C — read the durable local handoff manifest**

```powershell
$Handoff = Get-Content .\.local\fe-integration.json -Raw | ConvertFrom-Json
$TenantID = $Handoff.tenant.id
$TenantID
```

This is the recommended source for a frontend developer returning later, because the manifest also contains the matching URLs, user emails, passwords, header names, module list, and purpose of each test identity.

Useful inspection commands:

```powershell
$Handoff = Get-Content .\.local\fe-integration.json -Raw | ConvertFrom-Json

$Handoff.tenant
$Handoff.credentials.tenant_allowed
$Handoff.credentials.tenant_no_access
$Handoff.credentials.platform_superadmin
```

The manifest is a developer handoff artifact, not an endpoint and not a browser session store. Do not make production frontend code fetch a backend repository file.

### 2.3 What FE does with the tenant ID

For the current local login screen, FE uses the returned tenant ID in the JSON request body:

```json
{
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "email": "inventory.allowed+00000000@example.test",
  "password": "dev-tenant-password"
}
```

After login, FE keeps the same tenant ID alongside the returned ERP session token and sends both on every protected ERP request:

```http
X-Tenant-ID: 00000000-0000-0000-0000-000000000001
X-ERP-Session: <opaque-session-token>
```

Do not let a user change only the header and reuse the old session to “switch companies”. The backend binds the session to one tenant. A tenant switch means selecting the other tenant and obtaining a session for that tenant.

For convenient Vite development, the non-secret tenant ID may be copied into the frontend's ignored `.env.local` file:

```dotenv
VITE_ERP_TENANT_ID=00000000-0000-0000-0000-000000000001
```

```ts
const tenantId = import.meta.env.VITE_ERP_TENANT_ID;
```

Do not put the seeded passwords or session tokens in a `VITE_*` variable. Vite exposes those values to browser code. Enter local credentials through the login screen or use test-only tooling outside the production bundle.

### 2.4 Is the tenant ID a secret?

**A tenant ID is not a password or secret.** It is a routing and isolation identifier, comparable to an account number: it tells the backend which tenant boundary the request claims to address, but it grants no access by itself.

Security does not come from hiding the tenant ID. It comes from the backend verifying all of these facts together:

- the ERP session exists, is active, and has not expired or been revoked;
- the session belongs to the claimed tenant;
- the tenant has the requested module enabled;
- the authenticated tenant user has the required permission;
- database queries remain tenant-scoped.

This is also why FE must send `X-Tenant-ID` but the backend must never trust that header alone. Knowing or changing another tenant's UUID must not grant access.

Do not confuse these identifiers:

| Identifier | Identifies | Secret? | Typical source |
|---|---|---|---|
| Tenant ID | Customer organization/security realm | No | Local handoff now; trusted tenant discovery/provisioning later. |
| Tenant user ID | Person-account inside a tenant | No | Login response or `/api/v1/auth/me`. |
| Platform user ID | SaaS-operator account | No | Platform login response. |
| Session token | One authenticated session | Yes | Login response only. |
| Module ID | Product module such as `inventory` | No | Module endpoint/manifest. |

### 2.5 Supplying a different local tenant ID

The parameter exists so the primary local test tenant can use a deliberate UUID:

```powershell
$TenantID = [guid]::NewGuid().ToString()

.\scripts\setup-fe-dev.ps1 `
  -TenantID $TenantID `
  -TenantSlug "acme-local" `
  -TenantLegalName "Acme Private Limited" `
  -TenantDisplayName "Acme"
```

Keep the generated `$TenantID` and reuse the same command on later runs. Do not generate a new UUID on every restart.

The present helper is designed to reset one canonical FE development fixture, with deterministic seeded user and role IDs. On a database that already contains the default fixture, changing only `-TenantID` is **not** a supported way to accumulate multiple independent local fixtures. Use the documented default for ordinary FE integration; use a different ID only when deliberately initializing the fixture in a suitable clean/local environment.

In production, tenant identity must come from a trusted provisioning and tenant-discovery flow—for example an assigned domain/subdomain, an organization selector backed by server-side membership, or a trusted handoff. The current repository does not yet expose that production discovery/handoff endpoint. A free-text UUID field is acceptable as a temporary local developer input, not as the final production tenant-security design.

### 2.6 What “idempotent” means here

Running the normal default command again is expected and safe for the local fixture:

- it reuses the same tenant UUID;
- it updates/resets the local tenant metadata and test identities;
- it enables the registered modules;
- it validates fresh logins;
- it rewrites the local handoff with current results;
- it does not overwrite an existing `.env` file.

It is a bootstrap/reset helper, not a production tenant-provisioning API and not a schema for external SaaS federation.

Default local values:

| Value | Default |
|---|---|
| ERP API | `http://localhost:8080` |
| Control-plane API | `http://localhost:8081` |
| Tenant ID | `00000000-0000-0000-0000-000000000001` |
| Tenant slug | `dev-tenant` |
| Allowed ERP email | `inventory.allowed+00000000@example.test` |
| Allowed ERP password | `dev-tenant-password` |
| No-access ERP email | `inventory.noaccess+00000000@example.test` |
| No-access ERP password | `dev-noaccess-password` |
| Platform email | `dev.superadmin@example.test` |
| Platform password | `dev-superadmin-password` |

Use the allowed ERP identity for normal module development. Use the no-access identity to verify that the UI handles `403 permission_denied` correctly. Use the platform identity only in the superadmin/control-plane UI.

The normal setup validates and then revokes its temporary sessions. To intentionally include live session tokens in the manifest:

```powershell
.\scripts\setup-fe-dev.ps1 -IncludeSessionTokens
```

That option is useful for manual API exploration. A login screen should still call the real login endpoint instead of depending on a pre-generated token.

## 3. Browser networking: use a development proxy

The current Go APIs do not install CORS middleware. Direct browser calls from a Vite origin such as `http://localhost:5173` to ports `8080` or `8081` will trigger cross-origin preflight, especially because the APIs use custom session headers.

Postman and PowerShell are not restricted by browser CORS, which is why a request may work there and fail in the browser.

For Vite, proxy the two path namespaces:

```ts
// vite.config.ts
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
      "/control": {
        target: "http://localhost:8081",
        changeOrigin: true,
      },
    },
  },
});
```

The browser can then call relative URLs such as `/api/v1/auth/login` and `/control/v1/auth/login`. In production, use the same idea at the reverse proxy: route `/api/*` to the ERP API and `/control/*` to the control-plane API under the frontend's origin.

Do not solve local CORS by disabling browser security.

## 4. Recommended frontend structure

Keep these surfaces separate:

```text
src/
  api/
    api-error.ts
    tenant-client.ts
    platform-client.ts
    contracts.ts
  auth/
    tenant-session.ts
    TenantAuthProvider.tsx
    platform-session.ts
    PlatformAuthProvider.tsx
  modules/
    inventory/
    purchase/
    sales/
  platform/
    tenants/
    plans/
```

Do not make one generic client that sometimes attaches all three authentication headers. A request must be either a tenant ERP request or a platform request.

### Common API error type

Every application JSON error uses this shape:

```json
{
  "error": {
    "code": "permission_denied",
    "message": "permission denied"
  }
}
```

A reusable TypeScript parser:

```ts
export type ErrorBody = {
  error: {
    code: string;
    message: string;
  };
};

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export async function expectJson<T>(response: Response): Promise<T> {
  const text = await response.text();
  let payload: unknown;

  try {
    payload = text ? JSON.parse(text) : undefined;
  } catch {
    payload = undefined;
  }

  if (!response.ok) {
    const body = payload as Partial<ErrorBody> | undefined;
    throw new ApiError(
      response.status,
      body?.error?.code ?? "unexpected_http_error",
      body?.error?.message ?? `HTTP ${response.status}`,
    );
  }

  return payload as T;
}
```

The default Go `404` response may not use the application error shape, which is why the parser falls back to the HTTP status when the body is not JSON.

## 5. Tenant ERP authentication

### 5.1 Login

`POST /api/v1/auth/login`

No session headers are used on login.

```json
{
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "email": "inventory.allowed+00000000@example.test",
  "password": "dev-tenant-password"
}
```

Successful response:

```json
{
  "session_token": "opaque-64-character-token",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "user_id": "11111111-1111-1111-1111-111111111111",
  "email": "inventory.allowed+00000000@example.test",
  "expires_at": "2026-07-23T10:00:00Z"
}
```

TypeScript contract and call:

```ts
export type TenantLoginResponse = {
  session_token: string;
  tenant_id: string;
  user_id: string;
  email: string;
  expires_at: string;
};

export async function loginTenant(input: {
  tenant_id: string;
  email: string;
  password: string;
}): Promise<TenantLoginResponse> {
  const response = await fetch("/api/v1/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });

  return expectJson<TenantLoginResponse>(response);
}
```

The same email may exist in multiple tenants. `tenant_id` is part of the login identity, not an optional filter.

### 5.2 Persisting the tenant session

The backend does not set an HttpOnly cookie. A browser SPA must currently retain the header token itself.

Recommended current compromise:

- Keep the live value in React state/memory.
- Use `sessionStorage`, not `localStorage`, if reload persistence is required.
- Store the complete login result, including `tenant_id` and `expires_at`.
- Never log the token or include it in analytics/error reports.
- Clear the stored value when expired or rejected.

`sessionStorage` is still readable by JavaScript and therefore does not protect against XSS. A future BFF/HttpOnly-cookie layer would be stronger; it is not implemented today.

```ts
const TENANT_SESSION_KEY = "erp.tenant-session.v1";

export function saveTenantSession(session: TenantLoginResponse): void {
  sessionStorage.setItem(TENANT_SESSION_KEY, JSON.stringify(session));
}

export function loadTenantSession(): TenantLoginResponse | null {
  const raw = sessionStorage.getItem(TENANT_SESSION_KEY);
  if (!raw) return null;

  const session = JSON.parse(raw) as TenantLoginResponse;
  if (Date.parse(session.expires_at) <= Date.now()) {
    sessionStorage.removeItem(TENANT_SESSION_KEY);
    return null;
  }

  return session;
}

export function clearTenantSession(): void {
  sessionStorage.removeItem(TENANT_SESSION_KEY);
}
```

### 5.3 Tenant request wrapper

Every authenticated ERP request attaches both headers:

```http
X-Tenant-ID: 00000000-0000-0000-0000-000000000001
X-ERP-Session: <opaque token>
```

Do not send `X-User-ID`. The backend derives the user from the session.

```ts
export async function tenantRequest<T>(
  session: TenantLoginResponse,
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const headers = new Headers(init.headers);
  headers.set("X-Tenant-ID", session.tenant_id);
  headers.set("X-ERP-Session", session.session_token);

  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(path, {
    ...init,
    headers,
  });

  try {
    return await expectJson<T>(response);
  } catch (error) {
    if (
      error instanceof ApiError &&
      (error.code === "erp_session_invalid" ||
        error.code === "erp_session_required")
    ) {
      clearTenantSession();
    }

    throw error;
  }
}
```

### 5.4 Bootstrap after reload

When the application starts:

1. Load the stored tenant session.
2. Reject it locally if `expires_at` is past.
3. Call `GET /api/v1/auth/me`.
4. If valid, call `GET /api/v1/modules`.
5. Store the returned user and enabled module IDs in the auth context.
6. Render authenticated routes.

`GET /api/v1/auth/me` response:

```json
{
  "user": {
    "tenant_id": "00000000-0000-0000-0000-000000000001",
    "user_id": "11111111-1111-1111-1111-111111111111",
    "email": "inventory.allowed+00000000@example.test",
    "display_name": "Dev Inventory Allowed User",
    "status": "active"
  }
}
```

`GET /api/v1/modules` currently requires `X-Tenant-ID` but not `X-ERP-Session`. Call it after validating the session anyway. Its result is navigation/entitlement metadata, not proof that the user is authenticated or permitted.

```json
{
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "modules": [
    { "id": "accounting", "name": "Accounting", "status": "active" },
    { "id": "inventory", "name": "Inventory", "status": "active" },
    { "id": "purchase", "name": "Purchase", "status": "active" },
    { "id": "sales", "name": "Sales", "status": "active" }
  ]
}
```

Accounting is registered as a module, but no Accounting business routes are currently wired. Do not infer endpoint availability only from the module registry.

### 5.5 Logout

`POST /api/v1/auth/logout` with both tenant headers returns:

```json
{ "logged_out": true }
```

Call the backend before clearing local state so the server-side session is revoked. If logout fails because the network is unavailable, the frontend may clear its local UI state, but it should not claim that the remote session was revoked.

## 6. Platform authentication

Platform auth is for the superadmin UI only.

### Login

`POST /control/v1/auth/login`

```json
{
  "email": "dev.superadmin@example.test",
  "password": "dev-superadmin-password"
}
```

Response:

```json
{
  "session_token": "different-opaque-token",
  "platform_user_id": "00000000-0000-0000-0000-00000000aaaa",
  "expires_at": "2026-07-23T10:00:00Z"
}
```

Every protected control-plane call uses only:

```http
X-Platform-Session: <platform token>
```

Use a separate storage key, provider, request wrapper, and logout flow from tenant auth. Never attach a platform token to `/api/v1/*`, and never attach an ERP token to `/control/v1/*`.

There is currently no platform `/auth/me` endpoint. On reload, validate the stored platform session with a protected read such as `GET /control/v1/auth/sessions` or `GET /control/v1/modules`.

## 7. HTTP behavior shared by all APIs

### Status codes

| Status | FE interpretation |
|---|---|
| `200` | Read/action succeeded |
| `201` | Resource created |
| `400` | Missing/invalid input or tenant header; show field/business message |
| `401` | Invalid login or a required session header is absent |
| `403` | Session invalid, module disabled, or permission denied; inspect `error.code` |
| `404` | Entity or action route not found |
| `409` | Lifecycle/stock/duplicate conflict; refresh affected data |
| `423` | Platform login temporarily locked |
| `500` | Unexpected backend failure |
| `503` | Database health failure |

### JSON, timestamps, decimals, and IDs

- Send `Content-Type: application/json` for JSON bodies.
- UUID identifiers are strings.
- Timestamps are RFC 3339, for example `2026-07-22T10:30:00Z`.
- Omit an optional timestamp instead of sending an empty string.
- Quantities, prices, unit costs, totals, and balances are decimal strings, not JavaScript numbers.
- Currency codes are three uppercase letters such as `INR` or `USD`.

Keep decimal values as strings through form state and API DTOs. Converting money or quantities to IEEE-754 `number` can introduce rounding errors.

### Current list behavior

There is no public pagination/filter/sort query contract yet. Transactional lists generally return the latest 100 records; master lists currently return their complete ordered result. Do not invent query parameters in the client.

After a mutation, either use the returned entity to update the cache or invalidate/refetch the corresponding list.

## 8. Complete endpoint catalogue

### Health and tenant bootstrap

| Method | Path | Auth | Purpose |
|---|---|---|---|
| `GET` | ERP `/healthz` | None | ERP process health |
| `GET` | ERP `/healthz/db` | None | ERP database health |
| `GET` | Control `/healthz` | None | Control-plane process health |
| `GET` | Control `/healthz/db` | None | Control-plane database health |
| `POST` | `/api/v1/auth/login` | None | Tenant login |
| `GET` | `/api/v1/auth/me` | Tenant session | Resolve current tenant user |
| `POST` | `/api/v1/auth/logout` | Tenant session | Revoke current tenant session |
| `GET` | `/api/v1/modules` | `X-Tenant-ID` | List enabled tenant modules |

### Platform control plane

All routes below except login require `X-Platform-Session`.

| Method | Path | Request/response purpose |
|---|---|---|
| `POST` | `/control/v1/auth/login` | Create platform session |
| `POST` | `/control/v1/auth/logout` | Revoke current session |
| `GET` | `/control/v1/auth/sessions` | List active sessions for current platform user |
| `POST` | `/control/v1/auth/sessions/revoke-all` | Revoke all sessions; clear the current client token |
| `POST` | `/control/v1/auth/sessions/cleanup-expired` | Remove expired session records |
| `GET` | `/control/v1/tenants` | List tenants |
| `POST` | `/control/v1/tenants` | Create tenant |
| `GET` | `/control/v1/modules` | List registered modules |
| `GET` | `/control/v1/tenants/{tenant_id}/modules` | List tenant module entitlements |
| `POST` | `/control/v1/tenants/{tenant_id}/modules/{module_id}/enable` | Enable module |
| `POST` | `/control/v1/tenants/{tenant_id}/modules/{module_id}/disable` | Disable module |
| `GET` | `/control/v1/plans` | List plans |
| `POST` | `/control/v1/plans` | Create plan |
| `GET` | `/control/v1/plans/{plan_id}/modules` | List plan modules |
| `POST` | `/control/v1/plans/{plan_id}/modules/{module_id}/enable` | Add module to plan |
| `POST` | `/control/v1/tenants/{tenant_id}/subscription` | Assign/update tenant subscription and derived modules |

### Inventory

All Inventory business routes require the tenant session, the `inventory` entitlement, and the listed permission.

| Method | Path | Permission | Response data key |
|---|---|---|---|
| `GET` | `/api/v1/inventory/units` | `inventory.unit.read` | `units` |
| `POST` | `/api/v1/inventory/units` | `inventory.unit.write` | `unit` |
| `GET` | `/api/v1/inventory/items` | `inventory.item.read` | `items` |
| `POST` | `/api/v1/inventory/items` | `inventory.item.write` | `item` |
| `GET` | `/api/v1/inventory/locations` | `inventory.location.read` | `locations` |
| `POST` | `/api/v1/inventory/locations` | `inventory.location.write` | `location` |
| `GET` | `/api/v1/inventory/stock-movements` | `inventory.stock_movement.read` | `movements` |
| `POST` | `/api/v1/inventory/stock-movements` | `inventory.stock_movement.write` | `movement` |
| `GET` | `/api/v1/inventory/stock-balances` | `inventory.stock_balance.read` | `balances` |
| `GET` | `/api/v1/inventory/transfers` | `inventory.transfer.read` | `transfers` |
| `POST` | `/api/v1/inventory/transfers` | `inventory.transfer.write` | `transfer` |
| `GET` | `/api/v1/inventory/stock-counts` | `inventory.stock_count.read` | `stock_counts` |
| `POST` | `/api/v1/inventory/stock-counts` | `inventory.stock_count.write` | `stock_count` |
| `GET` | `/api/v1/inventory/reservations` | `inventory.reservation.read` | `reservations` |
| `POST` | `/api/v1/inventory/reservations` | `inventory.reservation.write` | `reservation` |
| `POST` | `/api/v1/inventory/reservations/{id}/release` | `inventory.reservation.write` | `reservation` |
| `GET` | `/api/v1/inventory/availability` | `inventory.reservation.read` | `availability` |
| `GET` | `/api/v1/inventory/cost-layers` | `inventory.cost.read` | `cost_layers` |
| `POST` | `/api/v1/inventory/cost-layers` | `inventory.cost.write` | `cost_layer` |
| `GET` | `/api/v1/inventory/reports/valuation` | `inventory.cost.read` | `valuation` |
| `GET` | `/api/v1/inventory/reports/summary` | `inventory.report.read` | `summary` |
| `GET` | `/api/v1/inventory/reports/stock-ledger` | `inventory.report.read` | `stock_ledger` |

### Purchase

| Method | Path | Permission | Response data key |
|---|---|---|---|
| `GET` | `/api/v1/purchase/suppliers` | `purchase.supplier.read` | `suppliers` |
| `POST` | `/api/v1/purchase/suppliers` | `purchase.supplier.write` | `supplier` |
| `GET` | `/api/v1/purchase/orders` | `purchase.order.read` | `purchase_orders` |
| `POST` | `/api/v1/purchase/orders` | `purchase.order.create` | `purchase_order` |
| `POST` | `/api/v1/purchase/orders/{id}/approve` | `purchase.order.approve` | `purchase_order` |
| `GET` | `/api/v1/purchase/receipts` | `purchase.receipt.read` | `receipts` |
| `POST` | `/api/v1/purchase/receipts` | `purchase.receipt.write` | `receipt` |
| `POST` | `/api/v1/purchase/receipts/{id}/reverse` | `purchase.receipt.reverse` | `receipt` |

### Sales

| Method | Path | Permission | Response data key |
|---|---|---|---|
| `GET` | `/api/v1/sales/customers` | `sales.customer.read` | `customers` |
| `POST` | `/api/v1/sales/customers` | `sales.customer.write` | `customer` |
| `GET` | `/api/v1/sales/orders` | `sales.order.read` | `sales_orders` |
| `POST` | `/api/v1/sales/orders` | `sales.order.create` | `sales_order` |
| `POST` | `/api/v1/sales/orders/{id}/confirm` | `sales.order.approve` | `sales_order` |
| `GET` | `/api/v1/sales/issues` | `sales.issue.read` | `issues` |
| `POST` | `/api/v1/sales/issues` | `sales.issue.write` | `issue` |
| `POST` | `/api/v1/sales/issues/{id}/reverse` | `sales.issue.reverse` | `issue` |

Every tenant business response also includes `tenant_id` beside the data key.

### Response entity contracts

The following TypeScript shapes mirror the currently returned JSON. Timestamps are represented as strings because JSON carries RFC 3339 text.

Shared and Inventory master/ledger entities:

```ts
export type Module = {
  id: string;
  name: string;
  status: string;
};

export type Unit = {
  id: string;
  code: string;
  name: string;
  description: string;
  status: "active" | "inactive" | "archived";
  created_at: string;
  updated_at: string;
};

export type Item = {
  id: string;
  sku: string;
  name: string;
  description: string;
  status: "active" | "inactive" | "archived";
  base_unit_id: string;
  base_unit?: Pick<Unit, "id" | "code" | "name">;
  created_at: string;
  updated_at: string;
};

export type Location = {
  id: string;
  code: string;
  name: string;
  description: string;
  status: "active" | "inactive" | "archived";
  created_at: string;
  updated_at: string;
};

export type StockMovementLine = {
  id: string;
  item_id: string;
  item_sku: string;
  item_name: string;
  location_id: string;
  location_code: string;
  location_name: string;
  quantity_delta: string;
};

export type StockMovement = {
  id: string;
  movement_number: string;
  movement_type: string;
  reference: string;
  notes: string;
  occurred_at: string;
  status: string;
  lines: StockMovementLine[];
  created_at: string;
};

export type StockBalance = {
  item_id: string;
  item_sku: string;
  item_name: string;
  base_unit_id: string;
  base_unit_code: string;
  location_id: string;
  location_code: string;
  location_name: string;
  quantity: string;
};

export type Reservation = {
  id: string;
  reservation_number: string;
  item_id: string;
  item_sku: string;
  item_name: string;
  location_id: string;
  location_code: string;
  quantity: string;
  reference: string;
  notes: string;
  status: string;
  expires_at: string | null;
  released_at: string | null;
  created_at: string;
};

export type Availability = {
  item_id: string;
  item_sku: string;
  item_name: string;
  location_id: string;
  location_code: string;
  on_hand: string;
  reserved: string;
  available: string;
};

export type CostLayer = {
  id: string;
  item_id: string;
  item_sku: string;
  item_name: string;
  unit_cost: string;
  currency_code: string;
  source: string;
  effective_at: string;
  created_at: string;
};

export type ValuationLine = {
  item_id: string;
  item_sku: string;
  item_name: string;
  location_id: string;
  location_code: string;
  quantity: string;
  unit_cost: string;
  currency_code: string;
  value: string;
};

export type InventorySummary = {
  item_id: string;
  item_sku: string;
  item_name: string;
  on_hand: string;
  reserved: string;
  available: string;
  unit_cost: string;
  currency_code: string;
  inventory_value: string;
};
```

Inventory operation entities:

```ts
export type TransferLine = {
  id: string;
  item_id: string;
  item_sku: string;
  item_name: string;
  source_location_id: string;
  source_location_code: string;
  destination_location_id: string;
  destination_location_code: string;
  quantity: string;
};

export type Transfer = {
  id: string;
  transfer_number: string;
  reference: string;
  notes: string;
  transferred_at: string;
  status: string;
  stock_movement_id: string;
  stock_movement_number: string;
  lines: TransferLine[];
  created_at: string;
};

export type StockCountLine = {
  id: string;
  item_id: string;
  item_sku: string;
  item_name: string;
  system_quantity: string;
  counted_quantity: string;
  variance_quantity: string;
};

export type StockCount = {
  id: string;
  count_number: string;
  location_id: string;
  location_code: string;
  reference: string;
  notes: string;
  counted_at: string;
  status: string;
  stock_movement_id: string;
  stock_movement_number: string;
  lines: StockCountLine[];
  created_at: string;
};
```

Purchase and Sales entities:

```ts
export type PartyRef = {
  id: string;
  code: string;
  name: string;
};

export type Supplier = {
  id: string;
  code: string;
  name: string;
  email: string;
  phone: string;
  address: string;
  status: "active" | "inactive" | "archived";
  created_at: string;
  updated_at: string;
};

export type PurchaseOrderLine = {
  id: string;
  item_id: string;
  item_sku: string;
  item_name: string;
  quantity: string;
  received_quantity: string;
  remaining_quantity: string;
  unit_price: string;
  line_total: string;
};

export type PurchaseOrder = {
  id: string;
  order_number: string;
  supplier: PartyRef;
  supplier_reference: string;
  notes: string;
  currency_code: string;
  ordered_at: string;
  expected_at: string | null;
  status: string;
  approved_at: string | null;
  lines: PurchaseOrderLine[];
  created_at: string;
  updated_at: string;
};

export type ReceiptLine = {
  id: string;
  item_id: string;
  item_sku: string;
  item_name: string;
  location_id: string;
  location_code: string;
  location_name: string;
  quantity_received: string;
};

export type PurchaseReceipt = {
  id: string;
  receipt_number: string;
  purchase_order_id: string;
  purchase_order_number: string;
  supplier: PartyRef;
  supplier_name: string;
  reference: string;
  notes: string;
  received_at: string;
  status: string;
  stock_movement_id: string;
  reversal_stock_movement_id: string;
  reversal_reason: string;
  reversed_at: string | null;
  lines: ReceiptLine[];
  created_at: string;
};

export type Customer = {
  id: string;
  code: string;
  name: string;
  email: string;
  phone: string;
  billing_address: string;
  shipping_address: string;
  status: "active" | "inactive" | "archived";
  created_at: string;
  updated_at: string;
};

export type SalesOrderLine = {
  id: string;
  item_id: string;
  item_sku: string;
  item_name: string;
  quantity: string;
  issued_quantity: string;
  remaining_quantity: string;
  unit_price: string;
  line_total: string;
};

export type SalesOrder = {
  id: string;
  order_number: string;
  customer: PartyRef;
  customer_reference: string;
  notes: string;
  currency_code: string;
  ordered_at: string;
  requested_delivery_at: string | null;
  status: string;
  confirmed_at: string | null;
  lines: SalesOrderLine[];
  created_at: string;
  updated_at: string;
};

export type SalesIssueLine = {
  id: string;
  item_id: string;
  item_sku: string;
  item_name: string;
  location_id: string;
  location_code: string;
  location_name: string;
  quantity_issued: string;
};

export type SalesIssue = {
  id: string;
  issue_number: string;
  sales_order_id: string;
  sales_order_number: string;
  customer: PartyRef;
  customer_name: string;
  reference: string;
  notes: string;
  issued_at: string;
  status: string;
  stock_movement_id: string;
  stock_movement_number: string;
  reversal_stock_movement_id: string;
  reversal_stock_movement_number: string;
  reversal_reason: string;
  reversed_at: string | null;
  lines: SalesIssueLine[];
  created_at: string;
};
```

Control-plane entities:

```ts
export type Tenant = {
  id: string;
  slug: string;
  legal_name: string;
  display_name: string;
  status: "trial" | "active" | "suspended" | "closed";
  created_at: string;
  updated_at: string;
};

export type Plan = {
  id: string;
  name: string;
  status: "draft" | "active" | "archived";
  created_at: string;
};

export type Subscription = {
  id: string;
  tenant_id: string;
  plan_id: string;
  status: "trialing" | "active" | "past_due" | "suspended" | "cancelled";
  starts_at: string;
  ends_at?: string;
  created_at: string;
};

export type PlatformSessionRecord = {
  id: string;
  platform_user_id: string;
  status: string;
  expires_at: string;
  last_seen_at?: string;
  revoked_at?: string;
  created_at: string;
};
```

## 9. Write-payload cookbook

These are valid request shapes. Replace placeholder UUIDs with IDs returned by earlier API calls.

### Platform: create tenant

`POST /control/v1/tenants`

```json
{
  "slug": "acme",
  "legal_name": "Acme Private Limited",
  "display_name": "Acme",
  "status": "trial"
}
```

`status` may be `trial`, `active`, `suspended`, or `closed`; omitted status defaults to `trial`. The server generates the tenant UUID.

### Platform: create plan and assign subscription

`POST /control/v1/plans`

```json
{
  "id": "growth",
  "name": "Growth",
  "status": "active"
}
```

Plan status may be `draft`, `active`, or `archived`; omitted status defaults to `draft`.

Enable modules on the plan with path actions, then assign it:

`POST /control/v1/tenants/{tenant_id}/subscription`

```json
{
  "plan_id": "growth",
  "status": "active"
}
```

Subscription status may be `trialing`, `active`, `past_due`, `suspended`, or `cancelled`; omitted status defaults to `active`. Assignment returns the subscription and the resulting `enabled_modules`.

### Inventory unit

`POST /api/v1/inventory/units`

```json
{
  "code": "EA",
  "name": "Each",
  "description": "Single sellable unit",
  "status": "active"
}
```

### Inventory item

Create an active unit first, then use its ID:

`POST /api/v1/inventory/items`

```json
{
  "sku": "CHAIR-001",
  "name": "Office Chair",
  "description": "Ergonomic office chair",
  "status": "active",
  "base_unit_id": "11111111-aaaa-bbbb-cccc-111111111111"
}
```

`base_unit_id` must identify an active unit in the same tenant.

### Inventory location

`POST /api/v1/inventory/locations`

```json
{
  "code": "MAIN",
  "name": "Main Warehouse",
  "description": "Primary stock location",
  "status": "active"
}
```

Master statuses are `active`, `inactive`, or `archived`; omitted status defaults to `active`.

### Direct stock adjustment

`POST /api/v1/inventory/stock-movements`

```json
{
  "movement_type": "adjustment",
  "reference": "OPENING-2026-001",
  "notes": "Opening balance",
  "occurred_at": "2026-07-22T10:00:00Z",
  "lines": [
    {
      "item_id": "22222222-aaaa-bbbb-cccc-222222222222",
      "location_id": "33333333-aaaa-bbbb-cccc-333333333333",
      "quantity_delta": "10.000"
    }
  ]
}
```

Direct creation supports only `adjustment`. Positive quantities add stock and negative quantities remove stock. Receipts, issues, transfers, and reversals must use their owning document APIs.

### Inventory transfer

`POST /api/v1/inventory/transfers`

```json
{
  "reference": "TR-001",
  "notes": "Move stock to showroom",
  "transferred_at": "2026-07-22T11:00:00Z",
  "lines": [
    {
      "item_id": "22222222-aaaa-bbbb-cccc-222222222222",
      "source_location_id": "33333333-aaaa-bbbb-cccc-333333333333",
      "destination_location_id": "44444444-aaaa-bbbb-cccc-444444444444",
      "quantity": "2.000"
    }
  ]
}
```

Source and destination must differ. The server rejects a transfer when available source stock is insufficient.

### Physical stock count

`POST /api/v1/inventory/stock-counts`

```json
{
  "location_id": "33333333-aaaa-bbbb-cccc-333333333333",
  "reference": "COUNT-JUL-2026",
  "notes": "Monthly count",
  "counted_at": "2026-07-22T12:00:00Z",
  "lines": [
    {
      "item_id": "22222222-aaaa-bbbb-cccc-222222222222",
      "counted_quantity": "9.000"
    }
  ]
}
```

The server calculates system quantity and variance and appends the required adjustment movement.

### Reservation and release

`POST /api/v1/inventory/reservations`

```json
{
  "item_id": "22222222-aaaa-bbbb-cccc-222222222222",
  "location_id": "33333333-aaaa-bbbb-cccc-333333333333",
  "quantity": "2.000",
  "reference": "CART-1001",
  "notes": "Reserved for customer order",
  "expires_at": "2026-07-23T12:00:00Z"
}
```

`expires_at` is optional but must be in the future. Release with `POST /api/v1/inventory/reservations/{id}/release` and no body. Reservations change available-to-promise, not on-hand stock.

### Cost layer

`POST /api/v1/inventory/cost-layers`

```json
{
  "item_id": "22222222-aaaa-bbbb-cccc-222222222222",
  "unit_cost": "1250.5000",
  "currency_code": "INR",
  "source": "opening-cost",
  "effective_at": "2026-07-22T10:00:00Z"
}
```

Cost layers are append-only. `unit_cost` must be non-negative.

### Purchase supplier

`POST /api/v1/purchase/suppliers`

```json
{
  "code": "SUP-001",
  "name": "Acme Supplies",
  "email": "orders@acme.example",
  "phone": "+91-9000000000",
  "address": "Kolkata, India",
  "status": "active"
}
```

Supplier code must be unique inside the tenant.

### Purchase order and approval

`POST /api/v1/purchase/orders`

```json
{
  "supplier_id": "55555555-aaaa-bbbb-cccc-555555555555",
  "supplier_reference": "QUOTE-91",
  "notes": "Restock office chairs",
  "currency_code": "INR",
  "ordered_at": "2026-07-22T10:00:00Z",
  "expected_at": "2026-07-30T10:00:00Z",
  "lines": [
    {
      "item_id": "22222222-aaaa-bbbb-cccc-222222222222",
      "quantity": "20.000",
      "unit_price": "1200.0000"
    }
  ]
}
```

The order starts as `draft`. Confirm the user action before calling `POST /api/v1/purchase/orders/{id}/approve`. The action has no request body and only works once while the order is draft.

### Purchase receipt and reversal

`POST /api/v1/purchase/receipts`

```json
{
  "purchase_order_id": "66666666-aaaa-bbbb-cccc-666666666666",
  "reference": "GRN-001",
  "notes": "First delivery",
  "received_at": "2026-07-25T10:00:00Z",
  "lines": [
    {
      "item_id": "22222222-aaaa-bbbb-cccc-222222222222",
      "location_id": "33333333-aaaa-bbbb-cccc-333333333333",
      "quantity_received": "10.000"
    }
  ]
}
```

The purchase order must be approved or partially received. Each item must exist on the order, and the receipt cannot exceed remaining quantity.

Reverse with `POST /api/v1/purchase/receipts/{id}/reverse`:

```json
{
  "reason": "Goods receipt entered against the wrong delivery"
}
```

Reversal appends compensating stock history and may reopen purchase-order receipt progress. Do not delete or edit the original receipt locally.

### Sales customer

`POST /api/v1/sales/customers`

```json
{
  "code": "CUS-001",
  "name": "Example Customer",
  "email": "buyer@example.test",
  "phone": "+91-9111111111",
  "billing_address": "Billing address",
  "shipping_address": "Shipping address",
  "status": "active"
}
```

Customer code must be unique inside the tenant.

### Sales order and confirmation

`POST /api/v1/sales/orders`

```json
{
  "customer_id": "77777777-aaaa-bbbb-cccc-777777777777",
  "customer_reference": "CUSTOMER-PO-42",
  "notes": "Deliver to head office",
  "currency_code": "INR",
  "ordered_at": "2026-07-22T10:00:00Z",
  "requested_delivery_at": "2026-07-28T10:00:00Z",
  "lines": [
    {
      "item_id": "22222222-aaaa-bbbb-cccc-222222222222",
      "quantity": "5.000",
      "unit_price": "1800.0000"
    }
  ]
}
```

The order starts as `draft`. Confirm with `POST /api/v1/sales/orders/{id}/confirm`, with no body. Confirmation only works once while the order is draft.

### Sales issue and reversal

`POST /api/v1/sales/issues`

```json
{
  "sales_order_id": "88888888-aaaa-bbbb-cccc-888888888888",
  "reference": "DISPATCH-001",
  "notes": "First dispatch",
  "issued_at": "2026-07-24T10:00:00Z",
  "lines": [
    {
      "item_id": "22222222-aaaa-bbbb-cccc-222222222222",
      "location_id": "33333333-aaaa-bbbb-cccc-333333333333",
      "quantity_issued": "3.000"
    }
  ]
}
```

The sales order must be confirmed or partially fulfilled. Items must exist on the order, quantity cannot exceed order remaining quantity, and the location must have sufficient available stock.

Reverse with `POST /api/v1/sales/issues/{id}/reverse`:

```json
{
  "reason": "Dispatch posted in error"
}
```

Reversal appends compensating stock and reopens order fulfillment progress. The original issue and stock movement remain immutable.

## 10. Business workflows the UI should enforce

### Inventory foundation

```text
Create unit
  -> create item using base_unit_id
  -> create location
  -> post opening adjustment or receive Purchase stock
  -> read balances / availability / valuation
```

Do not offer receipt, issue, or transfer as a choice on the direct stock-adjustment form. Those operations have dedicated APIs and business rules.

### Purchase lifecycle

```text
Supplier
  -> Purchase Order: draft
  -> approve
  -> approved
  -> one or more receipts
  -> partially_received / received
  -> optional receipt reversal
  -> receipt progress reopens when required
```

Disable the Approve button after the first successful call. A repeated call returns a lifecycle conflict.

### Sales lifecycle

```text
Customer
  -> Sales Order: draft
  -> confirm
  -> confirmed
  -> one or more stock issues
  -> partially_fulfilled / fulfilled
  -> optional issue reversal
  -> fulfillment progress reopens when required
```

Before posting an issue, show current `/inventory/availability` for the selected item/location. The backend remains the authority and may still return a conflict if stock changed concurrently.

## 11. Navigation and authorization UX

The frontend can build module navigation from `/api/v1/modules`:

```ts
const enabled = new Set(modules.map((module) => module.id));

const navigation = [
  enabled.has("inventory") && { label: "Inventory", to: "/inventory" },
  enabled.has("purchase") && { label: "Purchase", to: "/purchase" },
  enabled.has("sales") && { label: "Sales", to: "/sales" },
].filter(Boolean);
```

There is no current endpoint that returns the authenticated tenant user's complete role/permission set. Therefore:

- Module navigation can be hidden accurately.
- Action buttons cannot yet be pre-authorized from a permission manifest.
- The backend must remain authoritative.
- Handle `permission_denied` inline and show a stable forbidden state.
- Do not hide a backend error by treating it as an empty list.

The seeded allowed user has every currently seeded Inventory, Purchase, and Sales permission. The seeded no-access user can log in but business APIs should return `permission_denied`.

## 12. Error-handling policy

Recommended centralized behavior:

| Error code/category | UI action |
|---|---|
| `invalid_tenant_login`, `invalid_platform_login` | Keep login form open; show generic credential error |
| `platform_login_locked` | Disable retry temporarily and show lock message |
| `erp_session_invalid`, `platform_session_invalid` | Clear the matching session and return to the matching login |
| `permission_denied` | Stay logged in; render forbidden action/page |
| `module_not_enabled` | Refresh module entitlements and leave the module route |
| Validation `400` codes | Associate message with form/line when possible |
| Conflict `409` codes | Explain stale lifecycle/stock state and refetch |
| `500` or network error | Keep user input, show retry, avoid duplicate submission |

Mutation buttons should be disabled while a request is pending. Lifecycle actions and document creation are not supplied with client idempotency keys, so careless automatic retries can create duplicates.

## 13. Security checklist for FE review

- [ ] Tenant and platform sessions use different state and storage keys.
- [ ] The client never sends `X-User-ID`.
- [ ] The client never decodes a session token.
- [ ] Tokens are absent from logs, analytics, URLs, and error-report metadata.
- [ ] Tenant ID comes from the authenticated session state, not an editable request field after login.
- [ ] A tenant switch performs a new tenant login; it does not only replace `X-Tenant-ID`.
- [ ] The platform token is never sent to ERP routes.
- [ ] The ERP token is never sent to control-plane routes.
- [ ] The app distinguishes invalid session, disabled module, and denied permission.
- [ ] Decimal strings remain strings until formatted for display.
- [ ] Destructive/compensating lifecycle actions require user confirmation.
- [ ] Production networking uses TLS and a same-origin reverse proxy or an explicitly reviewed CORS policy.

## 14. External HRMS/custom-SaaS boundary

The current endpoints above authenticate the native ERP and control-plane UIs. They are not yet the common handoff protocol discussed for external modules.

Specifically, the current backend does not expose:

- an external-module handoff creation endpoint
- a one-time handoff-code exchange endpoint
- an audience-bound external-module assertion
- external identity-link resolution
- tenant provisioning for a legacy single-tenant SaaS

Until those endpoints exist:

- Do not put `X-ERP-Session` into a query string.
- Do not forward the ERP token to HRMS or another SaaS.
- Do not call an external module “integrated SSO” merely because the email matches.
- A link from ERP to an external module is navigation only; that module performs its own current login.

When the handoff protocol is implemented, FE should request a short-lived, one-time handoff from the ERP backend and navigate with that one-time artifact. FE should never manufacture or translate identity assertions itself.

## 15. Definition of done for the first FE integration

The initial frontend integration is complete when all of these work:

1. Vite proxy routes `/api` and `/control` correctly.
2. Allowed tenant user can log in and reload the page without losing a valid session.
3. `/auth/me` restores the user and invalid sessions return to login.
4. Module navigation is derived from `/api/v1/modules`.
5. Inventory unit, item, location, and opening adjustment can be created in dependency order.
6. Purchase order approval and receipt update inventory.
7. Sales order confirmation and issue update fulfillment and stock.
8. No-access user sees a forbidden state instead of a logout loop.
9. Tenant logout revokes the server session.
10. Platform login and tenant/plan/module screens use only the platform client.
11. Tokens never appear in URLs or logs.
12. The UI does not claim external-module SSO support before the handoff API exists.
