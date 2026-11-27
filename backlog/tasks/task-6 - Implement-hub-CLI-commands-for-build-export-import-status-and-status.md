---
id: TASK-6
title: Implement hub CLI commands for build, export, import-status, and status
status: To Do
assignee: []
created_date: '2026-11-26T11:37:00Z'
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
- [ ] #1 `miro build <site>` builds the site NixOS system closure via `nix build` and outputs the resulting store path
- [ ] #2 `miro export <site> <outdir>` writes an unsigned bundle containing `cache/` (via `nix copy --to file://...`), `models/<sha256>` (without embedding weights in the NixOS closure), and `manifest.json` with an incremented sequence number and file checksums, and supports pre-built closure/model inputs for offline test execution
- [ ] #3 `miro import-status <dir>` validates and ingests a site status bundle into local JSON hub state
- [ ] #4 `miro status` prints desired versus actual state per site (sequence, system path, model sha256, health, and pending exported bundles)
- [ ] #5 Go unit tests cover hub state persistence, bundle export assembly, status import, and status output, and `make check` and `nix build .#miro` pass
<!-- AC:END -->
