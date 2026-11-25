# Miro

Apollo-style continuous delivery for LLM inference to air-gapped NixOS sites.

A connected hub holds the desired state for a fleet of sites and exports signed bundles. A courier moves each bundle across a one-way gap to an offline site. The site agent verifies the bundle, imports the Nix closure and model weights, switches NixOS generations, runs an eval gate, rolls back if anything fails, and writes a signed status bundle for the trip back. The hub imports status bundles and shows desired vs actual state for each site.

Keep it small and readable. The point is the air-gapped delivery loop, not production completeness.

## Constraints

- Sites have no network at all. No NTP, no DNS, no package fetches, no Hugging Face downloads, no telemetry. Everything a site needs travels in the bundle.
- Transfer is a dumb directory or disk copy moved by hand or by the test driver. There are no sockets between hub and site.
- Never trust wall-clock time on sites. Order bundles and block replays with monotonic sequence numbers.
- Every bundle in either direction is signed. A site rejects unsigned bundles, unknown keys, corrupt files, wrong site IDs, and sequence numbers at or below the last applied sequence.

## Stack

- Language: Go for the hub CLI, hub dashboard, and site agent. Single static binaries built with `buildGoModule`.
- Nix: flake at repo root with NixOS configurations per site.
- Inference: llama.cpp server (OpenAI-compatible API), CPU only, small GGUF models (0.5B to 3B). Use the nixpkgs NixOS module if it fits.
- Model weights: kept outside the Nix store. Stored as content-addressed files under `/var/lib/miro/models/<sha256>` and listed in the bundle manifest. Nix handles the OS, agent, inference server, and config.
- Nix signing: hub binary cache key (`nix-store --generate-binary-cache-key`). Sites trust only that public key via `trusted-public-keys` with `require-sigs = true`.
- Manifest signing: ed25519 from the Go standard library. The hub signs outbound manifests. Each site has its own keypair and signs its status bundles.
- E2E tests and demo: NixOS VM test (`nixosTest` / `runNixOSTest`) with a hub VM and 2 or 3 offline site VMs. The test driver acts as the courier via `copy_from_vm` and `copy_from_host`.

## Architecture

### Hub (connected side)

Desired state per site lives in `nix/sites/<site>.nix`: NixOS config, model artifact, eval suite, and release channel.

Commands:

- `miro build <site>`: builds the site's NixOS system closure with `nix build`.
- `miro export <site> <outdir>`: writes an outbound bundle:
  - `manifest.json`: bundle ID, site ID, sequence number, target system store path, expected base system path (for future deltas), model artifacts with sha256 and size, eval suite reference, and an informational `created_at`.
  - `manifest.sig`: ed25519 signature over `manifest.json`.
  - `cache/`: Nix binary cache from `nix copy --to file://...` of the signed system closure.
  - `models/`: model files named by sha256.
- `miro import-status <dir>`: verifies and ingests a site status bundle.
- `miro status`: prints desired vs actual state for every site.
- `miro serve`: server-rendered HTML dashboard showing each site's desired version, actual version, last contact (sequence number and import time), health, rollout history (applied, failed eval, rolled back), and exported bundles still waiting on status.

Hub state lives in a local SQLite file or plain JSON files.

### Transfer

Outbound bundles go to `transfer/outbound/<site>/` on the hub and land in `transfer/inbound/` on the site. Status bundles leave `transfer/status-out/` on the site and return to the hub.

Bundles are self-describing. A partially copied bundle fails checksum verification and is ignored until complete.

### Site agent (offline side)

A systemd service triggered by a systemd path unit watching `inbound/`. It runs a persisted state machine so crashes and restarts are safe:

