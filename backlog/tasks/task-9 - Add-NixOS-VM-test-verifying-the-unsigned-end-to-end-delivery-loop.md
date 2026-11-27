---
id: TASK-9
title: Add NixOS VM test verifying the unsigned end-to-end delivery loop
status: To Do
assignee: []
created_date: '2026-11-26T11:37:00Z'
labels: []
milestone: m-1
dependencies:
  - TASK-6
  - TASK-8
priority: high
type: feature
ordinal: 5000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Milestone 1 is only proven when the complete hub-to-courier-to-offline-site-to-hub loop runs in an air-gapped NixOS VM test under nix flake check, exercising real nix copy, systemd path triggering, switch-to-configuration activation, llama.cpp serving, and status import back on the hub.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A `nixosTest` check in `flake.nix` boots `site-a` offline (only loopback interface) on an initial generation and exports an unsigned bundle for an updated `site-a` closure using `miro export`
- [ ] #2 The test verifies that a partial bundle copied into `inbound/` is ignored by `miro-agent` until the remaining files arrive
- [ ] #3 Once the courier finishes copying the bundle into `inbound/`, `miro-agent.path` triggers `miro-agent`, which imports the Nix closure and model weights, activates the new NixOS generation, confirms llama.cpp is healthy and answers a local completion request, and writes a status bundle to `status-out/`
- [ ] #4 The courier copies the status bundle back to the hub, `miro import-status` ingests it, and `miro status` reports `site-a` on the updated generation and healthy
- [ ] #5 `nix flake check` and `make check` pass
<!-- AC:END -->
