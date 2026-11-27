---
id: TASK-5
title: Define unsigned bundle and status schemas with checksum verification
status: To Do
assignee: []
created_date: '2026-11-26T11:37:00Z'
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
- [ ] #1 internal/bundle defines the outbound manifest schema (bundle ID, site ID, monotonic sequence number, target system path, optional base system path, model artifacts with sha256 and size, cache file checksums, and informational created_at) with JSON read/write helpers
- [ ] #2 internal/bundle verifies bundle completeness and integrity by checking site ID match and SHA-256 checksums of all listed model and cache files, distinguishing incomplete or corrupt bundles so callers can ignore partial copies until complete
- [ ] #3 internal/status defines the unsigned status bundle schema (site ID, last applied sequence, current system path and generation, current model sha256, health, rollout events, and agent log tail) with atomic write and read helpers
- [ ] #4 flake.nix includes ./internal in Go package sources so nix build .#miro and nix build .#miro-agent succeed
- [ ] #5 Go unit tests cover round-trip serialization, valid verification, missing or truncated files, checksum mismatches, and site ID mismatches, and make check passes
<!-- AC:END -->
