# ADR 0005: No Docker or Container-Based Local Development

## Status

Accepted

## Context

The development environment is Windows 11 with PowerShell 7+ and VS Code.

The project should not depend on Docker, Docker Compose, or container-based local infrastructure.

## Decision

Do not use Docker or containers for this project unless the human explicitly reverses this decision later.

Local development should prefer:

- native Go tooling
- native PostgreSQL installation or externally provided PostgreSQL
- PowerShell scripts
- simple environment files
- direct service binaries if needed later
- managed cloud services only when deliberately chosen

## Consequences

Do not add:

- Dockerfile
- docker-compose.yml
- container-only setup instructions
- container-dependent development workflow

Future infrastructure instructions must avoid container assumptions.

Redis, NATS, and object storage will not be introduced through containers.

They may be introduced later through native binaries, managed services, or postponed adapters.
