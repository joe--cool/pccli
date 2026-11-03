# pccli: independent architecture decision

Author: Astra architect, GPT-6 Astra (`gpt-6-astra`).

Reviewed: 2026-11-02, America/Los_Angeles.

Source baseline: `d8c596895cdff92b54ce91d89ae61335e5125eb8`.

Scope: independent review of the brief, repository instructions, command implementation, Services adapter, transport, configuration, output, tests, and public command/workflow documentation. This report is a proposed implementation contract. It does not describe changes already made to the executable. I did not read credential files, contact an organization API, perform mutations, or change application code. The brief's synthetic reproductions are attributed evidence; I confirmed their mechanisms in source, without rerunning those reproductions or the test suite. A concurrent documentation link change does not affect the reviewed application baseline.

**Controlling user clarification, received during this review:** “Nothing needs to be backward compatible. There isn’t any adoption yet. I assume good interfaces will foster good adoption.” This supersedes the brief's unknown-adoption constraint. The design below replaces inconsistent commands and JSON shapes directly, with no legacy aliases, compatibility adapters, deprecation period, or release-history gate. The staged implementation is for correctness and reviewability, not preservation of the old interface.

## 1. Decision and product direction

Build **a reliable Services song-library workbench for an operator who also scripts**. Optimize finding the right song, understanding its versions and keys, making a precise metadata change, and putting rehearsal material in a supported destination. Correct identity and an honest account of what completed are the defining product qualities.

Keep the current product-first shape and adopt sibling resource groups:

```text
pccli services songs        list | show | create | update | delete
pccli services arrangements list | show | create | update | delete
pccli services keys         list | show | create | update | delete
pccli services attachments  list | add
```

Parent relationships remain explicit through `--song` and `--arrangement`. Keep one target positional argument on show/update/delete. `attachments list` can select a key record; `attachments add` initially supports arrangements only. Do not infer write paths from read associations.

Keep the existing package set. Turn `services.Library` into the workflow boundary and put a concrete, typed Services API adapter beside it. Keep the generic HTTP interface behind that adapter. Retain Cobra, the existing configuration libraries, terminal renderer, build tooling, and releases. This work does not require a dependency upgrade.

The highest priority is correcting selection and mutation semantics. Renaming commands is a later usability change, not the mechanism that fixes those problems. An implementation agent can begin with the first PR in section 11 without another design round.

### Journeys to optimize

| Journey | Product behavior |
| --- | --- |
| Find and orient | Search titles, see IDs and authors, show a song with its arrangements, explicitly inspect one arrangement and its keys. |
| Maintain metadata | Identify one resource, change only supplied fields, clear supported metadata, hide or unhide with separate selection and state flags. |
| Prepare materials | Choose an arrangement, upload or attach a file/link through documented API capabilities, understand and recover from partial completion. |
| Script the same tasks | Use IDs, bounded or complete lists, consistent exit status, stable JSON, no prompts, and cancellable requests. |

Do not add plans, other Planning Center products, bulk changes, a raw endpoint browser, persistent local catalog, saved selection sessions, plugins, an interactive picker, or a TUI in this refactor. Do not add attachment show/update/delete merely to fill a CRUD grid.

This direction follows the actual product relationship: songs have versions, versions have keys, and material placement affects what appears in a plan. Planning Center's [song overview](https://help.planningcenter.com/en/139427-songs-page-overview.html) and [arrangement guide](https://help.planningcenter.com/en/139430-add-or-edit-an-arrangement.html) support that model. A directly inspected [worship-leader discussion](https://www.reddit.com/r/worshipleaders/comments/yotmsu/how_do_you_provide_arrangement_details_to/) describes coordinating charts, reference recordings, and arrangement notes before rehearsal. That is useful qualitative evidence for keeping context visible; it does not establish API behavior or represent formal user testing.

