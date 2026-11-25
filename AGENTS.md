# miro

Apollo-style continuous delivery of LLM inference to air-gapped NixOS sites. Read `PROJECT.md` and the direction (`backlog doc view doc-1`) first.

## Layout

No application code exists yet. Planned layout (starts in Milestone 1):

- `cmd/miro/`: hub CLI (`build`, `export`, `import-status`, `status`, `serve`).
- `cmd/miro-agent/`: offline site agent triggered by a systemd path unit.
- `internal/bundle/`: manifest schema, ed25519 signing, verification, checksums.
- `internal/status/`: signed status bundle format.
- `internal/eval/`: deterministic eval runner (temperature 0).
- `nix/`: NixOS modules (`agent.nix`, `inference.nix`), per-site configs (`sites/`), and `nixosTest` (`tests/e2e.nix`).
- `evals/`: eval suites.
- `keys/dev/`: dev-only keys, marked insecure.
- `backlog/`: project board, managed with the `backlog` CLI.
- `tools/`: repo tooling (`tools/bin/backlog`).

## Rules

- Sites have no network: no NTP, no DNS, no package or model downloads, no sockets to the hub.
- Transfer is a dumb directory copy. Partial bundles must be detected by checksum and ignored until complete.
- Never trust wall-clock time on sites. Ordering and anti-replay use monotonic sequence numbers.
- Every bundle in both directions is signed. Sites verify Nix store signatures (`require-sigs = true`) and ed25519 manifest signatures.
- Model weights stay outside the Nix store, stored content-addressed under `/var/lib/miro/models/<sha256>`.
- Keep Go code small and readable. Prefer the standard library. Add dependencies only when they save real complexity.
- Do not build items from the parked v2 list in `PROJECT.md` unless asked.

## Tests

- Go unit tests for bundle verification, signing, anti-replay, eval runner, and the agent state machine.
- NixOS VM test (`nix flake check`) for the full offline courier loop.

## Workflow

- Tools come from `nix-shell`, which puts the pinned `backlog` CLI on `PATH`.
- The user owns the direction (`doc-1`): the goal, milestone order, and pivots. Only the user edits `doc-1`. Bring them only direction calls, as one question with a recommendation, and record each answer with `backlog decision create`.
- Agents own the tasks: create, split, update, and close them without asking. Work the current milestone and build only what it needs.
- When starting a task, set its assignee to the agent doing the work (`@codex`, `@opencode`, `@claude`, or `@agy`).
- Keep docs short and honest about what is real vs simulated.
- Commits: `feat:`, `fix:`, `chore:`, `docs:`, `build:`, `test:`, optional scope like `feat(bundle):`. Short lowercase subject, no body, no co-authors or trailers. One logical change per commit.

<!-- BACKLOG.MD GUIDELINES START -->
<!-- backlog.md-instructions-version: 1.55.3 -->
<CRITICAL_INSTRUCTION>

## Backlog.md Workflow

This project uses Backlog.md for task and project management.

**At the beginning of each conversation in this project, run `backlog instructions overview` before answering or taking action. Re-read it only if you have not read it yet in the current conversation.**

Use the overview to decide whether to search, read, create, or update Backlog tasks.

Before task lifecycle actions, read the matching detailed guide:
- `backlog instructions task-creation` before creating or splitting tasks
- `backlog instructions task-execution` before planning, changing status or assignee, adding a plan or implementation notes, or implementing task work
- `backlog instructions task-finalization` before checking acceptance criteria, writing final summaries, or moving tasks to terminal statuses

Use `backlog <command> --help` before running unfamiliar commands. Help shows options, fields, and examples.

Do not edit Backlog task, draft, document, decision, or milestone markdown files directly. Use the `backlog` CLI so metadata, relationships, and history stay consistent.

</CRITICAL_INSTRUCTION>
<!-- BACKLOG.MD GUIDELINES END -->
