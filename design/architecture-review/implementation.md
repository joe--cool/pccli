# Implementation and remaining gaps

Implemented on `feature/song-library-architecture`, based on `origin/main` at `76b6db2` (release 1.0.0). The earlier feature branch had already been squash-merged; continuing from it would have missed the release baseline. This record supersedes the original packet's implementation-status statements, not its historical evidence.

## Product direction and why

pccli is a product-first, resource-first workflow CLI. Commands start with `services`, then a shallow resource group and an action. Songs, arrangements, and keys each have their own commands; explicit parent flags select the context. There is no default arrangement, compatibility grammar, or API-path namespace to memorize.

This follows the mature CLI conventions assessed in the [original brief](brief.md) while preserving Planning Center's product concepts. Cobra owns parsing/help; Services workflows own identity, visibility, scope, validation, and write outcomes; a shared transport owns HTTP concerns. Typed inputs and a small transport seam are sufficient for this Go application. A general plugin framework, generated endpoint CLI, or separate interface per operation would add machinery without improving the operator journey.

The important simplification is consistent decisions: names never silently select duplicate records, list limits never truncate identity checks, omitted patch fields differ from explicit empty/zero/false, and machine output has one versioned envelope. Human detail output composes useful sections instead of dumping the API. Public instructions are setup-first and task-oriented.

## Completed song-library coverage

The agreed scope is complete song-library workflows in Services, not the entire Services product or every reachable API endpoint.

| Workflow | Implemented behavior |
| --- | --- |
| Songs | List/search, detail, create, update, delete; explicit visibility; CCLI metadata lookup and documented create import fields |
| Arrangements | List, detail, create, update, delete; archive/unarchive; sections; explicit song context |
| Keys | List, detail, create, update, delete; explicit song and arrangement; alternate-key metadata |
| Attachments | List, detail, local upload or remote link, receipt reuse, update and delete at song, arrangement **and key** scopes |
| Tags | Discover grouped tags, inspect song assignments, replace the complete assignment or explicitly clear it; group constraints validated |
| Catalogs | Tag-group and attachment-type discovery, using the read capabilities documented by the API |
| Scheduling context | Song schedules and last-scheduled items; the latter requires an explicit service-type ID |
| Metadata | Every assignable Song, Arrangement, Key and Attachment create/update attribute in the checked-in versioned capability snapshot has a typed CLI flag |
| Automation | `--json`, stable `pccli/v1` envelope, bounded lists and `--all`, explicit outcomes, cancellation, non-interactive mutation flags |

Current user instructions live in [the command reference](../../docs/commands.md), [workflow guide](../../docs/managing-your-song-library.md), and [field reference](../../docs/song-fields.md).

## Resolution of the audit gaps

- Complete pagination precedes identity decisions. Exact names and explicit IDs are shared across follow-on commands. Ambiguity is an error, including duplicates beyond the first page. Visibility and archive selection are independent of mutation values.
- Typed pointer patches preserve intent. Invalid input and empty patches fail before credentials, lookup, upload, or writes. Readable assigned values are checked after mutation; unsupported response shapes and contradictory parents fail explicitly.
- JSON is staged in private temporary files before publication. `--json --all` walks pages incrementally; explicit song sorting uses bounded private disk runs. Late page failures publish no partial success document. Memory still includes O(N) ID/hash metadata to detect inconsistent duplicates; explicit sorting consumes temporary disk. Traversal is not an atomic remote snapshot.
- Uploads stream from regular files. Destination state is revalidated before attachment creation. Failed association preserves the known upload receipt and recovery context. Writes are never automatically replayed; transport uncertainty and output failure after a committed write have explicit outcomes.
- Transport uses Basic Auth, a pinned Services API version, origin/redirect checks, bounded GET retries, context cancellation, validated JSON:API envelopes and safe errors. Server error bodies are not echoed to the terminal.
- Configuration has one documented environment vocabulary and validated values. Go, module dependencies, and build/demo tools have been refreshed; obsolete terminal-library dependencies were removed.
- README, public guides, field reference, contributor guide, repository instructions and the mock tape describe the same interface. Research and architecture rationale remain outside public documentation.