1. Verify: check manifest signature, site ID, sequence number greater than last applied, and sha256 checksums of every model and cache file.
2. Import: run `nix copy --from file://<bundle>/cache <system path>`, then copy model files into `/var/lib/miro/models/<sha256>`.
3. Activate: set `/nix/var/nix/profiles/system` to `<system path>` and run `switch-to-configuration switch`.
4. Health check: wait for the local inference server to answer within a timeout.
5. Eval gate: run the bundle's eval suite against the local endpoint at temperature 0. Each case has a prompt and an expected substring or regex, and the suite has a pass threshold.
6. Commit or roll back: on pass, record the new version. On failure, roll back to the previous generation, confirm the old version is healthy, and record the failure reason.
7. Report: write a signed status bundle to `status-out/` with site ID, last applied sequence, current system path and generation, current model sha256, health, eval results, last N rollout events, and a short tail of agent logs (agent output only).
8. GC: keep the last N generations and models, then collect garbage.

Rejected bundles (bad signature, replay, wrong site, corrupt) are never applied, and the rejection is reported in the next status bundle.

## Planned layout

```
flake.nix
flake.lock
cmd/miro/                  hub CLI and dashboard
cmd/miro-agent/            site agent
internal/bundle/           manifest, signing, verification, checksums
internal/status/           status bundle format
internal/eval/             eval runner
nix/modules/agent.nix      NixOS module for the agent and path unit
nix/modules/inference.nix  inference server config
nix/sites/site-a.nix       per-site configs
nix/tests/e2e.nix          nixosTest: hub + offline sites + courier
evals/                     eval suites (YAML or JSON)
keys/dev/                  dev-only keys, marked insecure
docs/architecture.md       architecture doc with mermaid diagram
```

## Milestones

1. Skeleton: flake, Go module, `miro` and `miro-agent` build as Nix packages, one offline site VM serving a tiny GGUF model with llama.cpp.
2. One full loop: `miro export` builds a bundle, courier copies it, agent imports and activates it, agent writes status, `miro import-status` reads it. Unsigned for now.
3. Signing and anti-replay: Nix cache signing with `require-sigs`, ed25519 manifests, site keys, monotonic sequence numbers.
4. Rollback: health check failure triggers rollback to the previous generation.
5. Eval gate: eval suites in the bundle, rollback on eval failure.
6. Fleet: 2 or 3 sites on different models and versions, plus the `miro serve` dashboard.
7. E2E test: `nixosTest` running the demo scenario under `nix flake check`.
8. Docs: architecture diagram, quickstart (`nix run .#demo` if feasible), and air-gap gotchas found along the way.

## Demo scenario (e2e test)

1. Hub targets `site-a` and `site-b` at model v2. The v2 bundle for `site-b` has an eval suite it will fail (or an intentionally broken model).
2. Courier moves both bundles to the sites.
3. `site-a` upgrades and passes the eval gate. `site-b` upgrades, fails the eval gate, and rolls back to v1.
4. Courier carries both status bundles back to the hub.
5. Dashboard shows `site-a` on v2 (healthy) and `site-b` on v1 (rolled back, with reason).
6. Bonus: replay an older bundle to `site-a` and verify it is rejected.

## Parked (v2 and later)

Do not build these unless asked.

- Transfer and bundles: closure delta bundles (`nix-store -qR` diff), content-defined chunking for model weights, multi-volume split bundles, zstd compression, pure one-way mode (no return path), human review queue for status exports.
- Rollouts: release channels (stable, canary, dev), staged rollout rings, in-site canary, drift-tolerant maintenance windows, dependency constraints between models and inference runtime, latency and regression eval gates.
- Security and supply chain: SBOM (sbomnix) and SLSA provenance, offline key rotation and revocation, hub/site model policy engine, sops-nix or agenix secrets, TPM attestation, sequence-based bundle expiry.
- Hardware and scale: CUDA GPU sites, per-site hardware profiles, k3s + Zot OCI variant for comparison, multi-hop relay sites, real hardware demo on a mini PC or Raspberry Pi.
- Observability: per-site telemetry redaction rules, fleet rollout timeline and drift alerts, request and latency metrics in status bundles.
