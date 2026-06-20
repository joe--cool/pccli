# Developer Guide

## Tooling

Use Go `1.26.4` or newer patch releases in the same major/minor line. `.mise.toml` pins the expected toolchain and common release/demo tools:

```sh
mise install
mise run test
mise run build
```

The Makefile mirrors the common commands for contributors who do not use mise.

## Project Shape

- `cmd/pccli`: executable entrypoint.
- `internal/cli`: Cobra command tree and command-specific behavior.
- `internal/config`: dotenv and environment configuration.
- `internal/planningcenter`: Planning Center JSON-API HTTP client.
- `internal/services`: Services-domain use cases.
- `internal/output`: JSON and terminal rendering.
- `testdata/mocks`: deterministic mock API responses.
- `scripts/banner.ansi`: source for the banner asset.
- `internal/cli/banner.ansi`: embedded copy used by the Go binary.

## Dependencies

Use maintained libraries for solved problems. Current choices:

- Cobra for subcommands and flags.
- Viper plus godotenv for env/config loading.
- Charmbracelet Lip Gloss for terminal-aware styling and tables.
- GoReleaser for cross-platform release binaries.
- VHS for terminal demos.

Before adding a dependency or building custom infrastructure, check the current ecosystem and use the latest compatible release. Run:

```sh
go get -u ./...
go mod tidy
go test ./...
```

## Releases

Use Conventional Commits for commit messages. Release Please reads the commit history, maintains the release PR, updates `CHANGELOG.md`, creates the version tag, and publishes the GitHub release. When a release is created, the same workflow runs GoReleaser to attach cross-platform binaries and checksums.

For best GitHub Actions behavior, configure a `RELEASE_PLEASE_TOKEN` repository secret with a fine-grained PAT that can write contents and pull requests. The workflow falls back to `GITHUB_TOKEN`, but GitHub suppresses some follow-up workflow triggers from events created with that token.

Use the standard SemVer mapping:

1. `fix:` for patch releases.
2. `feat:` for minor releases.
3. `feat!:` or a `BREAKING CHANGE:` footer for major releases.

Do not hand-edit generated release PR content unless correcting release notes before merge.

## AI-Assisted Contributions

AI tools are welcome, but contributors are responsible for the code they submit. Please do not open PRs that pass unverified AI output on to maintainers to debug or validate. Before submitting, read the generated changes, confirm them against the relevant API docs and surrounding code, run the tests, and be prepared to explain the behavior.

## Feature Standard

Each feature starts with a user journey, not an endpoint list. For Planning Center work, confirm the current official docs and relevant community experience before choosing the command shape. Decide whether the work belongs under an existing domain command or needs a new subcommand group.

## Command Design

Follow the shape used by mature product-oriented CLIs: start with the Planning Center product, then keep workflows resource-first and shallow. Prefer `pccli services songs list` over deeper API-shaped paths like `pccli services library songs list`.

Use this decision order:

1. Start with the Planning Center product: `services`, `people`, `giving`.
2. Add the user-facing resource or workflow noun: `songs`, `plans`, `donors`.
3. Put the action next: `list`, `show`, `create`, `update`, `archive`.
4. Use flags for filters and context instead of extra hierarchy.
5. Add more hierarchy only when users naturally distinguish the workflows that way.
6. Keep aliases out until there is released usage to preserve.

Default to read-only operations first. For write operations, require explicit flags for destructive changes, support `--json`, and design for non-interactive automation before adding prompts.

Branding belongs on the root intro/help screen and in documentation demos. Do not print the banner from resource commands, JSON output, scripts, or errors; command output should stay composable.

When changing the banner, edit `scripts/banner.ansi`, then run:

```sh
mise run banner
mise run build
```

The generated binary under `bin/` is ignored and should not be committed.

## Documentation Standard

Public documentation lives in `docs/` and is built by GitHub Pages. Treat it like product documentation for CLI users:

1. Start with the task users are trying to complete.
2. Keep pages short and navigable: installation, authentication, quick start, and command reference.
3. Link to official Planning Center docs for Planning Center-owned procedures, such as creating API credentials.
4. Include runnable commands and expected concepts, not internal research notes.
5. Do not publish comparisons to other CLIs, research logs, or implementation rationale in `docs/`; keep that in issues, PR notes, or developer guidance.
6. Make the unofficial status visible anywhere a new user may enter the docs.
7. Keep terminal demos generated from `scripts/*.tape`; do not hand-edit generated GIFs.

When command behavior changes, update the command reference and any affected quick-start examples in the same change.

The quick start should stay setup-first: install, authenticate, confirm the CLI runs, then choose a Planning Center product. Do not make one feature area look like the permanent entry point for the whole product.
