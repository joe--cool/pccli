# Recommended architecture and implementation direction

Status: recommended design, independently reviewed by GPT-6 Astra; application implementation has not started.

Baseline: `d8c596895cdff92b54ce91d89ae61335e5125eb8`. See the [current-state brief](brief.md), [review assignment](architect-prompt.md), and [full Astra report](astra-review.md) for evidence and detailed contracts.

The owner confirmed that there is no adoption and no backward-compatibility requirement. Replace inconsistent commands and output contracts directly. Do not build aliases, legacy serializers, compatibility wrappers, or a deprecation program.

## Product vision

Make pccli a reliable Planning Center workbench for people who maintain a song library and sometimes script that work. Users should learn one model: **product → resource → action**, with named parent context and visible IDs. They should be able to find a song, inspect its versions and keys, change precisely the fields they intend, and place material in an explicitly supported destination.

Simplicity means predictable decisions and understandable outcomes, not the fewest characters in a command. An extra parent flag is worthwhile when it prevents a file or metadata change from going to an inferred arrangement.

Do not add products, plans, bulk operations, a generic endpoint browser, a TUI, saved selection sessions, or a local database as part of this work.

## Current state

The foundations are good: Go/Cobra, a single binary, product-first organization, concise terminal output, `--json`, Basic Auth, explicit deletion confirmation, and documentation/demo/release tooling.

The simplification is incomplete in two ways:

- The user grammar varies. Songs have explicit actions; arrangement and key commands combine implicit list/show behavior with action children; attachment creation uses a separate verb. Parent context alternates between positional arguments and flags. `--hidden` has several meanings.
- Workflow policy lives in the 1,476-line Cobra song file. Services mostly maps endpoint calls and discards pagination before identity resolution. There are two competing JSON:API envelope definitions. Explicit empty patch values disappear.

These are related problems: a command that looks convenient can hide an unsafe target choice because the internal boundary does not preserve enough evidence to make the choice correctly.

## Chosen user interface

```text
pccli services songs        list | show | create | update | delete
pccli services arrangements list | show | create | update | delete
pccli services keys         list | show | create | update | delete
pccli services attachments  list | add
```

These are proposed commands, not the interface currently implemented. Representative journeys:

```sh
pccli services songs list --search "Grace"
pccli services songs show "Amazing Grace"
pccli services arrangements list --song 1001
pccli services arrangements show 2001 --song 1001
pccli services keys list --song 1001 --arrangement 2001
pccli services keys show 3001 --song 1001 --arrangement 2001

pccli services songs update 1001 --author "John Newton"
pccli services songs update 1001 --hidden=true
pccli services songs update 1001 --visibility hidden --hidden=false
pccli services arrangements update 2001 --song 1001 --notes "Verse, chorus, repeat"
pccli services attachments list --song 1001 --arrangement 2001 --key 3001
pccli services attachments add --song 1001 --arrangement 2001 --file chart.pdf
```

Arrangements always require `--song`; keys always require both `--song` and `--arrangement`. Show/update/delete have one positional target. Attachment listing selects a specific song/arrangement/key association. Attachment addition initially requires an arrangement; key- and song-scoped writes are not promised without contract evidence.

Song show remains the main orientation view: metadata and an arrangement overview. It expands a particular arrangement and its keys only when requested using `--arrangement`. There is no implicit first/default arrangement. Empty overview sections succeed; a missing explicitly requested target fails.

### Shared behavior rules

| Concern | Contract |
| --- | --- |
| Search | One entry point: `songs list --search TEXT`, with literal contains semantics. `--title` means literal case-insensitive exact title and conflicts with `--search`. API wildcard syntax stays internal. Remove the separate search command. |
| IDs and names | Positive decimal IDs remain strings. Support `id:1234` and `name:1234` to remove numeric-name ambiguity. Invalid signed/zero IDs fail locally; oversized digits never become a name through integer overflow. |
| Reads | Prefer one exact case-insensitive name; allow a unique literal partial match only after exact absence is established. Explicit `name:` requests exact matching. |
| Writes and parents | Exact names or IDs only. Resolve every parent in scope. No guessed targets, hidden bypass for IDs, implicit upsert, or global fallback for a child in the wrong parent. |
| Completeness | Display limits never cap identity resolution. A failed or malformed page means incomplete lookup and zero writes/uploads. Distinct matching IDs are ambiguous even if names are identical. |
| Visibility | `--visibility visible|hidden|all` selects existing songs and defaults to visible. `--hidden=true|false` changes state on song create/update only. |
| Defaults | No first-result or name-based default arrangement selection. Parent context is explicit even if there is currently only one arrangement. |
| Musical pitch | Arrangement mutation uses `--chart-key`; key records use `--start`/`--end` for pitch. Attachment `--key` selects a key record. |
| Lists | `--limit N` is a total-item limit, default 25. `--all` exhausts the selected collection and conflicts with an explicit limit. API page size is internal. |
| Patches | Omitted means unchanged. Explicit empty/zero/false/empty array is transmitted if valid for that field, otherwise rejected clearly. Never drop the value and claim success. |
| Deletes | Always require `--yes`; confirmation cannot bypass target or capability validation. |
| Errors | Usage/value errors exit 2; operational/partial/unknown outcomes exit 1. Signal exits are 130/143. No prompts required. |

