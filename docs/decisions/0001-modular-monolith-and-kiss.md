# ADR 0001: Modular monolith and KISS-first development

Status: Accepted

Date: 2026-07-30

## Context

The ERP is a new rebuild with broad long-term product ambitions but no current
need for independently deployed services. It must work on a non-admin Windows
development machine, remain straightforward to host, and allow features to be
implemented without navigating unrelated infrastructure or hidden framework
behavior.

KISS must reduce unnecessary complexity without removing security, durability,
recovery, observability, contracts, tests, or documentation.

## Decision

Begin with:

- one Go modular monolith;
- one deployable `erp-api` process;
- PostgreSQL for durable ERP-owned relational state;
- explicit module ownership inside the monolith;
- plain Go and direct dependency construction;
- explicit SQL through `pgx` unless a later approved requirement justifies a
  different approach;
- structured logging through `log/slog`;
- native PowerShell development and verification;
- loopback-only local listeners.

Modules own their data, decisions, and invariants. Sharing one process or
database does not authorize direct writes across module boundaries.

Public and internal boundaries must be direct and discoverable. Necessary
advanced logic stays isolated behind cohesive modules and testable functions.

Do not initially add microservices, Redis, NATS, Kafka, object storage, gRPC,
WebSockets, background workers, Kubernetes, containers, a dependency-injection
framework, or speculative extension points.

## Consequences

Positive:

- one deployment and one local startup path;
- ordinary in-process calls instead of network coordination;
- simpler transactions and debugging;
- fewer credentials, ports, processes, and recovery surfaces;
- service extraction remains possible if a concrete scaling, security,
  operational, or ownership requirement appears.

Tradeoffs:

- module boundaries require discipline because the compiler and deployment
  topology do not enforce service separation;
- one process shares a failure domain, so startup, shutdown, and module failure
  handling must remain explicit;
- extraction later may require contract and data-boundary work.

## Alternatives rejected

### Microservices from the start

Rejected because there is no present independent-scaling or operational-
isolation requirement sufficient to justify distributed transactions,
deployment coordination, network failure, and duplicated operational tooling.

### Generic plugin platform

Rejected because only one real external HRMS adapter is currently required.
Compiled packages and explicit registration are simpler and safer.

### Minimal code without complete guarantees

Rejected because omitting authorization, transaction integrity, recovery,
verification, or documentation exports complexity into failures and manual
operation rather than removing it.

## Revisit when

Reconsider process extraction only when measured scaling, security isolation,
operational isolation, distinct ownership, or incompatible workload behavior
cannot be handled cleanly inside the modular monolith.
