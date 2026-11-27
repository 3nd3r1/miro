---
id: TASK-12
title: Enforce signature verification and anti-replay in miro-agent
status: To Do
assignee: []
created_date: '2026-11-27T12:51:00Z'
labels: []
milestone: m-2
dependencies:
  - TASK-10
priority: high
type: feature
ordinal: 3000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Offline sites cannot trust wall-clock time or reach external key servers. The site agent must verify hub ed25519 manifest signatures before importing or activating anything, rely on Nix signature enforcement during closure import, reject replayed or stale sequence numbers, sign outbound status bundles with its site key, and report every rejection back to the hub.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 `miro-agent` verifies `manifest.sig` against the configured hub ed25519 public key and rejects unsigned, invalid-signature, or unknown-key bundles without installing models or importing closures
- [ ] #2 `miro-agent` imports Nix closures with `nix copy --from` without `--no-check-sigs` so Nix enforces `require-sigs = true` against site trusted public keys
- [ ] #3 `miro-agent` rejects replayed or stale bundles whose sequence number is at or below `last_applied_sequence` (while keeping re-runs idempotent for the already-applied bundle) and records every rejection as a rollout event
- [ ] #4 `miro-agent` signs `status.json` into `status.sig` with the site ed25519 private key when writing `status-out/`
- [ ] #5 Go unit tests cover unsigned, bad-signature, unknown-key, and replay rejection reporting plus signed status emission, and `make check` and `nix build .#miro-agent` pass
<!-- AC:END -->