### One machine contract

Use `--json` for one versioned envelope, with data and explicit completeness metadata. Do not retain a second legacy JSON mode. The full report specifies detail, mutation, recovery, and error schemas. A list has this general shape:

```json
{
  "schema": "pccli/v1",
  "data": [],
  "meta": {
    "returned": 0,
    "has_more": false,
    "complete": true
  }
}
```

IDs are strings; known false, zero and empty values remain explicit. Human suppression of default noise does not change machine fields. Requested components and unrequested expansions are distinguishable. A remote total, when available, is an observation, not a snapshot guarantee.

JSON errors go to stderr in a structured error envelope with a stable code and actionable context; stdout contains no success data on failure before publication. Human errors remain concise. The same formatter handles Cobra parsing errors and service failures. Test JSON selection even when usage parsing fails, plus stdout/stderr routing and empty results.

Do not stream success-looking partial exports while still fetching pages. The report proposes private temporary spooling for large JSON results; implement this once in the output path, with cleanup and cancellation. A final pipe failure can still leave partial bytes and must exit nonzero. The API callback/walker is a small internal tool, not a generic export framework.

## Chosen internal setup and why

```text
Cobra command parsing and product presentation
    -> Services workflows and product models
        -> typed Services API adapter
            -> Planning Center HTTP transport and shared JSON:API envelopes
```

| Owner | Responsibilities | Reason |
| --- | --- | --- |
| `internal/cli` | Parse flags/targets; translate usage errors; lazily construct dependencies; map product results into terminal/JSON output. | CLI grammar and presentation can change without changing identity or mutation policy. |
| `internal/services.Library` | Plain Go workflow inputs, selection, visibility, scoped parents, complete resolution, detail composition, mutation validation and outcomes. | Every follow-on operation inherits the same rules, independent of Cobra. |
| Concrete typed Services adapter in `internal/services` | API paths, queries, wire attributes, model mapping, selected version/capabilities, validated page traversal and response shapes. | Keep endpoint mechanics away from workflows without adding another package or a giant mirrored CRUD interface. |
| `internal/planningcenter` | HTTP, Basic Auth, origin/redirect checks, one set of JSON:API envelopes, request errors, bounded GET retries and streaming upload. | Shared transport mechanics have one implementation; product semantics stay out. |
| `internal/output` | Writers, tables, styles, JSON encoding/publication primitives. | Presentation mechanics are reusable; product detail composition stays near its presenter. |
| `internal/config` and entrypoint | Documented environment/dotenv precedence; credential-free help/version; signal context and exit. | Preserve the working setup and make dependencies/test isolation explicit. |

Keep the existing package set and libraries. Split files by actual responsibility rather than enforcing a file-count target. The existing low-level HTTP interface is useful behind the adapter; it is not the workflow API. Initially Library can depend on the concrete typed adapter, tested with a request-aware HTTP server or round tripper. Introduce a narrower interface only when a real consumer needs it.

Keep the documented `PCCLI_*` environment names and remove undocumented `PCO_*`/`PLANNING_CENTER_*` compatibility branches. This simplifies configuration without changing the documented dotenv precedence or reading real credential files.

Create and patch inputs are separate. Pointers preserve patch presence, including pointer-to-empty-slice for `[]`. Do not invent a general optional/null framework or raw JSON patch language. Read models include fields needed to inspect requested writes, such as arrangement notes and full sequences.

The pager preserves completeness and validates cursors; the resolver decides what establishes identity; lists decide bounds and lookahead. One implementation must not silently substitute its display policy for another's identity requirements. Full traversal is not an atomic remote snapshot: recheck selected state before writing, prefer IDs for scripts, and report detected concurrent changes rather than silently retargeting.

### Why this over the main alternatives

- **Sibling resources over deeper nesting:** named parents preserve the domain relationship while one resource/action grammar improves discovery. The cost is longer key commands; the benefit is fewer positional chains and fewer exceptions.
- **Explicit parents over convenient defaults:** adding another arrangement cannot silently change an existing command's target. The composed song overview supplies the convenient read surface.
- **One list/search workflow:** a separate title-search command adds grammar and defaults without adding a distinct capability.
- **One JSON envelope over bare arrays or parallel modes:** with no compatibility constraint, completeness, recovery and errors can be designed once and taught once.
- **Workflow/API split inside existing packages over a framework rewrite:** it repairs the actual boundary errors without repositories, command buses, interface-per-type scaffolding, or endpoint generation.
- **Fix correctness before cosmetic restructuring:** moving files or renaming commands cannot repair incomplete candidate sets or lost patch values.

## Gaps and conditions before release

