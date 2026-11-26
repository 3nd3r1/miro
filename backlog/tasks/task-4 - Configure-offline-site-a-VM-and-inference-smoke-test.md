---
id: TASK-4
title: Configure offline site-a VM and inference smoke test
status: To Do
assignee: []
created_date: '2026-11-26T09:20:00Z'
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
- [ ] #1 nix/sites/site-a.nix defines an offline NixOS site configuration importing nix/modules/inference.nix with networking disabled
- [ ] #2 A pinned fetchurl definition provides Qwen2.5-0.5B-Instruct Q4_K_M to the test driver without embedding weights in the site NixOS closure
- [ ] #3 A nixosTest check in flake.nix boots site-a offline, places the GGUF model into /var/lib/miro/models/<sha256>, starts inference, and verifies an HTTP completion response on localhost
- [ ] #4 nix flake check passes
<!-- AC:END -->
