---
id: TASK-7
title: Implement offline site agent reconciliation and status reporting
status: Done
assignee:
  - '@agy'
created_date: '2026-11-26T11:37:00Z'
updated_date: '2026-11-27T12:22:00Z'
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
- [x] #1 `miro-agent` scans `inbound/`, verifies site ID and file checksums, and leaves incomplete or partial bundles unapplied without advancing persisted state so they are processed once the copy completes
- [x] #2 For a valid unapplied bundle, `miro-agent` imports the Nix closure with `nix copy --from file://<bundle>/cache <system-path>` (unsigned for m-1), copies model weights atomically into `/var/lib/miro/models/<sha256>`, and activates the generation via `nix-env --set` on `/nix/var/nix/profiles/system` and `switch-to-configuration switch`
- [x] #3 `miro-agent` persists applied state on disk so re-runs and restarts are idempotent and do not re-activate an already-applied bundle
- [x] #4 `miro-agent` checks the local llama.cpp health endpoint within a timeout and atomically writes a status bundle to `status-out/` with site ID, sequence, current system path and generation, model sha256, health, recent rollout events, and agent log tail
- [x] #5 Go unit tests cover partial bundle deferral, full apply flow, idempotent re-runs, wrong-site rejection reporting, and status emission, and `make check` and `nix build .#miro-agent` pass
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add persisted agent state (cmd/miro-agent/state.go) tracking last applied sequence, current system/generation/model, bounded rollout events, rejected bundle markers, and agent log tail without relying on wall-clock time.
2. Implement Agent reconciler (cmd/miro-agent/agent.go) with injectable Runner and HealthChecker: scan inbound/, defer incomplete bundles, report wrong-site/corrupt rejections, atomically install models, import Nix cache closure, activate via nix-env + switch-to-configuration, check local inference health, and atomically write status bundle to status-out/.
3. Wire miro-agent reconcile CLI flags in cmd/miro-agent/main.go.
4. Add unit tests covering partial bundle deferral, full apply flow, idempotent re-runs, wrong-site rejection reporting, and status emission, and verify with make check and nix build --no-link .#miro-agent.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implemented persisted agent state, inbound scanning with partial deferral and wrong-site rejection reporting, atomic model installation, unsigned Nix closure import, system profile activation, health polling, and atomic status bundle emission. Verified with make check and nix build --no-link .#miro-agent.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Implemented offline site reconciliation agent (miro-agent reconcile) in cmd/miro-agent with persisted sequence-based state, partial bundle deferral, wrong-site/corrupt rejection reporting, atomic model installation into /var/lib/miro/models/<sha256>, unsigned Nix closure import, NixOS generation activation, local health checks, and atomic status bundle output. Verified with make check (0 lint issues, all unit tests passing) and nix build --no-link .#miro-agent.
<!-- SECTION:FINAL_SUMMARY:END -->
