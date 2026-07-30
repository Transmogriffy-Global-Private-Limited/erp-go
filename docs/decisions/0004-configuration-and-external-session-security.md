# ADR 0004: Configuration and external-session security boundaries

Status: Accepted

Date: 2026-07-30

## Context

The house-HRMS adapter needs trusted network configuration and server-side
provider sessions. Mixing deployment facts, tenant choices, user credentials,
and mapping rules in one configuration system would create secret leakage,
unclear ownership, and unsafe runtime behavior.

The ERP must restore external connections after restart without exposing the
provider session to the browser.

## Decision

Separate ownership as follows:

### Deployment configuration

Owns:

- registered provider name and implementation kind;
- base URL and allowed host;
- explicit timeouts;
- TLS-related policy where needed.

### ERP PostgreSQL

Owns:

- tenant/provider selection;
- workforce/provider mappings;
- connection status;
- encrypted provider sessions and expiry metadata;
- relevant audit records.

### Environment or established secret mechanism

Owns:

- encryption keys;
- database credentials;
- other deployment secrets.

External passwords and OTPs are transient inputs. Never persist or log them.
Never return a provider token or session to the browser. Encrypt provider
session material before PostgreSQL persistence.

The provider client uses trusted host allowlisting, TLS for remote connections,
explicit timeouts, bounded response sizes, validation, rate limits for
authentication operations, and secret-redacted errors.

Configuration must not support arbitrary request templates, response mappings,
scripts, JSONPath, or an authentication DSL. Invalid configuration fails
startup with an actionable error.

## Consequences

Positive:

- secrets have one explicit owner;
- tenant choices remain durable and auditable;
- connection state survives process restart;
- the browser cannot reuse a raw provider credential;
- clients cannot turn the adapter into an arbitrary outbound proxy.

Tradeoffs:

- encrypted records require a versioned envelope and key lifecycle;
- lost keys may require explicit reconnect operations;
- session expiry and provider rejection must map to canonical reconnect state;
- operations and backup procedures must preserve keys separately from data.

## Alternatives rejected

### Store provider sessions in browser storage

Rejected because it exposes the standalone session outside the ERP boundary and
prevents server-side isolation and auditing.

### Store sessions only in process memory

Rejected because connections would disappear on restart and process memory
would become an accidental source of truth.

### Put user credentials or mapping expressions in configuration

Rejected because it mixes durable user state and secrets with deployment facts
and creates an unsafe generic integration engine.

## Required follow-up decision

Before Step 05, approve the encryption envelope, algorithm, authenticated
metadata, key identifier, rotation behavior, and reconnect behavior for
undecryptable or retired-key sessions.
