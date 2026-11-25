---
id: doc-1
title: Direction
type: other
created_date: '2026-11-25T11:50:00Z'
---
Owned by the user. Agents read it and do not edit it.

## Goal

Show Apollo-style continuous delivery of LLM inference to air-gapped NixOS sites. A connected hub exports signed bundles (Nix binary cache, GGUF model weights, eval suite). A courier carries them across a one-way gap to offline sites. Each site agent verifies the bundle, activates the NixOS generation, runs an eval gate, rolls back on failure, and writes a signed status bundle for the return trip.

Keep it small and readable. Prefer working end-to-end loops over production extras.

## Where we are (2026-11-25)

Repo skeleton only. `PROJECT.md`, `README.md`, `AGENTS.md`, `shell.nix`, and Backlog.md are set up. No Go or Nix code exists yet.

## Milestones, in order

1. Skeleton (`m-0`): flake, Go module, `miro` and `miro-agent` build as Nix packages, one offline site VM serving a tiny GGUF model with llama.cpp.
2. One full loop (`m-1`): `miro export` builds a bundle, courier copies it, agent imports and activates it, agent writes status, `miro import-status` reads it. Unsigned for now.
3. Signing and anti-replay (`m-2`): Nix cache signing with `require-sigs`, ed25519 manifests, site keys, monotonic sequence numbers.
4. Rollback (`m-3`): health check failure triggers rollback to the previous generation.
5. Eval gate (`m-4`): eval suites in the bundle, rollback on eval failure.
6. Fleet and dashboard (`m-5`): 2 or 3 sites on different models and versions, plus the `miro serve` dashboard.
7. E2E test (`m-6`): `nixosTest` running the demo scenario under `nix flake check`.
8. Docs (`m-7`): architecture diagram, quickstart, and air-gap gotchas found along the way.

## Parked

See the parked v2 list in `PROJECT.md` (closure deltas, weight chunking, release channels, SBOMs, GPU sites, k3s variant, telemetry redaction). Do not build these unless asked.


