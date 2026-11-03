# pccli architecture review brief

Status: input proposal for independent review; not an implementation specification.
Reviewed checkout: `d8c596895cdff92b54ce91d89ae61335e5125eb8`.
Prepared: 2026-11-02 (America/Los_Angeles).

Owner clarification during review: **There is no adoption and nothing needs to be backward compatible.** Optimize for a coherent interface and future adoption. Breaking command and JSON changes are permitted; no legacy aliases, deprecation cycle, compatibility adapters, or adoption investigation is needed. This clarification supersedes the initial migration uncertainty in the proposal. Astra received it before completing the report.

## Assignment and desired outcome

The owner asked whether the project is clean, whether its APIs were actually simplified, and whether the CLI is simple, understandable, and consistent with established CLIs. They now want a current-state overview, a better setup with a defense of its choices, a gap inventory, and an independent Astra architecture review that gives the implementing agent a coherent vision and direction in one pass.

This is a design assignment. Do not implement the refactor, publish anything, open a PR, access live organization data, or read `.env` files. Produce a concrete architecture and sequenced plan. The architect should challenge this proposal rather than endorse it by default. Finish and stop after delivering the review.

## Product intent and constraints

- Unofficial Go CLI for Planning Center, distributed as a single binary to church staff, worship planners, volunteers, and operators maintaining song libraries. Scripts are a first-class use case.
- A user-journey CLI, not comprehensive API endpoint coverage. Current scope is Services music: songs, arrangements, keys, attachments. Other products are future extension points, not current feature commitments.
- Full repository rules: [AGENTS.md](../../AGENTS.md), [DEVELOPER.md](../../DEVELOPER.md).
- Product first, shallow resource/action commands; add hierarchy only for a genuine user-facing ambiguity.
- Default concise human output; all data commands support `--json`; no banner in data output. Preserve useful detail with composed sections.
- Hidden resources excluded by default, including names and IDs; explicit selection needed to include them. No guessing when records are ambiguous.
- Non-interactive flags first; explicit confirmation for destructive operations.
- Keep Go 1.26.4, Cobra, Viper/godotenv, Charmbracelet, GoReleaser, VHS, and the release pipeline unless there is a concrete reason to change them.
- Personal Access Tokens use HTTP Basic Auth with `PCCLI_CLIENT_ID` and `PCCLI_CLIENT_SECRET`. Do not read, expose, or commit real credentials.
- Research/architecture stays outside public `docs/`. Implementation changes to commands must update user docs and VHS examples together.
- No local tags were returned by `git tag --list`; release manifest says `0.0.0`. The owner subsequently confirmed no adoption and no compatibility requirement. Choose the best command and output contracts without preserving legacy behavior.

## Current state

### Public command surface

| Current shape | Behavior |
| --- | --- |
| `services songs list [filters]` | List/search via flags; `--limit` actually supplies API `per_page`. |
| `services songs search QUERY` | Separate title-search entry point, overlapping list `--search`; default limit 10 rather than 25. |
| `services songs show SONG [ARRANGEMENT]` | Song details, arrangement list, selected/default arrangement, selected keys. |
| `services songs create/update/delete` | Song mutations; delete requires `--yes`. |
| `services songs arrangements SONG [ARRANGEMENT]` | Implicit list/show action; has explicit create/update/delete children. |
| `services songs keys SONG [ARRANGEMENT]` | Implicit list action; has create/update/delete children. Optional arrangement inferred. |
| `services songs attachments SONG --arrangement ... --key ...` | Attachment list; parent context uses flags. |
| `services songs attach SONG [source flags]` | Attachment creation; exactly one file/upload-ID/URL/content source. |

Names and numeric IDs share positional inputs. Reads allow unique partial matches; writes require exact names or IDs. Arrangement and key parent context is sometimes positional and sometimes flags. Optional arrangement selection prefers the first case-insensitive `Default`, otherwise the first returned record. `--hidden` means hidden-only filtering, permission to include hidden targets, or the new hidden state depending on the command.

Strengths: useful song detail composition, familiar top-level CRUD verbs, non-interactive execution, JSON data output, IDs in lists, Basic Auth, explicit deletion confirmation, no unnecessary library namespace, product documentation and generated-demo tooling already present.

