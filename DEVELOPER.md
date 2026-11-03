# Developer Guide

## Start locally

Install the exact tools pinned in `.mise.toml`: Go 1.27.3, golangci-lint, GoReleaser, and VHS.

```sh
mise install
mise run test
mise run build
```

The Makefile mirrors test, build, lint, banner, demo, and VHS tasks. Without mise, use the pinned Go toolchain and run:

```sh
go test ./...
go vet ./...
go build -o bin/pccli ./cmd/pccli
```

Read [AGENTS.md](AGENTS.md) for the behavioral rules. The [architecture review](design/architecture-review/README.md) records the design and its historical baseline; current code and tests establish implemented behavior.

## Follow a request through the code

| Location | Responsibility |
| --- | --- |
| `cmd/pccli` | Entry point, signal context, process exit |
| `internal/cli` | Cobra resources/actions, explicit flag presence, lazy dependency construction, product views, machine envelope and errors |
| `internal/services` | Typed product inputs/models, Services adapter, pagination, identity/scope/visibility policy, workflows, mutation verification and recovery |
| `internal/planningcenter` | HTTP Basic Auth, origin/redirect checks, selected API version, JSON:API envelopes, bounded GET retries and streamed multipart upload |
| `internal/config` | Environment/dotenv precedence and validation |
| `internal/output` | Terminal-safe text, display-width formatting, styling, tables, raw JSON encoder |
| `testdata/mocks` | Static deterministic mock routes for reads and demos |

The CLI depends on Services workflows, which depend on the typed adapter and shared HTTP transport. Keep selection and mutation policy out of Cobra; keep endpoint mechanics out of command constructors. The small HTTP interface is a test seam, not a mirrored public workflow API.

The machine contract is `pccli/v1`. List metadata distinguishes returned limits from complete selections; detail output distinguishes requested empty components from unrequested expansions. JSON is staged before publication, and output failures remain errors. Human output suppresses default noise while preserving useful metadata and explicit IDs.

## Preserve operator intent

Typed pointer fields preserve omitted versus explicitly supplied mutation values. CLI binders use flag presence, not zero-value checks. Validate the entire input before lookup or upload. API field types can be surprising: alternate keys are a string containing JSON, import fields are strings, and some accepted action fields have no readable equivalent.

Resolve names from complete candidate sets. Page failure, changed filters, malformed data, ambiguity, inconsistent visibility, or contradictory scope must stop the workflow before side effects. Display limits do not cap identity resolution. Revalidate scoped destinations before attaching an upload receipt. Do not automatically replay writes after an unknown result.

Services version `2018-11-01` is explicit. Verify each changed capability against the current official documentation; do not turn a read association into a write promise without evidence. Separate API acceptance, readable value verification, and asynchronous provider completion.

## Test the contract

```sh
go test ./...
go vet ./...
go build -o bin/pccli ./cmd/pccli
```

Tests should prove decisions and wire behavior: method, path, filters, pagination, field presence/types, version, Basic Auth, scoped parent checks, response validation, readback, upload recovery, cancellation, and no automatic write replay. Assert **zero writes/uploads** on incomplete or ambiguous selection. Include malformed/null attributes and duplicate IDs, not only happy-path fixture data.

CLI tests cover both plain and JSON results, parse errors before credential loading, stdout/stderr separation, exit codes, empty collections, false/zero values, terminal controls, and writer failures. Update affected command examples in the same change. Coverage reports can guide inspection, but a percentage is not proof that a workflow is safe.

Run examples with isolated configuration when testing credential-free behavior. Never inspect real dotenv secrets. For a mock read from the repository:

```sh
PCCLI_MOCK=true go run ./cmd/pccli services songs list
PCCLI_MOCK=true go run ./cmd/pccli --json services arrangements list --song 1001 --all
```

Mock mode uses the configured fixture relative to the working directory. Static routes are not a stateful Planning Center emulator and cannot establish real provider imports, permissions, or mutation effects. Report request-contract and mock verification separately from authorized live validation.

## Update dependencies deliberately

Use Cobra for commands, Viper/godotenv for configuration, Lip Gloss v2 for styling/tables, and Go's HTTP/JSON/context packages for transport mechanics. Prefer a maintained library over custom infrastructure when it materially improves the workflow.

Before changing a dependency, inspect its current stable release, Go requirement, release notes, and major-version import path. Update selected modules deliberately, for example:

```sh
go list -m -u all
go get charm.land/lipgloss/v2@VERSION
go mod tidy
go test ./...
go vet ./...
```

Replace `VERSION` with the verified intended release. Review `go.mod` and `go.sum`; understand new transitive modules and remove unused dependencies. Use the official Go download metadata and support policy when changing the Go pin, then update `.mise.toml`, `go.mod`, and relevant docs together. Keep release/demo/lint tool versions reproducible. A vulnerability scan supplements testing:

```sh
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

Do not claim a vulnerability check passed unless it completed for the code being reviewed.

## Documentation, banner, and demos

Public `docs/` content teaches installation, authentication, setup-first quick start, workflows, commands, metadata fields, and machine output. Keep research/comparisons in design documents, issues, or PRs. Keep README brief and make the unofficial status visible.

The read-only demo uses mock fixtures:

```sh
mise run vhs
```

This builds the binary and records `scripts/demo.tape` to `docs/assets/library-demo.gif`. Pages also regenerates the demo before building Jekyll. Do not demonstrate writes with static mock responses that cannot represent changed state. Check every taped command against the current CLI.

The banner source is `scripts/banner.ansi`; the embedded copy is `internal/cli/banner.ansi`. After changing it:

```sh
mise run banner
mise run build
```

Do not commit binaries under `bin/` or hand-edit generated demo assets.

## Review and release

Contributors own the validity of AI-assisted output. Read the diff, explain the behavior and limits, run the appropriate checks, and do not hand maintainers unverified generated changes.

Use Conventional Commits: `fix:` for corrections, `feat:` for new behavior, and `!` or a `BREAKING CHANGE:` footer for breaking contracts. Release Please maintains the release PR, changelog, tag, and GitHub release; GoReleaser attaches cross-platform binaries and checksums when the release is created. Do not manually rewrite generated release content except to correct release notes before merge.

The release workflow uses `RELEASE_PLEASE_TOKEN` when configured and otherwise `GITHUB_TOKEN`. Token configuration belongs in repository secrets, never checked-in files or logs.
