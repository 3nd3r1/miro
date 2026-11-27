---
id: TASK-13
title: Wire trusted keys and require-sigs into NixOS agent module and site-a
status: To Do
assignee: []
created_date: '2026-11-27T12:51:00Z'
labels: []
milestone: m-2
dependencies:
  - TASK-12
priority: high
type: feature
ordinal: 4000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
An air-gapped NixOS site must declaratively lock down Nix store trust to only the hub binary cache public key with require-sigs = true, and supply the hub ed25519 public key and site ed25519 private key to miro-agent.service.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 `nix/modules/agent.nix` exposes options for trusted hub Nix binary cache public keys, hub ed25519 public key, and site ed25519 private key, and passes the key paths to `miro-agent.service`
- [ ] #2 `nix/modules/agent.nix` sets `nix.settings.require-sigs = true` and `nix.settings.trusted-public-keys` to only the configured hub key(s) with empty substituters
- [ ] #3 `nix/sites/site-a.nix` configures `miro.agent` with the dev-only keys from `keys/dev/` and the existing `site-a-smoke` check continues to pass
<!-- AC:END -->
