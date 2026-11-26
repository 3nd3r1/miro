---
id: TASK-2
title: Initialize Go module and stub miro/miro-agent Nix packages
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
ordinal: 2000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Milestone 0 requires the Go module and both static CLI binaries (miro for the hub, miro-agent for offline sites) to build as Nix flake packages via buildGoModule so later milestones can add bundle, status, and agent logic without packaging friction.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 go.mod exists at repo root with cmd/miro and cmd/miro-agent compiling with basic CLI entrypoints
- [x] #2 flake.nix exposes packages.x86_64-linux.miro and packages.x86_64-linux.miro-agent built with buildGoModule
- [x] #3 nix build .#miro and nix build .#miro-agent succeed and produce runnable binaries
- [x] #4 make check (golangci-lint and go test) passes
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Create go.mod and minimal cmd/miro and cmd/miro-agent entrypoints with unit tests.
2. Add packages.x86_64-linux.miro and packages.x86_64-linux.miro-agent to flake.nix via buildGoModule.
3. Verify nix build .#miro, nix build .#miro-agent, binary execution, and make check.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Verified make check (0 lint issues, all Go unit tests pass) and nix build .#miro / .#miro-agent producing runnable static binaries printing 0.1.0.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Initialized go.mod, stub cmd/miro and cmd/miro-agent binaries with unit tests, and exposed both as buildGoModule packages in flake.nix.
<!-- SECTION:FINAL_SUMMARY:END -->
