---
id: TASK-3
title: Add inference NixOS module for llama.cpp
status: To Do
assignee: []
created_date: '2026-11-26T09:20:00Z'
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
- [ ] #1 nix/modules/inference.nix defines a NixOS module configuring the local OpenAI-compatible llama.cpp server on CPU
- [ ] #2 Model weights are loaded from /var/lib/miro/models/<sha256> outside the Nix store
- [ ] #3 Module exposes options for model sha256, listen host/port, and extra server flags
<!-- AC:END -->
