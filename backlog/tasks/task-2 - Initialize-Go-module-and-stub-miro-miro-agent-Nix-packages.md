---
id: TASK-2
title: Initialize Go module and stub miro/miro-agent Nix packages
status: To Do
assignee: []
created_date: '2026-11-26T09:20:00Z'
labels: []
milestone: m-0
dependencies:
  - TASK-1
priority: high
type: feature
ordinal: 2000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Milestone 0 requires the Go module and both static CLI binaries (miro for the hub, miro-agent for offline sites) to build as Nix flake packages via buildGoModule so later milestones can add bundle, status, and agent logic without packaging friction.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go.mod exists at repo root with cmd/miro and cmd/miro-agent compiling with basic CLI entrypoints
- [ ] #2 flake.nix exposes packages.x86_64-linux.miro and packages.x86_64-linux.miro-agent built with buildGoModule
- [ ] #3 nix build .#miro and nix build .#miro-agent succeed and produce runnable binaries
- [ ] #4 make check (golangci-lint and go test) passes
<!-- AC:END -->
