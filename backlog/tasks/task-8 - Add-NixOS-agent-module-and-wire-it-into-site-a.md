---
id: TASK-8
title: Add NixOS agent module and wire it into site-a
status: To Do
assignee: []
created_date: '2026-11-26T11:37:00Z'
labels: []
milestone: m-1
dependencies:
  - TASK-7
priority: high
type: feature
ordinal: 4000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
An air-gapped NixOS site needs declarative provisioning of transfer and state directories, offline Nix settings, and automatic bundle reconciliation when a courier drops a bundle into inbound/. Encapsulating this in nix/modules/agent.nix keeps site configs minimal and reusable across the fleet in later milestones.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 `nix/modules/agent.nix` defines a `miro.agent` NixOS module exported as `nixosModules.agent` in `flake.nix` with options for site ID, transfer directories (`inbound`, `status-out`), state/models directories, health endpoint, and agent package
- [ ] #2 The module provisions transfer and state directories, a `miro-agent.service` oneshot unit, and a `miro-agent.path` unit watching `inbound/` so arriving bundles trigger reconciliation automatically
- [ ] #3 The module configures offline Nix settings on the site with empty substituters and `require-sigs = false` for unsigned m-1 bundles
- [ ] #4 `nix/sites/site-a.nix` imports and enables `miro.agent` for `site-a`, and the existing `site-a-smoke` check continues to pass
<!-- AC:END -->