### Internal structure

```text
cmd/pccli/main.go
  -> internal/cli (Cobra parsing + validation + resolution + orchestration + rendering)
       -> internal/services.Library (primarily typed endpoint adapters and DTO mapping)
            -> internal/planningcenter.Client (HTTP, auth, JSON, errors, GET 429 retries, upload)
       -> internal/output (terminal/JSON primitives)
       -> internal/config (env/dotenv loading)
```

Important source map (line numbers apply to the reviewed SHA):

| File / symbol | Why the architect should read it |
| --- | --- |
| `internal/cli/songs.go` (1,476 lines) | All resource commands, resolution, mutation flags, detail orchestration, domain presentation. |
| `resolveSongID` ~1010 | Ten-result broad title lookup, local exact/fuzzy matching, hidden policy, ID checks. |
| `resolveArrangement`, `selectArrangement`, `resolveKey` ~1069–1163 | Repeated resolution policies; arrangements/keys have 100-result caps; numeric keys bypass fetching. |
| `defaultArrangement` ~1184 | Implicit first-match/first-record selection. |
| `newSongsUpdateCommand` ~300 | Visibility selection coupled to whether mutation flag changed. |
| `internal/cli/root.go` | Concrete Library construction, app context, global JSON flag. |
| `internal/cli/errors.go` | Usage errors exit 2; most errors exit 1; top-level stderr formatting. |
| `internal/services/library.go` (654 lines) | Broad low-level HTTP-shaped API interface, CRUD, DTOs, mapping, mutation payloads, upload/attach sequencing. |
| `collection`, `single`, `resource` ~442 | Duplicate JSON:API definitions; collection drops links/meta. |
| mutation helpers ~608 | String/number/slice zero values silently omitted. |
| `internal/planningcenter/jsonapi.go` | Another envelope implementation, with links/meta; production Services does not use it. |
| `internal/planningcenter/client.go` | Request construction, retry waits, API errors, fully buffered multipart upload, no explicit API version. |
| `internal/planningcenter/mock.go` | Static route matching; ignores queries and request bodies. |
| `internal/{cli,services,planningcenter,config}/*_test.go` | Existing verification and its limits. |
| `docs/commands.md`, `docs/managing-your-song-library.md` | Complete existing user contract, including current implicit defaults. |

The issue is responsibility placement, not simply file length. Transport DTOs and product models may reasonably differ; the duplicate transport envelopes and discarded completeness information do not have a clear benefit. A generic HTTP interface inside an adapter is not intrinsically wrong; exposing HTTP mechanics to workflow policy would be.

## Evidence and gap register

Labels: **reproduced** = executed against a synthetic local API, not production; **source** = directly visible in code; **contract gap** = needs API confirmation, not a demonstrated remote failure; **design** = product choice, not automatically a defect.

