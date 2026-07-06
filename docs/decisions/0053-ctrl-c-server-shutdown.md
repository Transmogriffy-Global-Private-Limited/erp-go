# ADR 0053: Ctrl+C Native Server Shutdown

## Status

Accepted

## Context

The APIs subscribed to interrupt signals but did not react to the cancelled context. This consumed Ctrl+C without terminating the HTTP servers. `go run` also introduced an avoidable parent/child process boundary on Windows.

## Decision

Both HTTP APIs react to interrupt cancellation by calling `http.Server.Shutdown` with a ten-second timeout. Signal handling is restored when shutdown starts so a second Ctrl+C can force termination.

Native PowerShell launchers build ignored executables under `.local/bin` and invoke them directly in the current console. The worker already exits from its cancelled context and now uses the same direct-binary launcher model.

## Consequences

Every foreground API and worker can be stopped from its console with Ctrl+C without leaving a `go run` child process behind.
