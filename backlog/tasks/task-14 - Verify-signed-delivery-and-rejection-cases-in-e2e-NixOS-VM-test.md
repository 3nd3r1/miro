---
id: TASK-14
title: Verify signed delivery and rejection cases in e2e NixOS VM test
status: To Do
assignee: []
created_date: '2026-11-27T12:51:00Z'
labels: []
milestone: m-2
dependencies:
  - TASK-11
  - TASK-13
priority: high
type: feature
ordinal: 5000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Milestone 2 is only proven when the full air-gapped VM test exercises real Nix binary cache signature enforcement (require-sigs = true), bidirectional ed25519 signatures, and rejection of tampered, untrusted, and replayed bundles.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 `nix/tests/e2e.nix` exports a signed bundle from `hub` (signed Nix cache and `manifest.sig`), delivers it to offline `site-a` with `require-sigs = true`, activates `v2`, and imports the site-signed status bundle (`status.json` + `status.sig`) on `hub`
- [ ] #2 The test delivers a tampered/bad-signature bundle and a replayed older-sequence bundle to `site-a`, verifying neither alters the active generation and both rejections appear in the signed status bundle returned to `hub`
- [ ] #3 The test verifies `miro import-status` rejects a tampered status bundle without overwriting valid hub state
- [ ] #4 `nix flake check` and `make check` pass
<!-- AC:END -->