| ID | Priority / evidence | Current issue and impact | Required acceptance condition |
| --- | --- | --- | --- |
| G1 | High / reproduced + source | `delete Grace --yes` fetched the first 10 contains matches, found one exact Grace, and sent DELETE for ID 1 although a second exact Grace existed behind `links.next`. All Services list decoders discard links/meta. Arrangement/key resolution has similar page caps. | Complete relevant candidate enumeration or a proven uniqueness query; stop on incomplete lookup; no write on ambiguity. Display limit is never a resolution limit. |
| G2 | Medium / reproduced + source | `update 1 --author=""` sent `{"data":{"type":"Song","attributes":{}}}` and exited 0 against the fake server. Empty strings, zero numbers, empty sequences disappear. | Preserve absent vs explicit empty/zero/false/list; reject invalid clears before HTTP; send valid clears exactly. |
| G3 | Medium / reproduced + source | `update Grace --hidden=true` searched `where[hidden]=true`, missing visible Grace. Numeric IDs behave differently. `--hidden=false` also searches hidden-only because the flag changed. | Visibility selection independent of new hidden value; same name/ID policy across operations. |
| G4 | Medium / reproduced + source | `keys create 1 --name Lead --start G` chose arrangement 10 when both 10 and 11 were named Default. Multiple non-default records also fall back to first. | Explicit parent for ambiguous writes; never infer a target solely from result order or non-unique display name. |
| G5 | Medium / source | Demo mock ignores query/body, returns 200 on all matched routes, and never changes state. Tests can pass while filters/payloads are wrong. | Keep simple demo mode if useful; behavioral tests inspect actual method/path/query/body, pagination, failures, and absence of unsafe writes. |
| G6 | Low / reproduced + source | `songs list unintended-argument` succeeds and ignores the argument. Song create has no positional restriction either. | Unknown arguments, conflicting filters, and invalid limits rejected before API access; consistent usage exit code. |
| G7 | Medium / contract gap | Song `--key`/`--meter` map to query fields absent from the consulted song reference; attachment POST paths at song/key scope are inferred by reusing list paths; abbreviated `V1,V2,C` examples are passed through although documented sequence examples use full labels. | Capability matrix against a chosen API version; supported fields/paths/value encodings demonstrated; remove, translate, or clearly gate unsupported promises. Do not assume web UI abilities imply API abilities. |
| G8 | Medium / source + contract gap | Client does not pin `X-PCO-API-Version`; transport relies on account/application defaults. DTOs omit arrangement notes although mutation flags permit writing them. | Select/document supported Services version; test header; make writable user metadata inspectable or explicitly document limitations. |
| G9 | Medium / source | Upload buffers the whole file; upload and attachment POST are separate side effects with no structured partial-success recovery. Retry waits ignore cancellation until sleep finishes. Root execution has no explicit signal-to-context wiring. | Bounded/streamed upload; clear upload-success/attach-failure result and recovery using upload ID where safe; cancellation-aware waits; explicit retry policy without unsafe write replay. |
| G10 | Design | Command grammar and context change by resource. `search` duplicates list filtering. `show` selects an arrangement before the user chose one. | One documented grammar and shared targeting rules, with a deliberate convenience policy. |
| G11 | Design + source | Bare list JSON arrays have no pagination metadata; detail JSON differs by command. Human hidden suppression uses JSON `omitempty` for false state as well. | Document output schemas, empty results, list completeness, detail composition, stderr and exit codes; don't accidentally equate terse human output with omitted machine state. |
| G12 | Design | No committed end-state contract, capability audit, behavioral contract suite, or beginner usability validation for the proposed direction. Owner confirmed no compatibility requirement. | Deliberate command/output contracts and executable acceptance matrix before broad feature expansion; no legacy migration work. |

Local reproduction assumptions: fake server returned normal JSON:API shapes, correctly advertised a next page, and accepted received writes so request choice could be observed. This proves client decision/payload behavior, not Planning Center's validation of those writes. Original harness was temporary under `/tmp`; the brief records the requests so it does not depend on that file surviving.

Verification at the reviewed SHA: Go reports `go1.26.4`; `GOCACHE=/tmp/pccli-audit-cache go test ./...` and `go vet ./...` pass. An initially reused build cache produced a spurious missing-standard-library error; a fresh cache resolved it. Focused repros covered human and JSON paths for G1/G2/G4/G6 and human output for G3. No live organization requests or real credentials used. Passing tests do not prove remote contract compatibility.

## Initial proposal to challenge

### Vision

Teach one model: choose a Planning Center product, choose a resource, choose an action, supply named parent context. A beginner should discover the next action from help; an operator should be able to use IDs and structured output without hidden inference. Keep the song detail view as the primary orientation surface.

### Candidate command grammar

```sh
pccli services songs list --search "Grace"
pccli services songs show "Amazing Grace"
pccli services songs update 1001 --author "John Newton"
pccli services songs update 1001 --visibility hidden --hidden=false
pccli services arrangements list --song 1001
pccli services arrangements show 2001 --song 1001
pccli services arrangements create --song 1001 --name "Full Band"
pccli services keys list --song 1001 --arrangement 2001
pccli services keys create --song 1001 --arrangement 2001 --name "Lead" --start G
pccli services attachments list --song 1001 --arrangement 2001 --key 3001
pccli services attachments add --song 1001 --arrangement 2001 --file chart.pdf
```

