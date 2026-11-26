---
id: TASK-1
title: Set up Nix flake and devShell
status: Done
assignee:
  - '@agy'
created_date: '2026-11-26T09:20:00Z'
updated_date: '2026-11-26T09:21:00Z'
labels: []
milestone: m-0
dependencies: []
priority: high
type: chore
ordinal: 1000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The repository currently has only shell.nix and no flake. Milestone 0 needs a pinned root flake.nix so NixOS site configurations, Go packages, and nix flake check can be built reproducibly while keeping the existing nix-shell developer workflow working.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 flake.nix and flake.lock exist at repo root with pinned nixpkgs
- [x] #2 devShells.x86_64-linux.default provides Go, golangci-lint, gnumake, nodejs_22, and puts tools/bin on PATH
- [x] #3 nix flake metadata succeeds without errors
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Create flake.nix pinning nixpkgs and defining devShells.x86_64-linux.default matching shell.nix.
2. Generate flake.lock and verify nix flake metadata and nix develop.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Locked nixpkgs to 2786a2a72088ecf32aea5b3ec52099e6a7cff78f (nixos-unstable). Verified nix flake metadata and nix develop with go, golangci-lint, make, node, and backlog on PATH.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added root flake.nix and flake.lock pinning nixpkgs and exposing devShells.x86_64-linux.default via shell.nix. Verified nix flake metadata and nix develop.
<!-- SECTION:FINAL_SUMMARY:END -->
