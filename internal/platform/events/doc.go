package events

// Package events will own internal event contracts.
//
// Core rule:
//
// APIs perform commands.
// Events announce facts.
// UI broadcasts invalidate views.
//
// Every tenant-owned event must carry tenant context.
