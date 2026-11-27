---
id: TASK-10
title: Add ed25519 signing and verification to bundle and status packages
status: To Do
assignee: []
created_date: '2026-11-27T12:51:00Z'
labels: []
milestone: m-2
dependencies: []
priority: high
type: feature
ordinal: 1000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Air-gapped bundles cross untrusted transfer media in both directions. Both internal/bundle and internal/status need standard-library ed25519 signing and verification primitives so sites reject unsigned, tampered, or unknown-key outbound manifests and the hub rejects unsigned or forged status bundles before mutating state.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Provide stdlib `crypto/ed25519` key generation, read, and write helpers without adding non-stdlib dependencies
- [ ] #2 `internal/bundle` signs `manifest.json` into `manifest.sig` and verifies `manifest.sig` against a trusted hub public key, returning distinct errors for missing/unsigned, invalid signature, and unknown key
- [ ] #3 `internal/status` signs `status.json` into `status.sig` and verifies `status.sig` against a trusted site public key on read
- [ ] #4 Go unit tests cover valid signing/verification, missing signatures, tampered payloads, and wrong public keys, and `make check` passes
<!-- AC:END -->
