---
id: TASK-11
title: Add dev-only keys and hub signing for Nix cache and outbound manifests
status: To Do
assignee: []
created_date: '2026-11-27T12:51:00Z'
labels: []
milestone: m-2
dependencies:
  - TASK-10
priority: high
type: feature
ordinal: 2000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
To sign outbound deliveries and verify returning site reports in development and VM tests, the repo needs clearly marked dev-only keys under keys/dev/ and hub CLI support to sign Nix binary cache paths and manifest.sig on export while verifying status.sig on import.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 `keys/dev/` contains dev-only Nix binary cache keypair, hub ed25519 keypair, and `site-a` ed25519 keypair with a README clearly marking them insecure and dev-only
- [ ] #2 `miro export` signs exported Nix binary cache store paths with the hub Nix signing key and writes `manifest.sig` with the hub ed25519 private key
- [ ] #3 `miro import-status` verifies `status.sig` against the configured site ed25519 public key and rejects unsigned, tampered, or unknown-key status bundles without modifying hub state
- [ ] #4 Go unit tests cover signed export assembly, signed status import, and rejection of unsigned, tampered, or wrong-key status bundles, and `make check` and `nix build .#miro` pass
<!-- AC:END -->
