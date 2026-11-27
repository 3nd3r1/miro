---
id: TASK-5
title: Define unsigned bundle and status schemas with checksum verification
status: Done
assignee:
  - '@agy'
created_date: '2026-11-26T11:37:00Z'
updated_date: '2026-11-27T12:15:00Z'
labels: []
milestone: m-1
dependencies: []
priority: high
type: feature
ordinal: 1000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Both the hub CLI (miro) and offline site agent (miro-agent) need shared, deterministic formats for outbound bundles and return status bundles. Because air-gapped transfer is a dumb directory copy, bundles must list SHA-256 checksums for model and cache files so partial or corrupted copies are detected and ignored until complete. Signing and anti-replay are deferred to m-2.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 internal/bundle defines the outbound manifest schema (bundle ID, site ID, monotonic sequence number, target system path, optional base system path, model artifacts with sha256 and size, cache file checksums, and informational created_at) with JSON read/write helpers
- [x] #2 internal/bundle verifies bundle completeness and integrity by checking site ID match and SHA-256 checksums of all listed model and cache files, distinguishing incomplete or corrupt bundles so callers can ignore partial copies until complete
- [x] #3 internal/status defines the unsigned status bundle schema (site ID, last applied sequence, current system path and generation, current model sha256, health, rollout events, and agent log tail) with atomic write and read helpers
- [x] #4 flake.nix includes ./internal in Go package sources so nix build .#miro and nix build .#miro-agent succeed
- [x] #5 Go unit tests cover round-trip serialization, valid verification, missing or truncated files, checksum mismatches, and site ID mismatches, and make check passes
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Create internal/bundle with Manifest, ModelArtifact, FileChecksum schemas, deterministic JSON read/write helpers, SHA-256 file hashing, and bundle verification distinguishing incomplete (missing/truncated) vs corrupt vs wrong-site errors.
2. Create internal/status with StatusBundle, HealthStatus, RolloutEvent schemas and atomic JSON write/read helpers.
3. Add ./internal to flake.nix goSrc fileset.
4. Write unit tests for internal/bundle and internal/status and verify with make check and nix build .#miro .#miro-agent.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implemented internal/bundle and internal/status with unit tests; updated flake.nix goSrc; verified with make check and nix build --no-link .#miro .#miro-agent.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added unsigned outbound bundle manifest and checksum verification in internal/bundle, unsigned status bundle schema with atomic read/write in internal/status, and included ./internal in flake.nix Go sources. Verified with make check (0 lint issues, all unit tests passing) and nix build --no-link .#miro .#miro-agent.
<!-- SECTION:FINAL_SUMMARY:END -->
