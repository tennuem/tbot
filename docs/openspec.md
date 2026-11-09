# OpenSpec in tbot

This project uses [OpenSpec](https://openspec.dev/docs/setup) 1.16.1 with the
`spec-driven` schema. Specifications and changes live alongside the code.
OpenSpec is development tooling; the bot binary and runtime image do not need it.

## Setup

From the repository root:

```fish
make openspec-install
make openspec-init
make openspec-validate
```

The install target uses OpenSpec's official standalone installer with a pinned
release and checksum verification. It installs `bin/openspec`, which is ignored
by Git. Node.js and a global OpenSpec installation are not required.

Run the CLI through the project wrapper:

```fish
./scripts/openspec --version
./scripts/openspec doctor
./scripts/openspec list
./scripts/openspec list --specs
```

The wrapper finds the repository root, uses `bin/openspec`, and keeps OpenSpec's
machine configuration and bundled-schema cache in `.cache/openspec/`. This folder
is ignored by Git and Docker. Telemetry and automatic update checks are disabled
for these project commands. Existing shell startup files and machine-wide agent
settings are not edited.

## AI tool integration

`make openspec-init` generates the core workflows for all three selected tools:

| Tool | Skills | Commands |
| --- | --- | --- |
| Codex | `.agents/skills/openspec-*` | Skills, e.g. `$openspec-propose` |
| OpenCode | `.opencode/skills/openspec-*` | `/opsx-propose`, `/opsx-apply`, `/opsx-sync`, `/opsx-archive`, etc. |
| Oh My Pi | `.omp/skills/openspec-*` | `/opsx-propose`, `/opsx-apply`, `/opsx-sync`, `/opsx-archive`, etc. |

The six installed workflows are explore, propose, apply, update, sync, and
archive. Their generated skill names include `openspec-apply-change`,
`openspec-update-change`, and `openspec-archive-change`. Restart or refresh your
agent session after adding them; in Codex Desktop, select the installed workflow
from Skills. See the [official tool matrix](https://openspec.dev/docs/supported-tools).

Commit the generated integrations with the specs. Re-running init refreshes them
and preserves the existing `openspec/config.yaml`. Do not edit generated workflow
files manually. Project-specific guidance belongs in `AGENTS.md` and the
configuration's `context`, `rules`, and `operations` fields.

Generated workflows use bare `openspec` in examples. The repository's `AGENTS.md`
directs agents to use `./scripts/openspec`, so they use the pinned local CLI.

To select another set explicitly:

```fish
make openspec-init OPENSPEC_TOOLS=codex,opencode,oh-my-pi
```

## Existing behavior and new work

The initial main specs describe behavior verified against the implementation at
commit `81f9122`:

- `track-resolution`: providers, normalization, discovered links, partial failure.
- `track-cache`: persistent aliases, immediate replies, bounded refresh, provenance.
- `sharing-history`: events, HTML extraction, preview/import/export, unknown IDs.
- `telegram-interface`: group commands, ping, list pages, HTML links, service logos.
- `runtime-configuration`: Yandex proxy, persistent SQLite, initialization, logs.

This is an initial baseline, not a reconstructed history of past proposals or
approvals. There are no fabricated completed changes. Treat a mismatch between
a spec and source as something to resolve, not as proof the code already does it.

For a new OpenSpec change:

1. Explore the relevant source, tests, and main specs when needed.
2. Use the propose workflow to create the proposal, delta specs, design, and tasks.
   A request for a proposal is planning work; inspect its scope and acceptance
   scenarios before asking for implementation.
3. Use the apply workflow for implementation authorized by the user. Follow the
   project's Go engineering and external-action approval rules.
4. Run focused checks, then the required Go checks and `make openspec-validate`.
   Verify the implementation against the scenarios; syntax validation alone is
   not an implementation review.
5. Synchronize the delta specs and archive the completed change. Pushes, PRs,
   deployments, and production history imports retain their usual authorization
   requirements.

For example, in Codex's skill-capable chat:

```text
$openspec-propose <description of a scoped change>
$openspec-apply-change <change-name>
$openspec-sync-specs <change-name>
$openspec-archive-change <change-name>
```

For OpenCode and Oh My Pi, the corresponding command names are `/opsx-propose`,
`/opsx-apply`, `/opsx-sync`, and `/opsx-archive`.

Artifacts are written in Russian while preserving OpenSpec's structural headings
and `SHALL`/`MUST` keywords. Never put production credentials, chat exports, or
personal history into specifications or changes.

## Checks and updates

`make openspec-validate` validates all active changes and main specs in strict,
non-interactive mode. GitHub Actions runs the same target before the Go checks.
The validator checks artifact structure; Go tests and review check implementation.

```fish
./scripts/openspec update --dry-run --json
make openspec-validate
```

For an intentional OpenSpec upgrade, update the pinned version in `Makefile`,
install it, regenerate integrations, review the diff, and rerun validation.
