---
id: TASK-9
title: Add NixOS VM test verifying the unsigned end-to-end delivery loop
status: Done
assignee:
  - '@agy'
created_date: '2026-11-26T11:37:00Z'
updated_date: '2026-11-27T12:35:00Z'
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
- [x] #1 A `nixosTest` check in `flake.nix` boots `site-a` offline (only loopback interface) on an initial generation and exports an unsigned bundle for an updated `site-a` closure using `miro export`
- [x] #2 The test verifies that a partial bundle copied into `inbound/` is ignored by `miro-agent` until the remaining files arrive
- [x] #3 Once the courier finishes copying the bundle into `inbound/`, `miro-agent.path` triggers `miro-agent`, which imports the Nix closure and model weights, activates the new NixOS generation, confirms llama.cpp is healthy and answers a local completion request, and writes a status bundle to `status-out/`
- [x] #4 The courier copies the status bundle back to the hub, `miro import-status` ingests it, and `miro status` reports `site-a` on the updated generation and healthy
- [x] #5 `nix flake check` and `make check` pass
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add nix/tests/e2e.nix using pkgs.testers.runNixOSTest with a hub VM, an offline (loopback-only) site_a VM on generation v1, and a site_a_v2 target closure registered in hub's Nix DB via virtualisation.additionalPaths.
2. Exercise the full unsigned courier loop: miro export on hub -> courier copies partial bundle to site_a inbound/ and verifies miro-agent ignores it -> courier completes the bundle -> miro-agent.path triggers miro-agent -> closure import, model install, switch-to-configuration to v2, llama.cpp health & chat completion check, status bundle emitted -> courier returns status to hub -> miro import-status and miro status report site-a on v2 and healthy.
3. Wire checks.x86_64-linux.e2e-loop in flake.nix and verify nix flake check and make check pass.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Added nix/tests/e2e.nix and checks.x86_64-linux.e2e-loop; enabled system.switch.enable = true in nix/modules/agent.nix (required under nixosTest); disabled GRUB in e2e.nix QEMU direct-boot VMs; verified full courier loop with make check and nix flake check (e2e-loop completed in 72s, all checks passed).
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added nix/tests/e2e.nix (exported as checks.x86_64-linux.e2e-loop) running a full air-gapped courier loop between a hub VM and an offline loopback-only site-a VM: miro export -> partial bundle deferral -> complete bundle delivery via miro-agent.path -> nix copy closure import + GGUF model install + switch-to-configuration to v2 -> llama.cpp health & chat completion -> status return -> miro import-status and miro status verification. Also enabled system.switch.enable = true in nix/modules/agent.nix so switch-to-configuration is built under nixosTest. Verified with make check and nix flake check.
<!-- SECTION:FINAL_SUMMARY:END -->
