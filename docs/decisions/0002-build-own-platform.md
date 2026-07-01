# ADR 0002: Build Own Platform, Study FOSS ERP Systems

## Status

Accepted

## Context

Several FOSS ERP systems exist, including:

- ERPNext
- Odoo Community
- iDempiere
- Dolibarr
- Tryton
- Apache OFBiz
- metasfresh

They are useful references, but the required product shape is specific:

- fully online SaaS
- superadmin control plane
- tenant licensing
- module subscriptions
- strong tenant isolation
- event-heavy module intercommunication
- future enterprise isolation options

## Decision

Build our own ERP SaaS platform.

Study FOSS ERP systems only for architecture/product references.

Do not fork or directly copy code/schema/UI from GPL/LGPL projects without deliberate license review.

## Reference study map

- ERPNext: product feel, document model, workflows, metadata-driven forms
- Odoo Community: module manifests, addons, dependency model
- iDempiere: enterprise ERP depth, accounting, multi-org thinking
- Dolibarr: simplicity and SME usability
- Apache OFBiz: ERP framework/service/entity thinking

## Consequences

The platform architecture remains ours.

Licensing obligations from FOSS projects must not accidentally infect proprietary code.

Conceptual learning is allowed.

Blind copying is not.
