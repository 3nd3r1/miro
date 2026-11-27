---
id: TASK-6
title: Implement hub CLI commands for build, export, import-status, and status
status: Done
assignee:
  - '@agy'
created_date: '2026-11-26T11:37:00Z'
updated_date: '2026-11-27T12:18:00Z'
labels: []
milestone: m-1
dependencies:
  - TASK-5
priority: high
type: feature
ordinal: 2000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
To complete the hub side of the offline delivery loop in m-1, operators and tests need miro commands to build a site NixOS closure, export a self-contained unsigned bundle (manifest, Nix binary cache, and external GGUF model weights), ingest returning status bundles, and inspect desired versus actual site state persisted locally without adding non-stdlib dependencies.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 `miro build <site>` builds the site NixOS system closure via `nix build` and outputs the resulting store path
- [x] #2 `miro export <site> <outdir>` writes an unsigned bundle containing `cache/` (via `nix copy --to file://...`), `models/<sha256>` (without embedding weights in the NixOS closure), and `manifest.json` with an incremented sequence number and file checksums, and supports pre-built closure/model inputs for offline test execution
- [x] #3 `miro import-status <dir>` validates and ingests a site status bundle into local JSON hub state
- [x] #4 `miro status` prints desired versus actual state per site (sequence, system path, model sha256, health, and pending exported bundles)
- [x] #5 Go unit tests cover hub state persistence, bundle export assembly, status import, and status output, and `make check` and `nix build .#miro` pass
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add hub state persistence (HubState, SiteState, DesiredState, ActualState, pending exported bundles) in cmd/miro/state.go using stdlib JSON and atomic writes.
2. Implement Hub operations (Build, Export, ImportStatus, FormatStatus) in cmd/miro/hub.go with an injectable command runner for nix build and nix copy, supporting --system and --model flags for offline test execution.
3. Wire build, export, import-status, and status subcommands in cmd/miro/main.go.
4. Add unit tests covering state persistence, build, export, import-status, and status formatting, and verify with make check and nix build --no-link .#miro.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implemented hub state persistence and miro build/export/import-status/status subcommands with unit tests; verified with make check and nix build --no-link .#miro.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Implemented hub CLI commands (build, export, import-status, status) and local JSON state persistence in cmd/miro using only the Go standard library. Supports pre-built --system and --model inputs for offline VM testing, deterministic unsigned bundle export with cache/model checksums, pending bundle tracking, and human/JSON status output. Verified with make check (0 lint issues, all tests passing) and nix build --no-link .#miro.
<!-- SECTION:FINAL_SUMMARY:END -->