These commands are proposals, not available commands. Parent flags may accept exact names as well as IDs. Attachment creation examples are conditional on G7 capability verification; do not expand inferred API support. `--key` in keys/attachments context means a key record; distinguish it from a musical pitch flag. Architect must settle naming and whether any default-parent convenience is justified.

Why prefer sibling resources: product/resource/action stays shallow, child resources get consistent list/show/create/update/delete verbs where supported, parent relationships become explicit flags, and positional arguments describe the target rather than a growing chain of parents. This costs repeated flags for key operations; alternatives must be compared on actual operator journeys, not token count alone.

Candidate selection contract:

- `--visibility visible|hidden|all` selects candidate resources consistently and defaults to visible. `--hidden=true|false` only mutates song state. Consider an equally clear simpler alternative if justified.
- Reads: exact name first, then a unique partial match. Writes: exact name or ID only. Ambiguity is an error after complete candidate lookup. No HTTP write before resolution and validation finish.
- Positive numeric IDs remain convenient. Decide an explicit escape for numeric names (`name:1234`) and ID disambiguation; avoid inventing an elaborate selector language.
- Song show lists arrangement context without choosing one arbitrarily. An explicit arrangement may request expanded keys; if exactly one exists, a clearly identified read-only expansion might be acceptable. Writes do not inherit a first-result default.
- List `--limit` means total returned items, not API page size; `--all` requests all results and conflicts with an explicit limit. Resolution enumerates independently, fails closed on errors, and stops once ambiguity is certain.
- Validation distinguishes flag presence from value, especially clears and false booleans. No input prompts required for automation. Keep `--yes` for deletion.

### Candidate internal architecture

Keep the existing package count unless a new boundary earns its cost:

```text
CLI parsing/presentation
  -> Services workflow operations and resolution policy (plain Go inputs, context)
       -> typed Services API adapter (paths, wire DTOs, capabilities, mapping)
            -> Planning Center transport (auth, request, decode, retry, upload)
```

- `internal/cli`: separate resource command files and product presenters; wire dependencies explicitly. No `*cobra.Command` in workflow inputs. Do not move product-specific presentation into a universal renderer framework.
- `internal/services`: product models, patch/create inputs, detail composition, parent resolution and ambiguity/visibility policy. Resource-specific adapter files may live in the same package initially. Introduce only small consumer-owned interfaces needed for tests and a real boundary, not an interface per struct or a giant mirrored CRUD interface.
- `internal/planningcenter`: one JSON:API envelope definition retaining pagination; reusable HTTP mechanics. Select a clear home for pagination traversal; adapter owns API-shaped queries, workflow owns completeness requirements.
- `internal/output`: terminal/JSON primitives; domain view composition stays near the product presenter. Stable machine output is designed separately from human suppression of false/default values.
- `internal/config`: one explicit precedence policy; preserve existing dotenv conventions. Remove redundant custom wrappers only where their semantics are unnecessary; do not churn dependencies to make an architecture diagram cleaner.
- Patch fields can use pointers (including pointer-to-slice), with explicit validation for permitted empty values. Null/clear semantics require per-field API evidence; do not add a generic optional-field framework unless truly needed.
- Use typed operational errors for not-found, ambiguous, invalid-input, incomplete-lookup, and partial attachment failure. CLI translates these to concise guidance and a small documented exit-code contract. Do not implement an error taxonomy more elaborate than callers need.

### Alternatives and tradeoffs

| Alternative | Appeal | Why not the initial preference |
| --- | --- | --- |
| Only split the two large files | Smallest diff | Leaves completeness, selection and mutation semantics in the wrong places. |
| Keep nested `songs arrangements` with explicit verbs | Parent relationship evident; fewer top-level resource groups | Preserves deeper paths and encourages parent positional chains; could still win if user journeys justify it. |
| Flatten everything under songs with many verbs | Very shallow | Noun/action inventory grows irregularly; resource discovery suffers. |
| Build a generic endpoint/CRUD engine | Less repetitive plumbing | Product policy becomes metadata and hooks; poor fit for user journeys and small current scope. |
| Full clean-architecture repository/service/entity stack | Formal separation | Too many layers for this project; interfaces and mapping become their own maintenance burden. |
| Full interactive TUI | Friendly exploration | Adds state and automation complexity before correctness and contracts are sound. Optional future enhancement only. |

