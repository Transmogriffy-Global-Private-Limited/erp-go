# ADR 0018: Platform Login and Sessions

## Status

Accepted

## Context

Control-plane authorization is DB-backed, but local development still sends X-Platform-User-ID directly.

This is better than trusting X-Platform-Role, but it is still not a login/session flow.

## Decision

Add platform login and platform sessions.

New endpoint:

POST /control/v1/auth/login

It accepts platform superadmin email/password and returns a session token.

Control-plane protected routes now accept:

- X-Platform-Session

Temporary fallback still accepted:

- X-Platform-User-ID

The fallback remains only to avoid breaking existing local verification scripts during the transition.

## Consequences

Platform control-plane access now has a session groundwork.

Future work should remove the X-Platform-User-ID fallback and move verification scripts to X-Platform-Session.
