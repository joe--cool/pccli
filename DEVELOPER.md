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

Read [AGENTS.md](AGENTS.md) for the behavioral rules and the [command reference](docs/commands.md) for the public interface.

## Follow a request through the code

pccli supports Services song-library maintenance: songs, arrangements, keys, attachments, tags, and scheduling context. Plans, teams, people, and scheduling administration are outside this interface. Keep commands product-first and resource-first, with explicit parents and consistent actions.

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

For `--json --all`, Services walkers retain one page plus O(N) ID/hash metadata for consistency checks. Explicit song sorting uses bounded private disk runs. The CLI stages the encoded result in a private temporary file and publishes it only after traversal succeeds. Walk callbacks can run before a later failure; a new caller must preserve that staging boundary. Propagate cancellation and clean up temporary files. A failure during the final stdout copy can still leave partial bytes and must exit nonzero.

## Preserve operator intent

Typed pointer fields preserve omitted versus explicitly supplied mutation values. CLI binders use flag presence, not zero-value checks. Validate the entire input before lookup or upload. API field types can be surprising: alternate keys are a string containing JSON, import fields are strings, and some accepted action fields have no readable equivalent.

Resolve names from complete candidate sets. Page failure, changed filters, malformed data, ambiguity, inconsistent visibility, or contradictory scope must stop the workflow before side effects. Display limits do not cap identity resolution. Revalidate scoped destinations before attaching an upload receipt. Do not automatically replay writes after an unknown result. Multi-request workflows are not remote transactions: pagination does not create a snapshot, and destination rechecks cannot eliminate concurrent changes.

Services version `2018-11-01` is explicit. Verify each changed capability against the current official documentation; do not turn a read association into a write promise without evidence. Separate API acceptance, readable value verification, and asynchronous provider completion. Attachment updates can accept write-only `song_part` and attachment-type fields without confirming their effects through readback. CCLI-only song creation can return a resolved title rather than the numeric lookup string; verify the documented import outcome instead of requiring literal title equality.

## Maintain API capability evidence

[testdata/contracts/services-song-library.json](testdata/contracts/services-song-library.json) is a versioned test fixture, derived from the official Services OpenAPI document linked in its `source` field. It records assignable Song, Arrangement, Key, and Attachment create/update fields, wire types, CLI flags, API version, and retrieval date. Keep that provenance: it explains the contract being tested.

When adding or changing an assignable field, independently check the upstream schema and endpoint, then update the typed input, flag binding, fixture, user field reference, and wire/readback tests together. [coverage_test.go](internal/cli/coverage_test.go) compares the inventory with CLI flag types and typed Song/Arrangement/Key inputs; changing the fixture merely to match the implementation would defeat the check. The field inventory supplements endpoint and lifecycle tests; it does not prove account permissions or provider completion.

## Test the contract

```sh
go test ./...
go vet ./...
go build -o bin/pccli ./cmd/pccli
```

Tests should prove decisions and wire behavior: method, path, filters, pagination, field presence/types, version, Basic Auth, scoped parent checks, response validation, readback, upload recovery, cancellation, and no automatic write replay. Assert **zero writes/uploads** on incomplete or ambiguous selection. Include malformed/null attributes and duplicate IDs, not only happy-path fixture data.

CLI tests cover both plain and JSON results, parse errors before credential loading, stdout/stderr separation, exit codes, empty collections, false/zero values, terminal controls, and writer failures. Update affected command examples in the same change. Coverage reports can guide inspection, but a percentage is not proof that a workflow is safe. [workflow_journeys_test.go](internal/cli/workflow_journeys_test.go) exercises stateful plain/JSON lifecycles, including all three attachment scopes; [export_test.go](internal/services/export_test.go) covers paging, sorting, payload retention, cancellation, and temporary-file cleanup. Use `go test -race ./...` when changing shared state or cancellation behavior.

Run examples with isolated configuration when testing credential-free behavior. Never inspect real dotenv secrets. For a mock read from the repository:

```sh
PCCLI_MOCK=true go run ./cmd/pccli services songs list
PCCLI_MOCK=true go run ./cmd/pccli --json services arrangements list --song 1001 --all
```

Mock mode uses the configured fixture relative to the working directory. Static routes are not a stateful Planning Center emulator and cannot establish real provider imports, permissions, or mutation effects. Report request-contract and mock verification separately from authorized live validation. Live acceptance requires an explicitly authorized account and disposable records; synthetic tests cannot establish real permissions, provider licensing, upload expiry, or production effects.

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

Public `docs/` content teaches installation, authentication, setup-first quick start, workflows, commands, metadata fields, and machine output. Keep README brief and make the unofficial status visible. This guide holds current architecture, contract maintenance, and contributor workflows; AGENTS.md holds working rules.

Keep change-specific audits, AI review prompts/reports, comparisons, transition plans, branch status, and completed validation logs in issues or pull requests. Extract lasting decisions into the relevant maintained documentation rather than committing a parallel status packet. Keep executable evidence—contract fixtures, behavior tests, and demo sources—in the repository.

The read-only documentation videos use mock fixtures:

```sh
mise run vhs
```

This builds the binary and runs `scripts/record-demos.sh`. The recorder copies only the binary, tapes, and mock fixture into a temporary directory, with an empty home and a clean environment. It publishes the MP4/poster set only after all recordings finish. Pages uses the same script with its host-built binary and a network-disabled container built from `scripts/Dockerfile.demos` (pinned VHS plus Debian’s jq package) before building Jekyll. Local VHS needs `ttyd`, FFmpeg, and `jq`; the recorder checks them before starting. The container build uses an empty context, so no checkout or dotenv files are sent as build input.

| Tape | Placement | Teaching goal |
| --- | --- | --- |
| `scripts/demo.tape` | Front page | Find a song, understand its versions, select a key |
| `scripts/attachment-demo.tape` | File-placement workflow | Compare song, arrangement, and key destinations |
| `scripts/json-demo.tape` | JSON reference | Inspect completeness, then extract fields for scripts |

Keep recordings focused on one journey. Use the shared `demo-settings.tape` for readable type and dimensions. Introduce each step with a short shell comment, hold the command for one second before Enter, and leave small results for 7–10 seconds and denser detail views for 12 seconds. Clear the screen only between completed steps. End with a useful result and a 10-second hold. Include a representative result poster, an accurate caption, and equivalent readable steps on the page.

Embed videos through `docs/_includes/terminal-demo.html`: native controls, no autoplay or looping, and no media preload. This gives readers time to pause, inspect, and replay. The recordings are silent; the page text supplies the same instructions and expected outcomes. MP4s/posters are ignored generated assets, regenerated by Pages. To review locally, record, inspect the posters and result frames, and watch the timing/ending in a browser before committing tape changes.

Do not demonstrate writes with static mock responses that cannot represent changed state. Check every taped command against the current CLI. Never record from a credential-bearing working directory.

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
