# miro

Apollo-style continuous delivery of LLM inference to air-gapped NixOS sites.

Status: skeleton only. No Go or Nix code exists yet. The full design and milestone plan live in [PROJECT.md](PROJECT.md) and [backlog/docs/doc-1 - Direction.md](backlog/docs/doc-1%20-%20Direction.md).

## How it will work

A connected hub holds the desired state for a fleet of offline sites and exports signed bundles containing a Nix binary cache, GGUF model weights, and an eval suite. A courier copies each bundle across a one-way gap into a site's inbound directory.

The offline site agent verifies the ed25519 manifest signature, monotonic sequence number, and file checksums, imports the Nix closure and model weights, activates the new NixOS generation, and runs the eval suite against the local llama.cpp server. If health checks or evals fail, it rolls back to the previous generation. Either way, it writes a signed status bundle for the courier to carry back to the hub.

## Planned demo

1. Hub exports model v2 for `site-a` and `site-b` (`site-b` gets a failing eval suite).
2. Courier copies bundles to the offline NixOS VMs.
3. `site-a` upgrades and passes. `site-b` fails the eval gate and rolls back to v1.
4. Courier carries status bundles back. `miro serve` shows `site-a` on v2 and `site-b` rolled back to v1 with the reason.
5. Replaying an old bundle to `site-a` is rejected.

## Development

```sh
nix-shell
backlog board
```

## License

MIT