## API evidence and design refinements

The richer official [Services OpenAPI document](https://api.planningcenteronline.com/services/v2/open_api/2018-11-01), recorded in the capability snapshot as retrieved on 2026-11-02, resolves several uncertainties in the initial review. It documents nested attachment writes at all three scopes and additional assignable chart/import fields. A small [capability snapshot](../../testdata/contracts/services-song-library.json) records those eight assignable schemas and their CLI flags. A contract test checks the snapshot against flag types and typed Song/Arrangement/Key inputs, so omissions cannot quietly recur.

Attachment updates distinguish verified readable values from accepted write-only fields (`song_part` and attachment-type assignments). Provider import fields likewise represent accepted requests, not verified asynchronous completion. CCLI-only creation may return a resolved title rather than the numeric lookup string; verification accounts for that documented behavior without relaxing other assigned-field checks.

## Review and validation

Astra supplied the independent architecture assessment, alternatives, contracts, implementation sequence and stress cases; its completed report and reusable assignment remain in this packet. Separate implementation reviewers inspected the domain/transport and CLI behavior. Findings about scope checks, paging, streaming, output failures, CCLI lookup and flag parsing were addressed before completion. The architect and implementation agents have finished their assignments.

Executed validation includes:

- Full Go suite, race detector, `go vet`, module tidy/verification, Linux/macOS/Windows amd64 builds, and application vulnerability scanning; the completed vulnerability scan reported no vulnerabilities.
- Stateful synthetic HTTP lifecycle tests in plain and JSON modes: songs, arrangements, keys, and identical attachment filenames at all three scopes. Tests assert methods, paths, payload presence, parent IDs, resulting state and deletion.
- Negative tests for incomplete/hostile pagination, duplicate identities, malformed/null responses, hidden/archived selection, contradictory scopes, unknown outcomes, upload recovery, output failures, cancellation, and zero side effects on invalid selection.
- Streaming tests for late errors, private temporary files, cleanup, bounded payload retention, sorted multi-run merges and cancellation.
- Independent field-inventory checks, 73 concrete documentation examples checked against help, 54 local documentation links checked after the final contributor record, and all six mock demo commands executed with isolated configuration.
- VHS recording using the pinned image digest and disabled container networking; the rendered terminal frame was visually inspected. Generated media remain ignored and are regenerated by Pages. The subsequent focused video design and recording instructions are documented in DEVELOPER.md.

Tests and reviews used injected HTTP or isolated mock configuration. Real dotenv secrets were not inspected and no authenticated live account operations were performed.

## Remaining gaps and boundaries

1. **Live account acceptance:** synthetic contracts cannot establish account permissions, provider licensing, real upload expiry behavior, or production mutation effects. Any live acceptance pass must use an explicitly authorized account and disposable records. This is a validation gap, not a claim of observed production failure.
2. **Provider completion:** there is no job/status API contract here that proves imported charts or provider assets finished processing. Accepted import requests must not be advertised as completed imports. Write-only attachment fields are reported as accepted and unverified.
3. **Concurrent remote changes:** identity and destination rechecks reduce mistakes but cannot make a multi-request workflow transactional. Unknown outcomes require inspection before retry; neither writes nor uploads are blindly replayed.
4. **Product scope:** plans, teams, scheduling management, people, media administration and unrelated nested plan-item editing are outside the agreed song-library work. Catalog discovery does not invent catalog administration endpoints.
5. **Publication:** these changes are local commits. No branch push, pull request, release, or Pages deployment is implied by local verification. The generated demo has been verified locally; the hosted Pages build runs after publication.

Future changes should start from an operator task, verify the current API evidence, extend typed workflows, and add meaningful wire/lifecycle tests. Keep the field snapshot and public examples synchronized with any new capability rather than equating endpoint count or line coverage with product completeness.
