# Repository Instructions

This repository builds `pccli`, an unofficial Go CLI for Planning Center. Design a reliable user journey rather than a thin wrapper around API endpoints. There is no backward-compatibility requirement for the replaced song-library interface: use the current contract directly, without legacy commands, environment aliases, or serializers.

## Before changing behavior

- Read the relevant code and tests, current official Planning Center documentation, and useful community workflow notes. Compare command behavior with mature product-oriented CLIs when choosing a new interface.
- Start from the operator's task. Keep commands shallow: `pccli services RESOURCE ACTION`, with explicit parent flags instead of API-shaped nesting.
- Confirm request fields, types, paths, actions, response shapes, API version, and readback capabilities independently. Do not infer a write capability from a read endpoint. Keep research and implementation rationale in contributor guidance, design documents, issues, or PRs; public docs teach users.
- Preserve the supported Services API version (`2018-11-01`) explicitly. Unknown capabilities need evidence or an honest limitation, not a guessed request or silent fallback.

## Ownership and architecture

- `internal/cli`: Cobra grammar, flag presence, usage errors, human product views, and the `pccli/v1` machine envelope. Construct dependencies lazily so help and version need no credentials.
- `internal/services`: typed Services models and adapter, complete identity resolution, visibility, parent scope, mutation validation, workflow composition, readback, and recovery outcomes. Keep these policies usable without Cobra.
- `internal/planningcenter`: Basic Auth, HTTP/origin checks, shared JSON:API envelopes, bounded GET retries, cancellation, and streaming upload. Never automatically replay writes.
- `internal/config`: documented environment/dotenv precedence and validation.
- `internal/output`: terminal-safe text, width-aware formatting, tables, styles, and JSON encoding primitives. Product detail composition stays with the CLI presenter.
- Prefer the current package boundaries over new framework layers or endpoint-mirroring interfaces.

## Selection and mutation rules

- Exclude hidden songs by default everywhere, including direct IDs, searches, and parent resolution. `--visibility visible|hidden|all` selects existing records; `--hidden=true|false` mutates song state.
- IDs remain strings. Support explicit `id:` and `name:` selectors. Reads prefer exact case-insensitive names, then a unique literal partial target; writes and parent flags require exact names or IDs. Never guess on duplicate names.
- Every arrangement requires a song. Every key requires song and arrangement. Never infer a first/default arrangement. Archived selection requires `--include-archived`.
- Identity resolution must exhaust candidate pages independently of display limits. Validate response type, IDs, known fields, relationships, visibility, duplicate IDs, and cursor origin/path/filter/progress. Failed or malformed lookups cause zero writes or uploads.
- Preserve mutation presence: omitted means unchanged; explicit empty/zero/false/empty array is transmitted if valid, otherwise rejected. Use typed inputs; no arbitrary attribute passthrough. Validate the complete input before network calls, including non-finite numbers and conflicting fields.
- Deletes require `--yes` and still validate identity, scope, and capabilities. Tag replacement sends the whole desired set and respects required/single-selection groups.
- Distinguish verified changes, accepted write-only actions, rejection, and unknown commitment. Never claim an import completed merely because its request was accepted. Preserve upload receipts and destination context after later failure.

## Output and configuration

- Human output is concise and composed: use hierarchy, aligned labels, useful groups, and visible IDs. Suppress noisy default false/zero state when it adds no value. Sanitize remote text before terminal rendering.
- Every data command supports `--json`. Use one versioned envelope, string IDs, explicit values, and honest completeness metadata. Errors go to stderr; stdout must not contain success-looking data before fetching and encoding finish. Propagate writer failures and cancellation.
- Keep non-interactive automation first. Add prompts only for an established user need, with equivalent flags.
- Never read, print, or commit real `.env` secrets. Use isolated test directories and mock fixtures. Update `.env.example` when configuration changes.
- Personal Access Tokens use HTTP Basic Auth with `PCCLI_CLIENT_ID` and `PCCLI_CLIENT_SECRET`. Only documented `PCCLI_*` configuration names are supported. Precedence is environment, project `.env`, user `~/.pccli/.env`, defaults.

## Tooling and dependencies

- Use Go and the exact tool versions pinned in `.mise.toml` (currently Go 1.27.3). Use Cobra, Viper/godotenv, Charmbracelet Lip Gloss v2, GoReleaser, and VHS unless there is a concrete reason to change.
- Prefer maintained libraries for solved problems. Check current stable releases and their migration requirements before dependency updates; do not blindly run broad upgrades as a feature prerequisite.
- After dependency changes run `go mod tidy`, inspect the graph/diff, and validate affected behavior. Do not commit generated binaries under `bin/`.

## Verification and review

- Run `go test ./...` before finishing code changes; run `go vet ./...` and build for substantial changes. Use request-aware fixtures for endpoint/query/payload assertions.
- Test human and JSON output, error routing/exit codes, empty results, incomplete pagination, ambiguity, wrong parents, visibility, presence-preserving patches, late failures, recovery, cancellation, and writer failures as relevant.
- Prove zero side effects on invalid or incomplete selection. Keep useful existing regressions. Use meaningful behavior tests rather than tests that merely mirror implementation.
- Use `PCCLI_MOCK=true` for demos/screenshots. Static mock routes do not prove real mutations or provider integrations work.
- Review generated/AI-assisted changes yourself. Report exactly what was validated, and distinguish mock/request-contract tests from live account evidence. Do not claim exhaustive coverage or live validation without evidence.
- Use Conventional Commits so Release Please can maintain changelogs, tags, and releases.

## Documentation and demos

- Update user docs in the same change as user-visible behavior. Keep README concise; public Pages content under `docs/` covers installation, authentication, setup-first quick start, workflows, fields, and command reference. Make the unofficial status clear.
- Check examples against actual help and mock execution. Keep research trails and architecture rationale out of public product docs.
- Update the affected `scripts/*demo.tape` recordings whenever demonstrated commands change; use the isolated recorder, readable step pauses, and a final hold. Keep page captions and text alternatives synchronized. Record read-only mock journeys unless a stateful mock explicitly supports the demonstrated writes. Generated demos are built from tapes, not edited by hand.
- For banner changes edit `scripts/banner.ansi`, run `mise run banner`, and build; keep the embedded copy synchronized.