The useful precedents are resource/action organization, explicit context, limits, filters, and machine output. [GitHub's issue list](https://cli.github.com/manual/gh_issue_list) combines list, search flags, a result limit, and repository context. [doctl's droplet list](https://docs.digitalocean.com/reference/doctl/reference/compute/droplet/list/) demonstrates product/resource/action and selectable JSON. Neither establishes a universal command naming standard. Retain `show` and `update`; renaming them to imitate another tool earns nothing here.

## 2. Findings and corrections to the brief

The project has a sound small-CLI foundation. The current detail view is more useful than a raw API response, default output is already separated from the banner, IDs are exposed, help loads without credentials, writes can run unattended, deletes require confirmation, and Basic Auth is correct. There is no evidence that it needs a framework rewrite.

The responsibility problem is concrete: `internal/cli/songs.go` chooses identities and parents while `internal/services/library.go` discards the evidence needed to know whether those choices are safe. Splitting files alone cannot repair that contract.

| Classification | Finding and source | Consequence |
| --- | --- | --- |
| Confirmed source defect; reproduction supplied in brief | `resolveSongID`, `resolveArrangement`, and `resolveKey` operate on capped pages. `services.collection` at line 442 discards links and metadata. | A unique match on an incomplete result is treated as unique overall. This is the release-blocking defect for name-based writes. |
| Confirmed source defect; reproduction supplied | `setString`, `setInt`, `setFloat`, and `setStringSlice` at lines 616–644 omit explicitly supplied empty/zero values. | Some updates send an empty attribute object and can report success without attempting the requested change. The existing pointer-to-bool helper already preserves false correctly. |
| Confirmed source defect; reproduction supplied | `newSongsUpdateCommand` passes whether `--hidden` changed into target resolution. Name lookup then filters hidden-only; numeric lookup bypasses the same test. | Selection changes with the mutation value's presence. This is not a mere naming preference. |
| Confirmed unsafe policy; reproduction supplied | `defaultArrangement` at line 1184 picks the first `Default`, then the first result. | Multiple possible parents can cause a write to the wrong arrangement. A display name is not an authoritative default marker. |
| Confirmed additional defect | `songs show` only resolves the explicit arrangement inside `if len(arrangements) > 0`. | An explicitly requested arrangement can be ignored when the song has none. The correct result is an error. |
| Confirmed additional validation gap | `isNumericID` accepts signed integers and zero; overflow becomes a name. `resolveKey` fabricates an ID-only key without fetching it. | Input interpretation is unstable, and key existence/scope is not checked before subsequent operations. The scoped upstream endpoint may still reject a wrong parent; a demonstrated cross-parent mutation is not established. |
| Confirmed additional protocol gap | JSON decoding accepts `{}`, missing `data`, empty successful bodies with a non-nil destination, and resources lacking identity as zero-value successes. | A malformed response can be mistaken for an empty collection or successful mutation. Valid JSON alone is insufficient validation. |
| Test limitation, with a correction to emphasis | The demo transport ignores query/body and returns 200 for matched routes. Services unit tests do inspect several queries and payloads already. | The suite is not entirely a blind fixture check. It lacks multi-page, request-sequence, scope, malformed-response, and no-write assertions. Keep useful existing tests. |
| Contract unknown, not a proved remote failure | Song key/meter filters, attachment song/key POST paths, and abbreviated sequence writes lack the required contract evidence. | Stop promising unsupported operations. Do not describe them as known Planning Center failures. |
| Additional contract unknown | Attachment `where[filename_like]`, `where[type]`, and `song_part` are also absent from the consulted attachment reference's relevant query/assignable lists. | Expand G7 to cover these promises, not only attachment paths. `themes` is text metadata; help should not call it a substitute for relational tags. |
| Design choice | Deeper resource paths, duplicate `search` entry point, bare JSON arrays, concrete dependencies, and different wire/product types. | None is intrinsically a defect. Change only the parts with an operator or correctness benefit. |

The CCLI-only create behavior is not an accidental fabricated title: the source intentionally places the CCLI number in the title field, matching the [documented Song creation behavior](https://api.planningcenteronline.com/docs/apps/services/versions/2018-11-01/vertices/song). Keep that workflow and explain it accurately.

Meaningful differences from the initial proposal:

1. **Use one JSON envelope under `--json`.** Completeness metadata and structured errors are part of the contract from the beginning. There is no separate legacy array mode or extra envelope flag.
2. **Require parents consistently in the canonical grammar, even for a sole arrangement.** The read-only song overview is the convenience surface. Parent omission should not mean different things on successive commands.
3. **Replace the old surface directly.** The user's no-adoption clarification removes the reason for adapters, duplicate search commands, and migration gates.
4. **Restrict attachment creation now to the documented arrangement scope.** A key-specific destination is a legitimate user need but remains deferred, not a promised part of this architecture.
5. **Use a concrete typed API adapter initially.** A second giant CRUD interface would mostly mirror it. Request-aware tests can exercise the workflow through an HTTP test transport.

## 3. Canonical command contract

The following forms are the end-state contract for the implementation sequence, not commands currently available in this checkout. All data commands accept `--json`, with one envelope contract specified below.

### Resource actions and parent inputs

| Command | Required inputs and meaning |
| --- | --- |
| `services songs list` | No positionals. Optional filters, visibility, and list controls. |
| `services songs show SONG [--arrangement ARRANGEMENT]` | One song target. Show song metadata and a bounded arrangement overview. Expand an arrangement only when requested. |
| `services songs create` | No positionals. Nonblank `--title`, or positive `--ccli`. Both are allowed for an explicitly named song with a CCLI number. |
| `services songs update SONG` | One target and at least one changed metadata flag. |
| `services songs delete SONG --yes` | One target, explicit confirmation. |
| `services arrangements list --song SONG` | Explicit song; no positionals. |
| `services arrangements show ARRANGEMENT --song SONG` | Explicit target and song. Show arrangement metadata and bounded keys. |
| `services arrangements create --song SONG --name NAME` | Explicit song and nonblank name. |
| `services arrangements update ARRANGEMENT --song SONG` | Explicit target and song, at least one changed field. |
| `services arrangements delete ARRANGEMENT --song SONG --yes` | Explicit target and song, confirmation. |
| `services keys list --song SONG --arrangement ARRANGEMENT` | Both parents required. |
| `services keys show KEY --song SONG --arrangement ARRANGEMENT` | Both parents and one key-record target. This small read fills an actual inspect-before-edit gap. |
| `services keys create --song SONG --arrangement ARRANGEMENT --name NAME --start PITCH` | Both parents, nonblank record name, musical starting pitch. Omitted `--end` defaults to the starting pitch on create only. |
| `services keys update KEY --song SONG --arrangement ARRANGEMENT` | Both parents, target, and at least one changed field. Updating start alone does not silently rewrite end. |
| `services keys delete KEY --song SONG --arrangement ARRANGEMENT --yes` | Both parents, target, confirmation. |
| `services attachments list --song SONG [--arrangement ARRANGEMENT [--key KEY]]` | Select exactly the API association represented by the supplied scope. Key requires arrangement. No implicit union of song/arrangement/key materials. |
| `services attachments add --song SONG --arrangement ARRANGEMENT SOURCE` | Arrangement required. Exactly one source flag: `--file PATH`, `--upload-id UUID`, `--url URL`, or `--content TEXT`. No positional source. No `--key` creation support yet. |

Required parent flags accept IDs or names using the same rules as positional targets. A child ID never waives parent validation or the song's visibility policy. Resolve song, then arrangement within that song, then key within that arrangement. Never fall back to a global search when a child does not belong to the supplied parent.

There is no inferred default arrangement in canonical commands. Two arrangements named `Default` are simply duplicate names. The song overview shows both IDs; requesting `--arrangement Default` reports ambiguity. With no arrangements, the overview succeeds with an empty section; an explicit `--arrangement` fails. With one arrangement, the overview still does not silently fetch its keys. This keeps request scope and output stable as another arrangement is added later.

Overview child lists display at most 25 records and disclose truncation. `arrangements list` and `keys list` provide `--limit`/`--all` for larger collections. An explicitly requested arrangement resolves independently of those overview limits. Detail composition fails if a requested component fails; it does not print a half-successful song and then exit as though complete.

### Names, IDs, and search

Use one parser for every selector:

| Input | Interpretation |
| --- | --- |
| `1234` | Positive decimal ID. Keep IDs as strings; do not parse into a machine-sized integer. |
| `id:1234` | Explicit ID, validated as positive decimal digits. |
| `name:1234` | Literal numeric name, exact-match selection. |
| `name:Grace` | Explicit exact name, also useful when a partial read would be undesirable. |
| `Amazing Grace` | Exact name first; a unique partial match is permitted only for read operations. |
| `0`, `-4`, `+4`, `id:-4`, `id:abc` | Invalid selector, exit 2. A literal such name must use `name:`. |

An arbitrarily long nonzero decimal ID does not overflow locally or turn into a name; send it as an ID and let the API reject a nonexistent identifier. Reject an all-zero ID. Normalize leading zeros in IDs; preserve them in `name:`. `name:` strips one prefix only, so `name:name:1234` selects the literal name `name:1234`. Reject empty/whitespace-only selectors. Preserve meaningful name characters rather than applying transliteration or punctuation cleanup.

Exact names compare case-insensitively, with no accent removal or token reordering. Partial reads use literal case-insensitive substring matching. Writes, including every parent of a create/add operation, accept only IDs or exact names. A name never becomes an implicit upsert. Creating a resource is allowed to produce an additional record; subsequent selection must still disambiguate it.

Discovery is `songs list --search TEXT` (with `-q`). It means literal title-contains, not a full-text language. `--title` means a literal case-insensitive exact title; `--search` and `--title` conflict. Exact author, CCLI, and themes filters remain. Do not expose raw API wildcard syntax through ordinary flags. The adapter may use a server wildcard as an optimization only when it preserves the CLI's literal semantics, then locally filter results. Remove unsupported song key/meter and attachment filename/type filters from the command definitions; unknown flags fail clearly before network access.

Remove `songs search QUERY`; `list --search` is the only title-search entry point. Every list defaults to 25. List filters do not alter how a later show/update resolves its own target.

Keep `--key` only as the attachment list's **key-record selector**. Rename arrangement mutation pitch to `--chart-key`. Keep key creation/update `--start` and `--end` for musical pitches. Help examples must show `--key "Female Lead"` versus `--start Bb`; do not imply that an arrangement name or key record is its pitch.

### Visibility and confirmations

`--visibility visible|hidden|all` selects **songs**, defaults to `visible`, and is accepted on every operation that looks up an existing song, including all descendant commands. It does not alter the new state. `--hidden[=true|false]` belongs only to song create/update and changes state. Song create rejects selection-only visibility because it has no existing song to select.

```sh
# Hide an existing visible song, by name or ID.
pccli services songs update "Amazing Grace" --hidden=true
pccli services songs update 1001 --hidden=true

# Unhide an existing hidden song, by name or ID.
pccli services songs update "Amazing Grace" --visibility hidden --hidden=false
pccli services songs update 1001 --visibility hidden --hidden=false

# Include both states in identity selection; duplicates across them remain ambiguous.
pccli services songs show "Grace" --visibility all
```

ID lookups fetch and validate the song even under `all`. `hidden` rejects a visible ID just as `visible` rejects a hidden ID. Report a state mismatch and the appropriate flag; never silently broaden the selected set. A visible/hidden duplicate does not make the visible-only operation ambiguous, but it does make the `all` name operation ambiguous. Disambiguating output includes ID, title/name, author, CCLI when present, and visibility; arrangements include song ID and chart pitch, keys include arrangement ID and pitches. Cap displayed examples at 10 and report that more exist when known. Never claim a complete candidate count after stopping early.

Do not conflate song hiding with arrangement `archived_at`. The API exposes that separate field; its association/direct-read behavior needs a contract check. Map and display archive state when returned. Do not add archival mutations or promise that `--visibility all` includes archived arrangements. A write to an explicitly detected archived arrangement should fail with an unsupported-state explanation until an intentional archive-selection workflow exists.

Deletes require `--yes` in terminals and scripts. Omission is a usage error before any network request; do not prompt. Explain relevant descendant effects in delete help, only to the extent documented. Metadata changes and uploads need no additional prompt. `--yes` cannot bypass ambiguous names, hidden selection, invalid scope, or an unsupported capability.

## 4. Completeness, pagination, and identity

There are two different operations: a **bounded result for display** and a **proof sufficient to select an identity**. They must not share a limit.

### Lists

- `--limit N` means at most N matching records in the result, not API `per_page`. Require N >= 1. Default 25 on canonical list commands.
- `--all` follows the collection to exhaustion and conflicts with an explicitly supplied `--limit`, even `--limit 25`. It does not mean all visibility states.
- Use internal API pages of at most 100. Do not expose page-size tuning as an ordinary operator flag.
- Default songs to API `order=title` and arrangements to `order=name`. Keys and attachments use `order=created_at`. Preserve traversal order within equal sort values; do not claim an undocumented server-side tie-breaker. Song `--order` accepts the documented title/created_at/updated_at/last_scheduled_at fields with an optional leading minus. Do not expose ordering on every command without a workflow need. Stable total ordering across concurrent updates is not promised.
- Count after local filtering and resource-ID deduplication. Local filters may require additional API pages to fill N results.
- Read one matching record beyond N when necessary to distinguish a full result from a truncated one. A failed lookahead is a failed list, not a claimed complete result.
- Return metadata internally: returned count, whether more matching records were observed, and whether the result exhausted its selected collection. An upstream `total_count` is optional observational metadata; never fabricate it or use it as proof of uniqueness.
- Fetch failure before completion of the requested result returns nonzero without committing a successful JSON document. Default lists may succeed with `has_more=true`; `--all` may not succeed with incomplete traversal.

### Resolution

1. Validate selectors, fields, and parent requirements before loading credentials or making requests.
2. For an ID, fetch the scoped resource and validate its identity, parent relationship where supplied, and visibility/state. Return the fetched model to avoid redundant GETs.
3. For a song name, exhaust the documented exact-title query in the selected visibility set and locally check exact equality. The official [JSON:API filtering guide](https://api.planningcenteronline.com/docs/overview/json-api) documents case-insensitive equality and boundary wildcards. This supports an exact-query first pass rather than the current ten-result broad search.
4. For a bare read selector with no exact match, exhaust a title-contains candidate query and locally check literal substring matching. No read-only fallback is allowed after an error in the exact pass. Writes and `name:` stop at exact matching.
5. For arrangement/key names, walk the complete scoped collection because a server-side name filter is not established here. Apply exact-first matching locally; only consider partials after exact absence is established.
6. Return one candidate only after exhausting the relevant set. Stop as soon as two distinct exact matches prove ambiguity. Two partials alone do not justify an early ambiguous result if a later exact match can still win. If exact absence has already been established in a separate complete pass, two partials prove ambiguity.
7. Any failed page, invalid response, cursor cycle, scope-changing link, or cancellation makes lookup incomplete. Return a typed error and issue **no upload, POST, PATCH, or DELETE**.

`--visibility all` name resolution explicitly enumerates both visible and hidden filtered sets; uniqueness must cover their union. This avoids relying on undocumented default visibility of an unfiltered search. Deduplicate by resource ID across the sets. If one ID appears with conflicting identity/state snapshots, return a concurrent-change/incomplete-lookup error rather than choosing one. Stop early only when ambiguity is already proved.

For `songs list --visibility all`, the intended adapter contract is a single unfiltered collection containing both states. Verify that behavior before enabling it. If omission excludes a state, implement two filtered streams with an explicit merge in this workflow; do not equate omission with `all` silently. The safe interim behavior is an unsupported-capability error for that list form. This narrow contract gate does not block visible lists or explicit two-set resolution.

Literal `%`, `_`, non-ASCII case behavior, or an unsupported contains operator must not narrow a candidate query so much that matches are missed. The overview establishes general equality/wildcard behavior, but it does not prove Song-specific wildcard support, literal escaping, or equivalence between database collation and Go case folding. Those are contract gaps, not established defects. Use a broad, complete selected collection and local matching when the adapter cannot establish that its server-side predicate is a superset of the CLI match rule. Until those gaps are verified, use that fallback for literal wildcard characters and non-ASCII names/searches; keep exact ASCII equality as the documented optimization. Contains optimization requires its Song-specific contract check. This fallback trades extra requests for correctness; it is not an invitation to build a local index. Do not silently retry a denied query as a broader operation on an authentication/permission error.

The stress case with one exact Grace and nine partial matches on page one must fetch page two before choosing a result; the second exact Grace makes the write fail. A page-two failure also makes the write fail. A display limit of ten has no influence on either decision.

### Pagination trust and malformed data

Consolidate the JSON:API envelope in `planningcenter`; validate it at the Services adapter. A collection requires a present array `data`, including `[]` for empty. Missing, null, or object-shaped collection data is invalid. A single-resource response requires object data, expected type, nonempty valid ID, and the fields required for the operation. Preserve the difference between missing, null, and zero when it matters to visibility or mutation verification. Ignore unknown attributes for forward compatibility; do not use blanket `DisallowUnknownFields`.

The transport resolves a next URL against the configured API origin, rejects other schemes/origins/userinfo/fragments and cross-origin redirects, and never forwards credentials to a pagination-supplied origin. The Services pager additionally requires the same endpoint path, parent IDs, and immutable filter/order/version context. Accept only validated pagination changes, reconstruct the request from the original query plus the validated cursor, and detect repeated/non-progressing cursors. Test relative and absolute same-origin links. The configured upload origin is a separate explicit origin, not a permissible collection cursor destination.

The adapter should fail on contradictions it can detect: a nonempty next link with a nonsensical cursor, a page larger than its contract, wrong resource types, a single response for the wrong requested ID, or an advertised count that clearly exceeds an exhausted matching collection under unchanged counts. `total_count` is advisory under concurrent changes and local filters; absence of metadata is not itself invalid if collection/link semantics are valid. Never invent pagination by incrementing offsets after a missing next link merely because a guess seems likely.

Deduplicate repeated IDs with identical selection-critical fields; they do not represent two identities. Contradictory repeats fail resolution. Distinct IDs with the same name remain distinct. In lists, use the first consistent occurrence and disclose detected instability as an error rather than falsely claiming a complete export.

### What this cannot guarantee

Offset pagination over a changing remote collection is not a snapshot. A record can move between pages, a hidden song can change state between two passes, and a duplicate can be created after lookup. Full traversal eliminates the current deterministic truncation bug; it does not provide serializable selection.

Immediately before a write, re-fetch the selected resource and required parent state and ensure the same ID/name/scope still satisfies the request. If it changed, stop; do not re-resolve to another record automatically. A name recheck cannot prove that no new duplicate appeared. Use conditional requests only if Planning Center documents a supported precondition mechanism; do not invent ETag guarantees. Prefer IDs in scripts and describe name uniqueness as established over the completed observations, not atomically guaranteed at mutation time.

## 5. Mutation contract

Create and patch are separate input types. Cobra flag defaults are not patch values. For each update flag, `Flags().Changed` decides whether to populate a pointer; the field's contents decide validation, not presence.

| Field/input class | Create rule | Update rule |
| --- | --- | --- |
| Song title; arrangement/key record name | Required nonblank except CCLI-only song import. | Omitted means unchanged; empty or whitespace-only is invalid. |
| Free text: author, admin, copyright, themes, arrangement notes | Optional; explicit empty may be sent as empty. | `--author=""`, `--notes=""`, etc. send `""` and mean clear. They must not disappear from the payload. |
| CCLI | Positive integer when supplied. CCLI-only create places its decimal value into the API import field. | Positive when supplied; zero/negative are invalid. Do not pretend zero clears it without evidence of the API's clearing representation. |
| Length | Optional nonnegative integer seconds; `m:ss` accepted. | `--length 0` sends numeric zero; empty, negative, overflowing, or malformed input is invalid. |
| BPM | Optional finite positive number. | Positive finite value; reject NaN, infinities, zero, and negative. Tempo clearing is deferred until its supported representation is established. |
| Meter/chart pitch/start/end pitch | Validate documented value set for the field. | Empty is invalid until a documented clear representation exists. Do not conflate a key-record name with pitch. |
| Hidden / lyrics-enabled / item-details | Omitted lets create defaults apply; explicit false is sent. | Omitted is unchanged; `false` is sent exactly. `item-details` currently belongs to attachment addition, not a general attachment update. |
| Sequence | Optional array of full section labels. | Omitted is unchanged; `--sequence ""` deliberately sends `[]`. Reject mixed empty elements. Use full labels such as `"Verse 1,Chorus 1"`; do not pass undocumented abbreviations through. |
| JSON null | No general raw-JSON input. | No null-setting CLI syntax in this scope. Do not reinterpret empty text as null. Add a per-field clear operation later only if an API field requires null and its contract is established. |

A fully numeric literal title on create can trigger server-side CCLI import. Reject an all-digit `--title` with guidance to use explicit `--ccli` for import; `name:1234` is a target escape, not an unsupported promise to force literal numeric-title creation. This validation change needs release notes.

For updates, reject an input with no fields present before HTTP. Then serialize every present, valid field, including false/zero/empty. Keep the encoded attribute set nonempty. Do not fill update defaults such as end=start or copy the entire fetched resource into a patch.

The generic API reference establishes field assignability but does not exhaustively document every accepted empty value. Treat free-text empty, empty sequence, zero length, and false booleans as requested semantics to implement and verify, not as already tested live behavior. Validate their returned state. Decode the mutation response with attribute presence; if it lacks a touched field, GET the resource. If the API rejects a clear, surface that rejection. If it returns success but preserves the old nonempty value, return `verification_failed`, explain that a write was submitted, and do not report the requested clear as completed. Compare canonical sequence values, not `sequence_short` presentation. Map notes and full sequence into the read model so verification is possible.

Verification after a write is not a rollback transaction: other requested fields might already have changed, and another client might immediately overwrite them. Do not retry or restore automatically. Return the affected ID and an honest submitted/observed outcome. If the field was already empty and the explicit request is correctly serialized and confirmed, success is legitimate; the defect to prevent is dropping the requested operation and claiming it ran.

## 6. Go boundaries and dependency ownership

Use these file-level boundaries within the existing packages:

```text
cmd/pccli/main.go
  signal context, process exit
    -> internal/cli
       Cobra parsing, usage validation, lazy dependency construction,
       human presenters, JSON schema mapping
         -> internal/services.Library
            typed workflow inputs, field validation, target resolution,
            visibility, parent checks, detail composition, partial outcomes
              -> internal/services.API (concrete adapter)
                 typed endpoint operations, DTO mapping, version/capabilities,
                 scoped pagination traversal and response validation
                   -> internal/planningcenter.Client
                      HTTP, Basic Auth, origin/redirect protection, shared
                      JSON:API envelopes, HTTP errors, retries, streaming upload

internal/config: explicit configuration precedence and validation
internal/output: writers, styling, tables, JSON encoding/spooling primitives
```

Suggested files, without a new package for each noun:

```text
internal/cli/{songs,arrangements,keys,attachments}.go
internal/cli/{services_presenter,services_json,errors,root}.go
internal/services/{library,models,inputs,resolve,details,attachments,errors}.go
internal/services/{api,api_songs,api_arrangements,api_keys,api_attachments,paging}.go
internal/planningcenter/{client,jsonapi,upload}.go
```

Separate files by responsibility when extracted; the list is an ownership map, not a mandatory file-count target. Do not split a tiny helper merely to match the diagram.

### Small type sketches

These sketches establish ownership, not a complete code generator specification:

```go
// services: product and workflow inputs. No Cobra or HTTP types.
type Visibility uint8 // Visible, Hidden, All
type MatchMode uint8  // ExactOnly, ExactThenPartial
type Selector struct { ID string; Name string; Exact bool }
type Selection struct { Value Selector; Visibility Visibility }
type SongScope struct { Song Selector; Visibility Visibility }
type ArrangementScope struct { SongScope; Arrangement Selector }
type KeyScope struct { ArrangementScope; Key Selector }

type SongCreate struct {
    Title string
    CCLI *int64
    Author *string
    Hidden *bool
    // Remaining explicitly supported create fields.
}
type SongPatch struct {
    Title *string
    Author *string
    Hidden *bool
    // Remaining optional patch fields.
}
type ArrangementPatch struct {
    Name *string
    Notes *string
    Length *int64
    LyricsEnabled *bool
    Sequence *[]string
    // Other supported optional fields.
}

// services.API: typed API methods accept resolved IDs, never selectors.
type API struct { http jsonTransport }
type Library struct { api *API }
type Page[T any] struct {
    Items []T
    Next *pageCursor // validated adapter-private cursor
    APITotal *int64  // absent differs from zero
}
type ListPolicy struct { Limit int; All bool }
type ListInfo struct {
    Returned int64
    HasMore bool
    Complete bool
    APITotal *int64
}

func (a *API) SongPage(ctx context.Context, q songQuery, next *pageCursor) (Page[Song], error)
func (a *API) GetKey(ctx context.Context, songID, arrangementID, keyID string) (Key, error)
func (a *API) PatchSong(ctx context.Context, id string, p SongPatch) (Song, error)
func (l *Library) UpdateSong(ctx context.Context, in UpdateSongInput) (Song, error)
func (l *Library) ShowSong(ctx context.Context, in ShowSongInput) (SongDetail, error)
func (l *Library) ListSongs(ctx context.Context, in ListSongsInput, yield func(Song) error) (ListInfo, error)
```

`Selector` is constructed through validation; it cannot contain both an ID and name. Keep the visibility value next to the root song selection, not duplicated at every descendant. Canonical input validation rejects omitted parents. `--yes` is a CLI consent check, not a boolean on every API method.

Use pointer-to-slice with an allocated empty slice for an explicit `[]`; pointer-to-nil-slice would marshal as null. A generic tri-state optional-field framework is unnecessary while the public contract has no null-setting operation.

The current five-method HTTP interface can remain, renamed to unexported `jsonTransport` in the adapter. It is a legitimate testing boundary for GET/POST/PATCH/DELETE/upload mechanics. Do not pass it to resolvers or presenters. Initially `Library` can hold `*API`; HTTP test servers and fake round trippers provide controlled integration tests. Introduce a narrow consumer-owned interface only if a specific workflow test benefits enough to justify it. Do not create a giant second interface mirroring every endpoint, repository entities, a command bus, or a dependency-injection framework.

The shared `planningcenter.Collection[T]`, `Single[T]`, and `Resource[T]` are wire envelopes. Product-specific attribute DTOs are unexported `services` adapter types. DTOs retain pointers/raw presence only where required to distinguish absent/null/value; exported product models describe useful data and relationships. API response structs are never the public JSON contract. The CLI's schema mapping owns envelope v1 serialization; presenters own omission of human noise. This requires one serializer contract, not a parallel old/new schema system.

`paging.go` owns following and validating a collection cursor. A small internal walker accepts a typed page fetch and item callback, returns whether it exhausted or was deliberately stopped, and propagates errors. `resolve.go` chooses exhaustive or early-ambiguity termination; list workflows choose limit-plus-lookahead. Pagination does not choose identities. Detail composition belongs in `services/details.go`, including explicit expansion and per-collection completeness, so CLI handlers do not orchestrate several API calls.

The item callback is warranted by `--all`: CLI JSON output can encode validated records to a private temporary spool, then publish stdout after traversal succeeds, without keeping every record's full content in RAM. Identity deduplication still requires an ID set; state that cost honestly. Human output can collect bounded rows; a human explicitly requesting `--all` accepts a larger table. Do not add a persistent cache or resumable export database.

Root command construction accepts an injected dependency factory and output/error writers. Load configuration only in operational command execution, after usage validation. Use shell environment > working-directory dotenv > home dotenv > built-in defaults. Keep only documented `PCCLI_*` environment names; remove undocumented `PCO_*`/`PLANNING_CENTER_*` compatibility branches while simplifying configuration. Report invalid configuration clearly and never include secret values. `--help`, `--version`, and command-group help must not load dotenv or require credentials. Update `.env.example` for any actual configuration change, without reading real dotenv files.

## 7. Human output, JSON, errors, and cancellation

### Human output

Keep data on stdout, errors/progress/truncation notices on stderr, and the banner confined to root introduction/help. Lists should expose IDs and enough distinguishing metadata; song details group identity/credits, scheduling, arrangements, and an explicitly requested arrangement's keys. Show an actual hidden or archived state when it matters. Omit ordinary false/default booleans and empty fields. Notes and sequence belong in the relevant arrangement detail, not an ever-wider list table.

An empty list succeeds and prints `No songs found.` or the equivalent scoped noun. Empty child sections state that no children were found. A missing explicit target fails. Updates show the confirmed resource and requested state, including a transition to false or zero when that is the point of the operation. User text must be rendered as text: strip terminal control sequences from human strings without corrupting the underlying JSON values.

### One JSON contract

`--json` selects a single versioned envelope. Lists use an array in `data`; detail/mutation commands use the resource or composed detail object in `data`. `schema` is always `pccli/v1`; `meta` is always an object. IDs are strings, numeric values remain numbers, booleans remain booleans, and names use snake_case. Empty collections are `[]`. The extra `.data` in a script is a small cost for making truncation and nested completeness explicit. There is no bare-array compatibility mode and no `--json-envelope` flag.

The list shape is:

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

If the upstream total is present and relevant, `meta.api_total` may appear, documented as an observation rather than a snapshot guarantee. Do not expose opaque credential-bearing URLs or a raw API `next` link as a stable CLI cursor. To retrieve the full list, use `--all`.

The exact detail shapes are:

| Command | `data` | `meta` |
| --- | --- | --- |
| Song show | `{song, arrangements, selected_arrangement, keys}`; selected_arrangement and keys are null unless an arrangement was explicitly requested. | `arrangements` list metadata; `keys` is null if unrequested, otherwise list metadata. |
| Arrangement show | `{arrangement, keys}` | `keys` list metadata. |
| Key show | The key object. | `{}` |
| Create/update/add | The returned resource object, including parent IDs. | `{}` on a fully verified success. |
| Delete | `{deleted:true, resource:"song", id:"1001"}` and required parent IDs for children. | `{}` |

A requested empty key collection is `[]` with complete=true, distinct from unrequested `null`. Resource models include known false/zero/empty values, such as `hidden:false`; use null for a nullable field with no value, not zero. Decode missing attributes separately from null when validating required state. Include parent IDs in child models. Human suppression never determines machine fields. Keep one schema mapping near the presenter instead of using `omitempty` as a universal human/machine policy.

A bounded list with more data is a successful result with `meta.has_more=true` and `meta.complete=false`. Scripts needing everything use `--all` or check completeness. A fetch/validation failure produces no successful data document. For a large JSON request, encode to an automatically removed mode-0600 temporary file, check the complete traversal and encoding result, then copy to stdout. A pipe/write failure during that final copy can still leave partial bytes; exit nonzero, and document that consumers must check exit status. No CLI can retract bytes already written to a pipe.

With `--json`, errors are a single JSON object on stderr, with stdout empty unless the final output copy itself failed:

```json
{
  "schema": "pccli/v1",
  "error": {
    "code": "ambiguous_target",
    "message": "More than one song exactly matches Grace.",
    "resource": "song",
    "candidates": [
      {"id": "1", "title": "Grace", "hidden": false},
      {"id": "2", "title": "Grace", "hidden": true}
    ],
    "candidates_complete": false,
    "hint": "Use a song ID."
  }
}
```

JSON mode suppresses freeform progress text; an error or `meta.warnings` array carries any necessary warnings. Usage errors also use this contract when JSON was requested. Human mode keeps concise text errors. One output choice controls both data and errors, so scripts need no second flag. Document `jq '.data[]'` examples and require count/completeness checks before converting a list to a write target.

### Errors and status

Use a small operational error struct with a code, typed context, optional candidates/recovery, and an underlying cause. Required codes are `invalid_input`, `not_found`, `ambiguous_target`, `visibility_mismatch`, `incomplete_lookup`, `invalid_response`, `unsupported_capability`, `api_error`, `partial_success`, `outcome_unknown`, `verification_failed`, and `cancelled`. These are stable machine codes, not a type hierarchy or separate exit code per condition. Retain sanitized upstream HTTP status/error codes as secondary details; do not dump an arbitrary HTML response or raw request body to the terminal.

| Exit | Meaning |
| --- | --- |
| 0 | The requested read/write result completed; an empty list is successful. |
| 2 | CLI syntax/argument/flag/value/confirmation error. Wrap Cobra unknown flags/commands consistently. |
| 1 | Operational failure, ambiguous/missing target, unavailable capability, remote error, partial success, unknown write outcome, or failed post-write verification. |
| 130 | Interrupted by SIGINT. |
| 143 | Terminated by SIGTERM. |

An envelope error after a submitted write includes the resource ID and completed/uncertain stages. It must not suggest that exit 1 means nothing happened. For interruption during attachment creation, preserve recovery information in the error where available while using the signal exit code.

Wire signals to a context in `main`, execute Cobra with that context, and pass it through resolution, HTTP, retries, uploads, and output spooling. Preserve signal identity for the exit code. A normal application timeout is operational exit 1. Every retry wait selects on `ctx.Done()`; cancellation stops promptly without another request. Cleanup closes files/pipes and removes temporary files. Error formatting uses the injected stderr writer, not hard-coded `os.Stderr`, so command tests cover what actual users receive.

## 8. API capabilities, version, and attachments

The current [Services documentation entry point](https://api.planningcenteronline.com/docs/apps/services) resolves to version `2018-11-01`; its [OpenAPI description](https://api.planningcenteronline.com/services/v2/open_api/2018-11-01) supplies an additional machine-readable reference. Pin `X-PCO-API-Version: 2018-11-01` for Services requests. This is a supported contract selection, not a claim that dated documentation means the API is unmaintained. Planning Center documents [header precedence and warns against production `LATEST`](https://api.planningcenteronline.com/docs/overview/versioning). Do not add a user-selectable arbitrary API-version flag or retry without the version header when a request fails. The upload service has its own contract; do not blindly attach the Services version header to it.

Maintain a small checked-in capability table with official source links, the selected version, supported scopes, and the status of contract evidence. Encode capabilities as typed fields/enums used by validation; do not derive paths through an arbitrary verb plus association string.

| Resource operation | Decision from current evidence |
| --- | --- |
| Song list/get/create/update/delete | Supported by the [Song reference](https://api.planningcenteronline.com/docs/apps/services/versions/2018-11-01/vertices/song). Keep the existing common metadata subset. |
| Arrangement list/get/create/update/delete | Supported by the [Arrangement reference](https://api.planningcenteronline.com/docs/apps/services/versions/2018-11-01/vertices/arrangement). Map notes and full sequence in reads. No reliable default marker is established. |
| Key list/get/create/update/delete | Supported by the [Key reference](https://api.planningcenteronline.com/docs/apps/services/versions/2018-11-01/vertices/key). Keep the current name/start/end subset. |
| Attachment list at song/arrangement/key associations | The resource association references establish reads. Return what that selected association means; do not claim it is the effective inherited material set for a plan. |
| Attachment create on arrangement | Documented by the [Attachment reference](https://api.planningcenteronline.com/docs/apps/services/versions/2018-11-01/vertices/attachment). Support only its established fields and source workflows. |
| Attachment create on song/key; show/update/delete via inferred song/key paths | Not established by the consulted reference. Unsupported in this scope; no fallback destination. |
| Song key/meter filters, attachment filename/type filters and song-part writes | Contract evidence incomplete. Disable unsupported promises rather than leaving a successful-looking no-op. A future local filter is a deliberate feature, not an automatic workaround. |

For example, `AddAttachmentInput` contains a required `ArrangementScope`; it has no optional `KeyID` from which a POST path can accidentally be constructed. Attachment listing uses a separate tagged scope for Song/Arrangement/Key. Source input is a discriminated file/upload/link/content value, not four independent strings that can coexist after validation.

Before any upload: validate source count, source value, supported scope/fields, file readability and regular-file status, URL scheme, and all parent identities. File metadata overrides must be valid before the first side effect. `--url` is passed to the documented remote-link field; the CLI does not fetch arbitrary URLs itself. Empty content is invalid because this is creation, not a content-clear operation.

### Retry and streaming policy

Keep the current no-automatic-write-replay policy. GET may retry 429 and transient 502/503/504/network failures, up to three total attempts; auth errors, 404, validation errors, and malformed successful responses do not retry. Honor a valid Retry-After delay, including HTTP-date form, with a cancellable timer. Otherwise use short bounded backoff with jitter. If the server asks for a wait above the configured retry budget, fail with retry guidance; do not retry earlier than requested. Retain a per-request timeout. Traversing multiple pages is not restricted to one request timeout, but is always cancellable.

POST/PATCH/DELETE/upload get one application-level attempt unless a specific documented idempotency mechanism is added later. A connection error or timeout after dispatch can mean the server committed the write. Return an unknown outcome and verification guidance, not a confident failure-to-write claim. Do not attach an invented idempotency key.

Stream multipart upload from an opened file with bounded buffers. An `io.Pipe` writer must propagate copy failures, close on request failure/cancellation, and be joined so it cannot leak a goroutine. If the upload service requires Content-Length, compute it from the fixed multipart framing and file size or spool to disk; do not restore whole-file RAM buffering. Require a regular file in this initial workflow. Contract verification of chunked upload acceptance can occur without making it a condition of the architecture.

### Partial success and recovery

The official [upload guide](https://api.planningcenteronline.com/docs/overview/file-uploads) establishes two operations: upload to receive a file UUID, then assign that UUID. The response also includes an expiration time. Preserve UUID, filename, size, expiration when present, resolved destination IDs, and the stage reached. Treat the UUID as recovery data; do not put credentials or file contents in errors.

| Stage/result | Required behavior |
| --- | --- |
| Upload rejected before a usable UUID | Return the upload error. No attachment request. |
| Upload succeeded; attachment rejected definitively, e.g. documented validation response | Return `partial_success` with upload receipt and destination. Explain that no attachment was confirmed. After correcting the problem, the user can reuse `--upload-id` before expiry where the service allows it. |
| Upload succeeded; attachment response lost, timed out, malformed, or otherwise cannot establish commitment | Return `outcome_unknown`, preserve the same receipt, and do not automatically replay either operation. |
| Attachment creation confirmed; verification/presentation fails | Preserve the attachment ID and say it was created. Failure to display it is not evidence of rollback. |

For an uncertain attachment outcome, tell the operator to inspect the arrangement's attachments before retrying. Where the API supports selecting `file_upload_identifier`, exhaustively inspect that field and compare the UUID as additional evidence; the reference states that this attribute requires field selection. A unique matching UUID can confirm a completed attachment. A filename match alone cannot. An absent match cannot conclusively prove non-creation under delayed visibility or concurrent changes, so do not auto-retry from that result. If the API cannot expose sufficient evidence, the operator must verify in Planning Center before intentionally rerunning.

Human errors print a usable receipt and a safely shell-quoted recovery example. Envelope errors include `recovery.upload_id`, `expires_at`, resolved IDs, `stage`, and `attachment_outcome` (`not_created` only when established, otherwise `unknown`). Do not delete a staged upload or create a replacement file automatically; cleanup capabilities and UUID reuse semantics are not established. No local recovery database is required.

## 9. Alternatives rejected

| Alternative | What it gets right | Why this design rejects it |
| --- | --- | --- |
| Keep `songs arrangements`/`songs keys` with explicit action children | Shows the parent concept in help. | It does not encode a specific parent; users still supply song/arrangement separately. Four actions plus positional parent chains become harder to scan and evolve. Sibling nouns with named scope teach one grammar. |
| Flatten everything into `songs add-key`, `songs attach`, etc. | Starts every journey from a song. | CRUD verbs become an irregular expanding inventory. Arrangement/key detail and list operations lose a consistent home. The composed song view already provides the song-first entry. |
| Keep search as a second primary workflow | Positional search is convenient. | It duplicates list filters, limits, and examples without a distinct search capability. Use list search and remove the separate command. |
| Allow automatic sole/default arrangement selection everywhere | Saves a parent flag. | A script's behavior changes when another arrangement appears; arbitrary default-name selection is unsafe. Explicit parent requirements are a small recurring cost that purchases clarity. |
| Force exact-only names for every read | Simplifies resolution. | It removes an existing useful discovery convenience. Exact-first/unique-partial reads are safe when exhaustive; write selection stays exact. |
| Bare JSON arrays or an optional second envelope mode | Makes simple jq expressions slightly shorter. | Arrays cannot carry completeness metadata. A second mode duplicates documentation, serialization, and tests. With no adoption to preserve, one envelope is simpler overall. |
| Only split the large files | Low review cost. | Leaves identity policy, completeness, and presence semantics unchanged. Fix those contracts, then split along their ownership. |
| Full repository/service/entity/use-case layering | Strict dependency separation. | Adds interfaces and mappings beyond the needs of a small in-process CLI. Existing packages plus a concrete typed adapter provide the needed boundary. |
| Generated generic CRUD engine or raw API mode | Easy endpoint expansion. | Makes endpoint coverage the product and hides policy in hooks. It would recreate the mismatch this project explicitly wants to avoid. |
| New interactive picker/TUI | Friendly disambiguation for some users. | Does not fix script correctness or uncertain remote capabilities. Deterministic errors plus IDs solve the present need without stateful UI infrastructure. |

The cost of sibling commands is longer key operations. Make that cost visible and acceptable through help examples and copyable IDs; do not introduce implicit sessions to hide it. Usability validation should test whether operators can discover those parents, not vote again on the architecture.

## 10. Replacement and release strategy

The user's explicit statement that there is no adoption resolves the migration question. The empty local tag list and `0.0.0` manifest are no longer relevant to a compatibility decision. Do not spend implementation effort investigating old usage, building adapters, or planning deprecation periods. Replace the current interface and documentation together.

| Change category | Delivery decision |
| --- | --- |
| Identity, scope, ignored input, patch presence | Fix first because these determine correctness regardless of the final spelling. Intermediate development commits may still use existing commands; they are not a supported compatibility layer. |
| Internal extraction / envelope consolidation | Keep each internal move separate from unrelated behavior changes so the diff is reviewable. Preserve only behavior that belongs in the chosen end state. |
| Sibling resource commands | Replace nested song child commands. Require explicit verbs and parent flags. Delete old constructors after their useful logic moves. |
| Search and selector flags | Remove separate search, implicit parent defaults, positional arrangement expansion, selection uses of `--hidden`, and the arrangement pitch `--key` flag. Use list search, `--visibility`, `--arrangement`, and `--chart-key`. |
| JSON/errors | Replace bare arrays/raw resource output with one `--json` envelope and structured stderr errors. Update every jq example and schema fixture. |
| Unsupported capabilities | Remove unsupported filters/fields from help and command registration. Reject unsupported attachment scope before upload; never silently fall back. |
| Deferred features | Key/song attachment creation, archive operations, additional field clears, and bulk work require their own evidence and explicit future scope. |

Keep README concise, quick start setup-first, and operational examples in the workflow guide. Update command reference, VHS tapes, generated-demo references, and fixtures with the interface change. Do not publish old and new command tutorials side by side. Do not copy this report into public `docs/`.

Use Conventional Commits and the existing Release Please process. Mark the interface replacement with the repository's breaking-change convention so release history is accurate, even though there are no users to migrate. Do not hand-edit the release manifest to manufacture a version. This review does not authorize publishing; release readiness is the implementation acceptance boundary, and no backward-compatibility gate remains.

## 11. Gap register, verification, and implementation sequence

### Prioritized gaps

P0 blocks safe name-based writes. P1 blocks the relevant changed workflow from release. P2 is a bounded follow-up; it must not hold unrelated fixes hostage.

| Gap | Priority | Evidence needed / decision | Exit criterion |
| --- | --- | --- | --- |
| G1 complete identity lookup | P0 | Source and supplied reproduction already sufficient to begin. | Every resolver traverses relevant pages or proves ambiguity; incomplete lookup produces zero writes/uploads. |
| G4 parent inference | P0 for writes | No default marker established; do not wait for one. | Multiple possible arrangements never select by order/name convention. Canonical parents required. |
| G2 presence and clear semantics | P1 | Request-aware serialization tests now; field clear/readback contract checks for supported empty values. | Empty/zero/false/[] transmitted exactly or rejected before HTTP; no empty-patch success; affected read fields inspectable. |
| G3 visibility | P1 | Test identical name/ID behavior across actions; verify all-state list semantics. | Hide/unhide works with explicit selection; cross-state duplicates handled; no ID bypass. |
| G5 behavioral tests | P1, accompanies every fix | Existing test inventory shows missing failure/pagination sequences. | Tests inspect methods, paths, queries, bodies, request ordering, and absence of effects. Demo mock is not the proof. |
| G6 input validation | P1 | No further external evidence required. | Extraneous args, conflicting flags, invalid numeric selectors/limits/lengths/pitches rejected before dependency loading. |
| G7 API capability audit, expanded | P1 for each capability | Versioned docs/OpenAPI; explicitly authorized disposable contract checks where docs do not settle behavior. | Supported matrix checked in; unproved attachment scopes/filters/fields blocked before upload; examples match. |
| G8 version and inspectable fields | P1 | Header tests and documented supported version; map notes/full sequence. | Every Services request pins version; clear verification does not rely on absent fields defaulting to zero. |
| G9 cancellation, upload, recovery | P1 for attachment release | Streaming and failure injection tests; upload UUID expiry/reuse/readback contract limits recorded. | Bounded upload memory, prompt cancellation, receipt on partial/unknown completion, no automatic write replay. |
| G10 grammar/defaults | P2 after safety fixes | Help-driven journey checks with novice/occasional users. | One resource/action/context grammar; obsolete commands removed; parents always explicit. |
| G11 output contract | P1 for JSON; P2 presentation | Define one v1 schema with empty/null/truncation semantics. | All JSON uses the envelope; completeness/errors tested; no stdout chatter; false/zero values explicit. |
| G12 replacement/docs consistency | P1 before release | User has resolved adoption: no compatibility needed. Inspect all examples and help, not old-user usage. | One command reference, tested examples, no old paths/aliases or dual schemas. |
| G13 response/cursor/scope validation | P0 for resolver path, P1 elsewhere | Synthetic invalid envelopes, hostile links, cycles, repeated IDs, wrong scoped IDs. | Invalid data cannot prove absence or success; credentials never follow an untrusted next URL. |
| G14 archived arrangement scope | P2, but block affected writes | Confirm standard vs unscoped arrangement association and direct-ID archive behavior. | Archive status represented honestly; no claim that song visibility includes it; detected archived writes fail pending explicit support. |
| G15 output sanitization / error writer | P2 | Control-character and injected-writer tests. | Human output cannot execute terminal controls; errors test through the same writer path as production. |
| G16 matching query completeness | P1 for any narrowing optimization | General equality/wildcard docs are available; Song-specific contains, literal escaping, Unicode collation, and omitted-hidden all-state listing need precise verification. | Every optimized predicate is an established superset of local matches; otherwise use complete local filtering or the stated unsupported all-list result, never a potentially incomplete identity proof. |

Unknown remote clear values should block release of that particular clear behavior, not pagination fixes or the internal split. Unknown key attachment creation should remain unavailable, not delay arrangement attachment reliability. Passing mocks or local tests never upgrades a contract unknown to live validation.

### Required behavioral matrix

Use `httptest.Server` or a request-aware round tripper with synthetic credentials and an injected client. Every unsafe-path test asserts both the returned result and that mutation/upload counts are zero. Test plain and JSON forms for CLI behavior, including the envelope schema. Do not make tests load working-directory or home credentials.

| Case | Expected assertion |
| --- | --- |
| Exact Grace on page one plus second exact Grace on page two | Ambiguous; no DELETE/PATCH/POST/upload; bounded list limit has no effect on resolution. |
| Page two fails during lookup | `incomplete_lookup`; no chosen resource and no mutation. |
| Two partial names precede a later exact name | Exact wins only after its uniqueness is established; no premature partial ambiguity. |
| Duplicate identical ID repeated across pages | One identity; conflicting name/state snapshots fail. |
| Visible/hidden duplicates; visibility visible/hidden/all | Correct selected set for list/name/ID; all-state name is ambiguous; IDs remain independently selectable with the correct flag. |
| Numeric title, negative, signed, zero, leading-zero, huge numeric selector | `name:1234` targets a name; invalid ID-like inputs fail locally; huge digits remain a string ID, never a title fallback. |
| Literal wildcard characters and non-ASCII case variants | No accidental query-language interpretation; exhaustive fallback preserves the exact/substring match contract; no unproved narrowing optimization. |
| Two Default arrangements; multiple non-default arrangements | Missing parent is usage error; explicit duplicate name is ambiguous; neither chooses first. |
| Hide/unhide by name and ID | Same visibility selection; new hidden state never changes lookup filter. |
| Clear author/notes/sequence; length=0; boolean=false | Exact payload values; successful response/readback confirms the requested change; server ignoring a clear is not success. |
| Invalid title/name/pitch, CCLI=0, BPM NaN, huge duration, unknown args | Exit 2 before credentials/network; presence is distinct from validity. |
| Explicit arrangement on song with zero arrangements | Not found error; no silently ignored input. |
| Wrong song/arrangement/key IDs | Scoped GETs and relationship validation; no global fallback or fabricated key. |
| List limit 1, 25, 101, 1000, all; empty collection | Correct total-item bound and lookahead metadata; zero is invalid; empty succeeds; all is exhaustive. |
| Large JSON list with late page/encoding failure | No committed successful stdout document; private spool cleaned; memory does not contain the whole export. |
| `{}`, `data:null`, wrong type, missing ID, empty 200 read/write response | Invalid response, not empty success; unknown write outcome if request may have committed. |
| Cross-origin/changed-path next link, redirect, altered filter, cycle | Rejected before sending credentials/request to the wrong destination; resolution incomplete. |
| Local file + unsupported key attachment destination | Capability error before upload. |
| Upload succeeds; attach rejects or times out | Receipt preserved, correct definitive/unknown outcome, no replay; recovery output machine-readable under --json. |
| SIGINT during 429 wait; cancellation during multipart write | Prompt return, correct exit, no next attempt, files/pipes closed. |
| Help/version with absent/bad credentials | Help/version works without configuration or network; data operations still validate credentials. |
| JSON/plain output and injected stdout/stderr | Expected envelope schema, no banner/progress on stdout, correct error routing, no terminal controls in human values. |
| Removed old command/flag forms | Clear usage error, no accidental fallback command or side effect; docs show only the new forms. |

### PR-sized sequence

Each step is a reviewable unit; split a large row into resource-specific follow-ups rather than combining several phases in one change. Run `go test ./...` for code changes, `go vet ./...`, and a build as appropriate. Run `go mod tidy` only when dependencies change. Use a fresh temporary Go cache if the previously reported cache issue recurs. Test both plain output and `--json`; use mock mode only for demos/screenshots. Do not change application behavior through a documentation-only release.

**1. Fix unsafe identity lookup without changing the command tree.**

Start directly in `internal/services/library.go` and `internal/cli/songs.go`: add a typed paged collection path retaining links, validate envelopes/cursors, and add `internal/cli/resolution_test.go` with a request-aware multi-page fixture. Reuse the envelope in `planningcenter/jsonapi.go` rather than introducing a third one. Route all name resolvers through exhaustive traversal. Remove first/default arrangement inference for writes and require an explicit parent; fetch numeric key targets. Add the ignored-explicit-arrangement regression. Existing single-page list functions can remain temporarily within this development step; the final list contract follows in step 6.

Dependencies: none. Acceptance: G1/G4 and resolver-path G13 tests pass, with zero-effect assertions on all incomplete/ambiguous paths; relevant plain/JSON workflows still work with explicit valid parents. Update affected docs in the same PR. Stop if the pager cannot establish a complete candidate set; return incomplete instead of retaining the old write behavior. Do not postpone the fix for command renaming or live API access.

**2. Repair patch presence, input validation, and visibility selection.**

Introduce create/patch types, `--visibility`, the selector parser, presence-aware serialization, finite/range validation, and notes/full-sequence read fields. Remove selection uses of `--hidden`; it now only changes song state. Add same-policy name/ID hide/unhide tests and exact false/zero/empty payload tests. Implement confirmation of requested clears; return explicit unsupported errors where a clearing representation cannot be established.

Depends on step 1. Acceptance: G2/G3/G6 pass; successful no-op payloads are impossible; no valid field is silently dropped. Stop only the uncertain field/capability, not the entire release. The patch types must not grow a general JSON patch language.

**3. Pin the API contract and narrow unsupported operations.**

Check in the version/capability inventory with source links, pin the Services header, and separate attachment list scope from add scope. Block inferred song/key creates before upload and remove unsupported filter/field flags. Confirm narrowing-query predicates or retain the exhaustive local fallback. Correct sequence documentation and examples. Add a contract-check checklist for any later disposable-organization test; running it requires its own authorization and is not implied by this design review.

Depends on step 1; can be reviewed before step 2 finishes, but both must land before broad mutation release. Acceptance: no unsupported destination can upload a file; each advertised field/path has evidence; docs/demo commands execute against the mock. Stop before adding a speculative API endpoint or silently redirecting a file to a different parent.

**4. Extract the workflow/API boundary.**

Move resolution and composition into `services.Library`, move typed transport operations into `services.API`, split CLI commands/presenters by resource, inject the lazy factory and writers, and delete duplicate envelopes/helpers only after callers migrate. Retain request-aware tests through the real adapter.

Depends on steps 1–3. Acceptance: no Cobra types or rendering in Services; no URL/query/multipart logic in CLI; workflows do not call generic HTTP methods; the corrected policy and requests remain characterized. Stop adding layers when these ownership conditions hold.

**5. Replace the command tree with sibling resource commands.**

Add explicit list/show resource actions and required parent flags, `--chart-key`, and one list/search workflow with literal text semantics. Delete old nested forms, separate search, positional parent variants, and default parent inference. Simplify configuration to documented PCCLI names. Update help, README, workflow/command docs, VHS tapes, and demo fixtures together.

Depends on step 4. Acceptance: every journey can be discovered from help and completed using IDs or exact names; removed forms fail as ordinary unknown commands/flags. Run focused usability checks: locate a song, inspect one arrangement, edit notes, and choose an attachment destination using help alone. Confusing wording may be revised without reopening the architecture. Stop before expanding unsupported attachment scope or introducing aliases to avoid updating examples.

**6. Complete list/output/cancellation contracts.**

Implement true total-result limits and `--all`, the single --json envelope v1, per-child completeness, streaming-to-spool JSON collection output, injected error rendering, signal context, and cancellable GET retry waits. Replace old JSON fixtures with v1 schema assertions and update jq examples. Human details can be refined in a separate small PR within this phase.

Depends on step 4; all commands from step 5 must consume the same result contract before release. Acceptance: large lists traverse correctly; failures do not look like successful exports; every data command and error follows the documented JSON contract; help remains credential-free; cancellation tests are deterministic. Stop before adding a second JSON mode or a public pagination cursor without a demonstrated need.

**7. Make arrangement attachment addition recoverable.**

Implement streamed multipart transfer, typed upload receipt including expiration, outcome classification, and the recovery instructions/JSON. Add injection tests for file-copy failure, cancellation, upload success plus attachment rejection, lost attachment response, and post-create output failure. Verify memory behavior with a large synthetic temporary file without contacting Planning Center.

Depends on steps 3, 4, and the error/cancellation portion of step 6. Acceptance: no automatic write replay, no lost known upload UUID, no unsupported scope accepted, bounded file buffering, no leaked goroutine/file. If reconciliation cannot prove an outcome, report uncertainty; do not invent automatic recovery.

**8. Finish release-readiness verification.**

Run the full test/build/docs/demo checks; review examples for accidental first-result selection such as unguarded `jq '.data[0].id'`. Replace those examples with an explicit count/completeness check or ID-based follow-up. Prepare release notes distinguishing bug fixes, the new syntax/output contract, and unsupported capabilities. Do not claim live contract validation that was not performed. Verify that no obsolete examples, compatibility handlers, or second serializers remain. Publishing remains a separate authorized action.

Depends on the phases included in that release; safety fixes can be reviewed earlier on their own. Acceptance: the command contract and documentation agree with executable behavior, generated demos use `PCCLI_MOCK=true`, and no secrets or generated binaries are included. The remaining product backlog is bounded: key/song attachment creation evidence, archive workflow, and broader field capabilities require separate feature work.

This completes the architecture decision. The implementing agent should start with step 1; no further architect iteration is required to begin.