The brief records evidence and initial gaps G1–G12. Astra expands them with G13–G15 and corrects the assessment of the existing tests: some already check payloads and queries; keep those useful tests. The gaps below summarize release significance, not a claim that all remote failures have been observed.

| Priority | Gaps | Required outcome |
| --- | --- | --- |
| P0: safe selection | G1 pagination; G4 inferred parents; G13 response/cursor/scope validation. | No selection from incomplete data; no arbitrary parent; no write/upload on ambiguity, page failure or malformed response; credentials cannot follow hostile pagination links. |
| P1: correct mutations | G2 patch presence; G3 visibility; G6 input validation; G8 version/read fields. | Consistent name/ID behavior; valid requested values reach payloads; rejected values fail early; selected API version is explicit; requested changes are inspectable. |
| P1: honest capabilities | G7 paths, filters, sequence encoding and clear semantics. | Check in a versioned capability inventory; remove unsupported promises; verify each uncertain field independently. Unknown key attachment creation must not delay a documented arrangement workflow. |
| P1: reliable operation and release | G5 behavioral coverage; G9 cancellation/upload/recovery; G11 output contract; G12 docs/contract consistency. | Request-aware tests, prompt cancellation, bounded upload memory, no automatic write replay, preserved upload receipts, honest outcomes, and executable help/docs/VHS using only the chosen interface. |
| P2: coherent product | G10 grammar/help-driven usability; G14 archive behavior; G15 terminal/error handling. | One discoverable grammar; detected archived writes blocked until intentionally supported; human strings cannot inject terminal controls. |

The consulted [Services reference](https://api.planningcenteronline.com/docs/apps/services/versions/2018-11-01/vertices/attachment) establishes arrangement attachment creation, not equivalent writes at every read association. Pin the supported Services contract (`2018-11-01` in the current reference) and do not silently retry without it. Version selection is separate from the upload service.

Still requiring evidence: literal/wildcard/non-ASCII matching semantics needed for candidate-query completeness; all-state list behavior; exact clear/readback behavior per field; archived arrangements; UUID recovery/readback/reuse details; and any additional attachment write scope. Use conservative complete enumeration or disable only the uncertain capability. Do not make live account mutations to settle these questions under this design assignment.

## Implementation sequence

Each row is a reviewable step, split further by resource if needed. There is no legacy migration phase. Internal changes may be staged before the final command rewrite to make reviews understandable, not to preserve old behavior.

| Step | Work and exit condition |
| --- | --- |
| 1. Establish safe identity | Add request-aware regressions for a duplicate on page two, a failed second page, duplicate defaults, wrong child scope, malformed data and hostile cursors. Preserve page evidence through one envelope and repair all resolvers. Assert zero side effects on unsafe paths. This step can start immediately. |
| 2. Repair write inputs | Separate create/patch, preserve explicit values, validate selectors/arguments/ranges, separate visibility from state, expose inspectable notes/full sequence. Verify hide/unhide by name and ID and supported clears. |
| 3. Fix the API contract | Pin Services version and record capability evidence. Separate attachment read scope from supported add scope. Remove unsupported flags/paths/examples before any upload can occur. |
| 4. Extract ownership | Move shared workflows and detail composition out of Cobra, introduce the concrete typed adapter, consolidate envelopes, split presenters/resource commands and inject construction/writers. Stop adding layers once responsibilities are clear. |
| 5. Replace the public grammar | Implement sibling resources, explicit parents and consistent actions. Remove replaced commands. Update help, README, setup-first quick start, workflow guides, command reference, VHS and demo fixtures together. |
| 6. Finish output and cancellation | Implement total-item lists, `--all`, the single JSON envelope, child completeness, structured errors, signal context and cancellable GET retry waits. Test late failures, output cleanup and parser errors in JSON mode. |
| 7. Make attachments recoverable | Stream multipart data; retain upload ID/expiry/destination/stage; distinguish rejection from unknown commitment; give inspection/retry guidance without replaying writes. Inject copy, cancellation, rejection, timeout and output failures. |
| 8. Validate and publish the new contract | Run full tests, vet, build, docs/examples and mock demo checks. Exercise key journeys using help alone. Release notes distinguish supported new contracts, corrected bugs and deferred capabilities. No claim of live validation without evidence. |

Detailed Go sketches, the full behavior matrix, failure cases, retry/recovery policy and per-step stop conditions are in [Astra's report](astra-review.md). No further architecture round is required to start step 1. Feature expansion waits until these contracts are reliable.

## Review outcome and validation limits

Astra independently inspected the implementation and official references, confirmed the main source mechanisms, and supplied additional findings and exact contracts. Its first draft conservatively retained compatibility; the owner then explicitly removed that constraint, and the final direction uses one new interface. The [saved prompt](architect-prompt.md) includes that clarification for future reuse.

Baseline `go test ./...` and `go vet ./...` passed with a fresh temporary Go cache. The earlier audit's synthetic local API tests establish client decisions and emitted payloads, not the behavior of a live Planning Center account. This deliverable changes contributor documents only; the proposed runtime architecture is not yet implemented or production-validated.
