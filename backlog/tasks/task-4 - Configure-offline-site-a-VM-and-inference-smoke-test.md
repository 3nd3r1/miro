---
id: TASK-4
title: Configure offline site-a VM and inference smoke test
status: Done
assignee:
  - '@agy'
created_date: '2026-11-26T09:20:00Z'
updated_date: '2026-11-26T09:27:00Z'
labels: []
milestone: m-0
dependencies:
  - TASK-2
  - TASK-3
priority: high
type: feature
ordinal: 4000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Milestone 0 finishes when an air-gapped NixOS site VM with no network can boot, load a tiny GGUF model (Qwen2.5-0.5B-Instruct Q4_K_M) placed under /var/lib/miro/models/<sha256>, and answer a local OpenAI-compatible inference request.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 nix/sites/site-a.nix defines an offline NixOS site configuration importing nix/modules/inference.nix with networking disabled
- [x] #2 A pinned fetchurl definition provides Qwen2.5-0.5B-Instruct Q4_K_M to the test driver without embedding weights in the site NixOS closure
- [x] #3 A nixosTest check in flake.nix boots site-a offline, places the GGUF model into /var/lib/miro/models/<sha256>, starts inference, and verifies an HTTP completion response on localhost
- [x] #4 nix flake check passes
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add nix/models.nix with pinned fetchurl for Qwen2.5-0.5B-Instruct Q4_K_M (used only by hub/test driver, never in site closure).
2. Add nix/sites/site-a.nix with networking disabled and miro.inference enabled.
3. Add nix/tests/smoke.nix nixosTest booting site-a offline, copying the GGUF into /var/lib/miro/models/<sha256>, and verifying a localhost chat completion.
4. Wire nixosConfigurations.site-a and checks.x86_64-linux.site-a-smoke in flake.nix and run nix flake check.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Verified nix flake check passes (site-a-smoke finishes in ~22.5s): site-a boots with only loopback interface (lo), loads Qwen2.5-0.5B-Instruct Q4_K_M from /var/lib/miro/models/74a4da8c9fdbcd15bd1f6d01d621410d31c6fc00986f5eb687824e7b93d7a9db, and answers /v1/chat/completions on 127.0.0.1:8080. Confirmed via nix path-info -r that the GGUF file is not in site-a's NixOS closure.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added nix/models.nix (hub/test-side fetchurl), nix/sites/site-a.nix (air-gapped NixOS site), nix/tests/smoke.nix (nixosTest), and wired nixosConfigurations.site-a and checks.x86_64-linux.site-a-smoke in flake.nix. nix flake check passes.
<!-- SECTION:FINAL_SUMMARY:END -->
