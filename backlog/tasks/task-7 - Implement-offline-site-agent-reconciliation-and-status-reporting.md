---
id: TASK-7
title: Implement offline site agent reconciliation and status reporting
status: To Do
assignee: []
created_date: '2026-11-26T11:37:00Z'
labels: []
milestone: m-1
dependencies:
  - TASK-5
priority: high
type: feature
ordinal: 3000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Offline sites receive bundles through dumb directory copies into inbound/ with no network access and unreliable wall-clock time. The site agent must safely reconcile bundles using persisted state, ignore partial copies until checksums pass, import the Nix closure and model weights, activate the new NixOS generation, check local inference health, and atomically write a status bundle for the return trip. Signing, anti-replay rejection, rollback, and eval gates belong to m-2..m-4.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 `miro-agent` scans `inbound/`, verifies site ID and file checksums, and leaves incomplete or partial bundles unapplied without advancing persisted state so they are processed once the copy completes
- [ ] #2 For a valid unapplied bundle, `miro-agent` imports the Nix closure with `nix copy --from file://<bundle>/cache <system-path>` (unsigned for m-1), copies model weights atomically into `/var/lib/miro/models/<sha256>`, and activates the generation via `nix-env --set` on `/nix/var/nix/profiles/system` and `switch-to-configuration switch`
- [ ] #3 `miro-agent` persists applied state on disk so re-runs and restarts are idempotent and do not re-activate an already-applied bundle
- [ ] #4 `miro-agent` checks the local llama.cpp health endpoint within a timeout and atomically writes a status bundle to `status-out/` with site ID, sequence, current system path and generation, model sha256, health, recent rollout events, and agent log tail
- [ ] #5 Go unit tests cover partial bundle deferral, full apply flow, idempotent re-runs, wrong-site rejection reporting, and status emission, and `make check` and `nix build .#miro-agent` pass
<!-- AC:END -->
