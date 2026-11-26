---
id: TASK-3
title: Add inference NixOS module for llama.cpp
status: Done
assignee:
  - '@agy'
created_date: '2026-11-26T09:20:00Z'
updated_date: '2026-11-26T09:23:00Z'
labels: []
milestone: m-0
dependencies:
  - TASK-1
priority: high
type: feature
ordinal: 3000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Offline sites run local CPU-only LLM inference with model weights stored outside the Nix store under /var/lib/miro/models/<sha256>. A dedicated NixOS module encapsulates the llama.cpp server configuration so per-site configs stay concise and consistent.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 nix/modules/inference.nix defines a NixOS module configuring the local OpenAI-compatible llama.cpp server on CPU
- [x] #2 Model weights are loaded from /var/lib/miro/models/<sha256> outside the Nix store
- [x] #3 Module exposes options for model sha256, listen host/port, and extra server flags
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Create nix/modules/inference.nix wrapping nixpkgs services.llama-cpp for CPU-only inference.
2. Load model weights from /var/lib/miro/models/<sha256> with an assertion forbidding /nix/store paths.
3. Expose options for modelSha256, modelsDir, host, port, and extraSettings.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Evaluated nixosModules.inference with pkgs.nixos and verified generated llama-server ExecStart uses /var/lib/miro/models/<sha256>, host/port/extraSettings options work, and /nix/store paths are rejected by assertion.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added nix/modules/inference.nix wrapping nixpkgs services.llama-cpp for CPU-only inference with weights under /var/lib/miro/models/<sha256> and exported it as nixosModules.inference in flake.nix.
<!-- SECTION:FINAL_SUMMARY:END -->