### Proposed sequence and success criteria

1. Characterize and fix G1–G6 before reshaping commands. Assert that incomplete/ambiguous lookups emit no write. Specify the intended JSON/human contracts without preserving defective or inconsistent old behavior.
2. Resolve G7/G8 capability questions with official references and isolated permitted verification if available. Unknown write capabilities stay unsupported until established; no live write verification is authorized by this design task.
3. Extract workflow policies and consolidate envelopes. File moves and semantic corrections should remain reviewable even though public commands may change freely.
4. Implement the chosen command and output grammar without compatibility aliases; update help, README, workflow guides, command reference, VHS tapes and demo fixtures together.
5. Stabilize output, cancellation and attachment recovery; run focused request-aware tests plus `go test ./...`, `go vet ./...`, build, and appropriate docs/demo checks. Do not claim unsupported remote capabilities as validated.

Acceptance journeys: find a partial title; inspect song/arrangements/keys; disambiguate duplicate titles; edit and clear metadata; hide/unhide by ID and name; choose a key-specific file destination; recover after upload succeeds but attach fails; consume complete lists and errors in scripts; discover the next action from help without reading implementation docs.

Non-goals: new Planning Center products, comprehensive endpoint exposure, bulk writes, new authentication flows, plugins, persistent local databases, cross-command sessions, and a TUI. Do not silently convert missing features into this refactor's scope.

## External references and evidence boundaries

Consulted 2026-11-02. Links are source evidence, not a claim that every implementation field is supported. The versioned reference consulted is `2018-11-01`; the CLI currently does not select a version explicitly.

| Source | Relevant observation / design implication |
| --- | --- |
| [Planning Center Song API](https://api.planningcenteronline.com/docs/apps/services/versions/2018-11-01/vertices/song) | Pagination is explicit (1–100 records per page); documented queries include title, author, CCLI, hidden and themes. Key/meter flags need separate evidence. |
| [Arrangement API](https://api.planningcenteronline.com/docs/apps/services/versions/2018-11-01/vertices/arrangement) | Parent song, keys, writable notes, and sequence array are distinct concepts; documented sequence examples use full labels. |
| [Attachment API](https://api.planningcenteronline.com/docs/apps/services/versions/2018-11-01/vertices/attachment) | Consulted creating section specifies arrangement attachment POST. Reading associations do not establish equivalent write paths. |
| [File uploads](https://api.planningcenteronline.com/docs/overview/file-uploads) | Upload yields an expiring file UUID, used by a later API operation; a natural partial-success boundary. |
| [API versioning](https://api.planningcenteronline.com/docs/overview/versioning) | An explicit request header overrides application defaults; select a supported contract rather than inheriting unknown defaults. |
| [Songs page overview](https://help.planningcenter.com/en/139427-songs-page-overview.html) | Songs contain arrangement versions and keys; attachment placement affects which materials appear in plans. Parent choice is meaningful product context. |
| [GitHub issue list](https://cli.github.com/manual/gh_issue_list) | Resource/action grammar, search/filter flags, result limit and JSON support; useful precedent, not an instruction to copy every gh behavior. |
| [doctl droplet list](https://docs.digitalocean.com/reference/doctl/reference/compute/droplet/list/) | Product/resource/action grouping and explicit machine output offer another established reference. CLI conventions vary; consistency and workflow fit matter more than claiming a universal standard. |
| [Community discussion: multiple arrangements](https://www.reddit.com/r/worshipleaders/comments/tb4j4z/) | Search surfaced an operator question about arrangement variants. Direct fetch was unavailable in this turn; use only as a research lead, not API evidence or validated user research. |

Remaining research: verify real operator expectations for defaults and required parent flags; complete capability checks. Adoption/compatibility uncertainty was resolved by the owner: there is no adoption and old syntax can be removed. Neither an architecture opinion nor a mock server replaces the remaining checks.
