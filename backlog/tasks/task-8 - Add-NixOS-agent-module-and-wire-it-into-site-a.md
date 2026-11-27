---
id: TASK-8
title: Add NixOS agent module and wire it into site-a
status: Done
assignee:
  - '@agy'
created_date: '2026-11-26T11:37:00Z'
updated_date: '2026-11-27T12:24:00Z'
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
- [x] #1 `nix/modules/agent.nix` defines a `miro.agent` NixOS module exported as `nixosModules.agent` in `flake.nix` with options for site ID, transfer directories (`inbound`, `status-out`), state/models directories, health endpoint, and agent package
- [x] #2 The module provisions transfer and state directories, a `miro-agent.service` oneshot unit, and a `miro-agent.path` unit watching `inbound/` so arriving bundles trigger reconciliation automatically
- [x] #3 The module configures offline Nix settings on the site with empty substituters and `require-sigs = false` for unsigned m-1 bundles
- [x] #4 `nix/sites/site-a.nix` imports and enables `miro.agent` for `site-a`, and the existing `site-a-smoke` check continues to pass
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Factor miro and miro-agent Go package definitions into nix/packages.nix so flake.nix and nix/modules/agent.nix share the exact same derivation without duplication.
2. Create nix/modules/agent.nix defining miro.agent options, tmpfiles transfer/state/models directories, offline nix.settings (empty substituters, require-sigs = false for m-1), miro-agent.service oneshot (restartIfChanged/stopIfChanged = false), and miro-agent.path watching inbound/.
3. Export nixosModules.agent in flake.nix and wire miro.agent into nix/sites/site-a.nix.
4. Verify nixosConfigurations.site-a evaluates/builds and nix flake check (including site-a-smoke) passes.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Factored Go packages into nix/packages.nix, added nix/modules/agent.nix with offline Nix settings, tmpfiles, miro-agent.service, and miro-agent.path, exported nixosModules.agent, wired into nix/sites/site-a.nix, and verified nix flake check (all 19 checks including site-a-smoke) passed.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added the miro.agent NixOS module in nix/modules/agent.nix (exported as nixosModules.agent in flake.nix) provisioning transfer/state/models directories, offline Nix settings (empty substituters, require-sigs = false for unsigned m-1 bundles), miro-agent.service oneshot (with restartIfChanged/stopIfChanged = false), and miro-agent.path watching inbound/. Wired miro.agent into nix/sites/site-a.nix and verified nix flake check passes including site-a-smoke.
<!-- SECTION:FINAL_SUMMARY:END -->
