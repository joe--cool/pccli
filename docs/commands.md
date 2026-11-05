---
title: Services Commands
nav_order: 6
---

# Services Commands

pccli is unofficial and is not built, endorsed, or supported by Planning Center. Commands follow `pccli services RESOURCE ACTION`. Use `--help` on any command for its installed flags.

## Shared behavior

| Option or rule | Behavior |
| --- | --- |
| `--json` | Versioned machine output; supported on every data command |
| `--help`, `-h` | Help without credentials |
| `--version`, `-v` | Root version without credentials |
| `--visibility visible\|hidden\|all` | Select existing songs; default `visible`, including parent lookups and direct IDs |
| `--include-archived` | Explicitly include archived arrangements, or returned archived service types/teams/candidate people where relevant |
| `--limit N` | Total results to display, default 25; positive integer |
| `--all` | Return the complete selected collection; conflicts with an explicit `--limit` |
| `--yes` | Required for every delete; never bypasses target validation |

### IDs, names, and parents

Targets accept a positive decimal ID or name. Use `id:1001` to force an ID and `name:1001` to force an exact numeric name. `name:Amazing Grace` forces an exact name. Exact name matching is case-insensitive. Read targets prefer an exact match, then accept a unique literal partial match. Writes and parent flags require IDs or exact names. Duplicate exact names are ambiguous; use the displayed IDs to choose.

Search uses literal text, not API wildcard syntax. `%` and `_` are ordinary characters. `songs list --title TEXT` is an exact case-insensitive filter; `--search TEXT` is a literal contains filter. They conflict with each other.

Arrangements require `--song SONG`. Keys require `--song SONG --arrangement ARRANGEMENT`. There is no automatic first/default arrangement. Attachment scope is a song, optionally an arrangement, optionally a key; `--key` requires `--arrangement`. Parent flags use the same visibility and ambiguity rules across follow-up commands.

Lookups traverse all candidate pages before choosing a target. Display limits never limit identity resolution. Page failures, malformed records, or ambiguous matches stop the operation.

## Service planning

Use the [planning workflow](planning-a-service.html) for an end-to-end example. All commands below support `--json`; lists accept `--limit N` or `--all`.

| Context | Required parent flags |
| --- | --- |
| Plans and teams | `--service-type SERVICE_TYPE` |
| Items, times, assignments, and needed positions | `--service-type SERVICE_TYPE --plan PLAN` |
| Team people and positions | `--service-type SERVICE_TYPE --team TEAM` |

Parent flags and write targets require exact names or IDs. Read targets also accept a unique literal partial name. Use IDs for untitled plans, unnamed times, duplicate titles, or repeated assignment names. Dates are displayed for orientation and are not selectors.

`--include-archived` selects returned archived service types, teams, arrangements, or candidate people where relevant. It cannot recover records the provider omits. Song-link selection uses `--visibility`; it does not filter planning records.

### Service types and plans

```text
pccli services service-types list
pccli services service-types show SERVICE_TYPE
pccli services plans list --service-type SERVICE_TYPE [--filter future|past|no_dates]
pccli services plans show PLAN --service-type SERVICE_TYPE
pccli services plans create --service-type SERVICE_TYPE [fields]
pccli services plans update PLAN --service-type SERVICE_TYPE [fields]
pccli services plans delete PLAN --service-type SERVICE_TYPE --yes
```

| Plan field | Create | Update | Behavior |
| --- | --- | --- | --- |
| `--title TEXT` | Yes | Yes | Empty text requests an untitled plan |
| `--series-title TEXT` | Yes | Yes | Empty text clears the label; conflicts with `--series-id` |
| `--series-id ID` | Yes | Yes | Existing positive decimal Series ID; validated before writing |
| `--public=true|false` | Yes | Yes | Sets public access explicitly |
| `--reminders-disabled=true|false` | No | Yes | Changes the plan reminder setting |

Creation makes one undated plan; labels are optional. Add service or rehearsal times to date it. Updates require at least one field, preserve omissions, and transmit explicit empty or false values. Creates and updates verify readable values independently. Deletes require `--yes` and verify HTTP 404 afterward.

### Copy selected plan components

```text
pccli services plans copy SOURCE --service-type SERVICE_TYPE [components] [options]
```

Choose at least one component:

| Flag | Copies |
| --- | --- |
| `--copy-items` | Supported item fields, selected song/arrangement/key IDs, and order |
| `--copy-times` | Service, rehearsal, and other times; requires a date option below |
| `--copy-people` | Supported whole-plan assignments, reset to unconfirmed with notification preparation off |
| `--copy-needed-positions` | Supported whole-plan needed quantities; position IDs remain unverified |
| `--copy-settings` | Known public-access and reminder settings |

Available source labels and Series selection are copied; `--title TEXT` overrides the title, including an explicit empty value. Without `--copy-settings`, creation sends `public: false`. Service-type scheduled publication rules still apply.

With `--copy-times`, choose exactly one of `--time-offset DURATION` or `--preserve-dates`. Offsets such as `168h`, `-24h`, or `0` are elapsed time, not calendar weeks or local-clock scheduling. Daylight-saving transitions can change the local clock time. No timezone or date shift is inferred.

Copying creates a new plan in the same service type through multiple verified requests. Copied songs require known selected key IDs; media, files, layouts, and other unlisted fields are excluded. It is not a full clone or remote transaction: read the [copied fields, exclusions, and recovery behavior](planning-a-service.html#copy-selected-parts-of-a-plan) before using it. Errors retain known destination IDs and the failed stage; inspect those records before retrying. Needed-position copies preserve their accepted/unverified result.

### Order of service

```text
pccli services items list --service-type SERVICE_TYPE --plan PLAN
pccli services items show ITEM --service-type SERVICE_TYPE --plan PLAN
pccli services items create --service-type SERVICE_TYPE --plan PLAN [fields]
pccli services items update ITEM --service-type SERVICE_TYPE --plan PLAN [fields]
pccli services items reorder --service-type SERVICE_TYPE --plan PLAN --sequence ID,ID
pccli services items delete ITEM --service-type SERVICE_TYPE --plan PLAN --yes
```

| Item field | Behavior |
| --- | --- |
| `--title`, `--description`, `--html-details` | Text; explicit empty values clear |
| `--length` | Nonnegative seconds or `m:ss`; zero is transmitted |
| `--service-position` | `pre`, `during`, or `post` |
| `--chart-sequence LABEL,LABEL` | Arrangement section labels; empty clears |
| `--type item|header` | Create only; omit when selecting a song |
| `--position N` | Create-only initial sequence number, nonnegative |
| `--song SONG --arrangement ARRANGEMENT [--key KEY]` | Select song links explicitly; no first/default arrangement |

Song linking uses `--visibility` and `--include-archived` for existing song/arrangement selection. Headers cannot acquire song links. Direct type changes, empty song-link clearing, and media/layout editing are not exposed.

Lists preserve provider order and show sequence, service position, and song ID when available. Reorder requires every scoped item ID exactly once; incomplete, duplicate, or foreign selections stop before writing. The complete collection is rechecked before the request and read again afterward to verify order.

### Service and rehearsal times

```text
pccli services plan-times list --service-type SERVICE_TYPE --plan PLAN
pccli services plan-times show TIME --service-type SERVICE_TYPE --plan PLAN
pccli services plan-times create --service-type SERVICE_TYPE --plan PLAN --starts-at TIMESTAMP --time-type TYPE [--ends-at TIMESTAMP] [--name TEXT]
pccli services plan-times update TIME --service-type SERVICE_TYPE --plan PLAN [fields]
pccli services plan-times delete TIME --service-type SERVICE_TYPE --plan PLAN --yes
```

| Time field | Behavior |
| --- | --- |
| `--starts-at`, `--ends-at` | RFC3339 timestamps with an explicit offset or `Z`; empty is rejected |
| `--time-type` | `service`, `rehearsal`, or `other` |
| `--name` | Optional label; empty clears |

Creation requires a start and type, with no invented end or duration. Years must be 0001–9999, with at most nanosecond precision. End cannot precede start. A one-sided timestamp update validates the resulting pair using current known values. Readback compares equivalent instants; output retains provider timestamp text.

These commands affect one time. Future-plan bulk changes, time-specific team assignment, and reminder rules are not exposed.

### Teams and scheduling candidates

```text
pccli services teams list --service-type SERVICE_TYPE
pccli services teams show TEAM --service-type SERVICE_TYPE
pccli services team-people list --service-type SERVICE_TYPE --team TEAM
pccli services team-people show PERSON --service-type SERVICE_TYPE --team TEAM
pccli services team-positions list --service-type SERVICE_TYPE --team TEAM
pccli services team-positions show POSITION --service-type SERVICE_TYPE --team TEAM
```

Team people are membership candidates, not guarantees of availability, eligibility, or permission to schedule. Candidate reads exclude known archived people by default; `--include-archived` permits inspection. Unknown archive state remains unavailable rather than proving activity.

### Scheduled people

```text
pccli services scheduled-people list --service-type SERVICE_TYPE --plan PLAN
pccli services scheduled-people show ASSIGNMENT --service-type SERVICE_TYPE --plan PLAN
pccli services scheduled-people create --service-type SERVICE_TYPE --plan PLAN --team TEAM --person PERSON --position POSITION [fields]
pccli services scheduled-people update ASSIGNMENT --service-type SERVICE_TYPE --plan PLAN [fields]
pccli services scheduled-people delete ASSIGNMENT --service-type SERVICE_TYPE --plan PLAN [--team TEAM] --yes
```

| Assignment field | Behavior |
| --- | --- |
| `--status STATE` | `unconfirmed`, `confirmed`, `declined`, or `U`, `C`, `D` |
| `--notes TEXT`, `--decline-reason TEXT` | Explicit empty strings clear |
| `--position POSITION` | Existing team position; updates require a unique serialized name |
| `--team TEAM` on update/delete | Guards the existing team; does not move the assignment |

Creation requires a known whole-plan team, current nonarchived member, and a position in that team. Duplicate position labels are ambiguous even when an ID is supplied, because assignments store the label. Creates default to unconfirmed and explicitly disable notification preparation. Updates preserve omitted preparation state; no command sends invitations or notifications.

Split-time and known partial assignments are rejected. Creation and updates reject unknown person archive state. Deletion permits cleanup after a person leaves the team or is archived, while retaining plan/team validation and confirmation. Moving a person or team requires explicit deletion and creation.

### Needed positions

```text
pccli services needed-positions list --service-type SERVICE_TYPE --plan PLAN
pccli services needed-positions show NEED --service-type SERVICE_TYPE --plan PLAN
pccli services needed-positions create --service-type SERVICE_TYPE --plan PLAN --team TEAM --position POSITION --quantity N
pccli services needed-positions update NEED --service-type SERVICE_TYPE --plan PLAN --quantity N
pccli services needed-positions delete NEED --service-type SERVICE_TYPE --plan PLAN --yes
```

Quantities are required and nonnegative, including explicit zero. Whole-plan teams are supported; explicit time or time-preference scopes are rejected. The raw provider `scheduled_to` value is retained without inferring its meaning.

Create verifies quantity, team, and readable position name, but the API does not expose the requested position ID for readback. Its result reports `outcome: accepted`, `unverified_fields: ["team_position_id"]`, and a separate `requested_position_id`. Quantity updates report `verified` after readback. The read model never invents a position ID.

## Songs

```text
pccli services songs list [filters] [--limit N | --all]
pccli services songs show SONG [--arrangement ARRANGEMENT]
pccli services songs create --title TITLE [metadata]
pccli services songs create --ccli NUMBER [metadata]
pccli services songs update SONG [metadata] [--visibility STATE]
pccli services songs delete SONG --yes [--visibility STATE]
pccli services songs schedules SONG [--limit N | --all]
pccli services songs last-scheduled SONG --service-type ID [--limit N | --all]
```

List filters:

| Flag | Meaning |
| --- | --- |
| `--search TEXT`, `-q TEXT` | Literal title contains match |
| `--title TEXT` | Literal exact title |
| `--author TEXT` | Exact case-insensitive author |
| `--ccli NUMBER` | Positive CCLI number |
| `--themes TEXT` | Exact case-insensitive themes text |
| `--order FIELD` | `title`, `created_at`, `updated_at`, or `last_scheduled_at`; prefix `-` to reverse |
| `--visibility STATE` | `visible`, `hidden`, or `all` |

`last-scheduled` requires a positive `--service-type ID` because Planning Center schedules the same song within different service types.

Show gives song metadata and an arrangement overview. Only `--arrangement` expands a selected arrangement and its keys. Empty related collections succeed; an explicitly requested missing target fails. Use `--include-archived` to include archived versions.

Create requires `--title` or `--ccli`. With CCLI alone, pccli uses Planning Center's documented title-based metadata import. Song metadata flags are listed in the [field reference](song-fields.html#songs).

Visibility selects the existing record; `--hidden` changes its state:

```sh
pccli services songs update 1001 --hidden=true
pccli services songs update 1001 --visibility hidden --hidden=false
```

## Arrangements

```text
pccli services arrangements list --song SONG [--limit N | --all]
pccli services arrangements show ARRANGEMENT --song SONG
pccli services arrangements create --song SONG --name NAME [metadata]
pccli services arrangements update ARRANGEMENT --song SONG [metadata]
pccli services arrangements delete ARRANGEMENT --song SONG --yes
pccli services arrangements archive ARRANGEMENT --song SONG
pccli services arrangements unarchive ARRANGEMENT --song SONG --include-archived
pccli services arrangements sections ARRANGEMENT --song SONG
```

Use `--include-archived` to select archived versions explicitly; ordinary lists and lookups exclude them. All commands accept song visibility selection. `sections` reads the sections derived from a chord chart; `--sequence` changes the order of full section labels.

The [arrangement field reference](song-fields.html#arrangements) covers chart text, pitch, tempo, notes, sequence, print settings, and provider import fields. Chart pitch uses `--chart-key`; `--key` selects a key record only in attachment scope.

## Keys

```text
pccli services keys list --song SONG --arrangement ARRANGEMENT [--limit N | --all]
pccli services keys show KEY --song SONG --arrangement ARRANGEMENT
pccli services keys create --song SONG --arrangement ARRANGEMENT --name NAME --start PITCH [metadata]
pccli services keys update KEY --song SONG --arrangement ARRANGEMENT [metadata]
pccli services keys delete KEY --song SONG --arrangement ARRANGEMENT --yes
```

Starting and ending pitches use `--start` and `--end`. On create, omitted `--end` defaults to `--start`. All commands accept `--visibility` and `--include-archived`. See [key fields](song-fields.html#keys) for alternate-key JSON and provider imports.

## Attachments

```text
pccli services attachments list --song SONG [--arrangement ARRANGEMENT [--key KEY]] [--limit N | --all]
pccli services attachments show ATTACHMENT --song SONG [--arrangement ARRANGEMENT [--key KEY]]
pccli services attachments add --song SONG [--arrangement ARRANGEMENT [--key KEY]] SOURCE [metadata]
pccli services attachments update ATTACHMENT --song SONG [--arrangement ARRANGEMENT [--key KEY]] [metadata]
pccli services attachments delete ATTACHMENT --song SONG [--arrangement ARRANGEMENT [--key KEY]] --yes
```

Every operation uses the selected association; a song list does not imply all child attachments. Attachment targets use an ID or filename. Use IDs for repeated filenames.

| Flag | Add | Update |
| --- | --- | --- |
| `--file PATH` | Upload a local regular file | Use `--upload-id` or `--content` for replacement |
| `--upload-id UUID` | Attach an existing upload receipt | Replace with an existing upload receipt |
| `--url URL` | HTTP(S) link without URL credentials | Replace or explicitly clear link |
| `--content TEXT` | Inline content | Replace or explicitly clear content |
| `--filename TEXT` | Destination filename | Change filename |
| `--attachment-type-ids TEXT` | File classification IDs | Replace classification IDs |
| `--page-order TEXT` | Planning Center page order string | Change page order |
| `--song-part TEXT` | Song section label | Change song section label |
| `--item-details=true\|false` | Set item-details import state | Change item-details import state |

Add requires exactly one nonempty source: `--file`, `--upload-id`, `--url`, or `--content`. Update accepts at most one nonempty replacement source. Both accept `--visibility` and `--include-archived` for parent selection.

Attachment updates return `outcome: verified` when requested values can be checked through readback. `outcome: accepted` and `unverified_fields` identify values the API accepted but does not expose for confirmation. Inspect the attachment in Planning Center when those effects matter.

File upload and attachment creation are separate stages. If a later stage fails, keep the reported upload receipt, expiry, destination, and outcome. A rejection differs from an unknown result: inspect the destination before retrying an unknown result. Reuse `--upload-id` only after confirming that the attachment was not created; do not upload the file again automatically.

## Tags and catalogs

```text
pccli services tags list [--song SONG [--arrangement ARRANGEMENT]] [--limit N | --all]
pccli services tags replace --song SONG [--arrangement ARRANGEMENT] --tags TAG,TAG
pccli services tags replace --song SONG [--arrangement ARRANGEMENT] --clear
pccli services tag-groups list [--limit N | --all]
pccli services attachment-types list [--limit N | --all]
```

Unscoped `tags list` discovers tags for songs and arrangements; a scope lists assignments. `replace` sends the **entire desired tag set**, not an addition. `--clear` explicitly removes assignments and conflicts with `--tags`. Tags are selected by ID or exact case-insensitive name. Duplicate names across groups require IDs. Required groups and single-selection rules are enforced.

Tags belong to songs or arrangements, not key records. Scoped tag commands accept `--visibility` and `--include-archived`. Tag groups and attachment types are read-only catalogs. Song `--themes` is text metadata and is separate from relational tags.

## JSON and errors

{% include terminal-demo.html name="json-demo" title="Export songs for a script" caption="Inspect the JSON envelope and completeness, then extract song IDs and titles with jq." %}

<details markdown="1">
<summary>Read the JSON export steps</summary>

These examples use Bash and require [jq](https://jqlang.org/). Enable pipeline error propagation first:

```sh
set -o pipefail
pccli --json services songs list --all | jq '{schema, meta}'
pccli --json services songs list --all | jq '.data[] | {id, title}'
```

The mock export reports schema **pccli/v1**, **returned: 2**, **has_more: false**, and **complete: true**. The second command extracts **1001 / Amazing Grace** and **1002 / Great Is Thy Faithfulness**. Hidden songs remain excluded by default even with `--all`. In automation, stop on a nonzero pipeline status before using the result; `jq` succeeding alone does not establish that pccli succeeded.

</details>

Successful data commands write one envelope to stdout:

```json
{
  "schema": "pccli/v1",
  "data": [],
  "meta": {"returned": 0, "has_more": false, "complete": true}
}
```

List `meta` describes the returned selection. `has_more: true` and `complete: false` mean the display limit omitted results; use `--all` to export all selected results. Detail results carry completeness metadata for requested child collections. An unrequested song expansion is `null`, distinct from a requested empty array. IDs remain strings; machine output retains false, zero, and empty values.

```sh
pccli --json services songs list --all | jq '.data[] | {id, title}'
pccli --json services songs show 1001 --arrangement 2001
```

Errors go to stderr; JSON errors use the same schema version with an `error` object containing a stable `code`, `message`, and relevant candidates or recovery information. A failure before publication emits no success result to stdout. A broken output pipe can still leave partial bytes; check the exit code before using an export.

Planning write errors may include `error.recovery` with known record IDs, parents, and a failed stage. Copy recovery additionally retains `source_plan_id`, `created` receipts, `observed` contradictory IDs, `last_outcome`, `last_error_code`, and accepted-but-unverified fields when applicable. Observed IDs are not claimed as newly created records.

| Error code | Meaning for a write |
| --- | --- |
| `verification_failed` | A request was accepted, but the requested state could not be confirmed |
| `outcome_unknown` | The write may have committed; inspect the destination before retrying |
| `partial_success` | A later copy step was rejected after earlier creations, or result output failed after an accepted/verified write; recovery or `error.result` retains context |
| `cancelled` | Execution was interrupted; copy recovery can still record an unknown last write outcome |


| Exit | Meaning |
| --- | --- |
| `0` | Success |
| `1` | Operational, incomplete, verification, or unknown-outcome failure |
| `2` | Invalid command, arguments, or values |
| `130` | Interrupt or cancellation |
| `143` | Termination signal |

Never treat a write error as permission to replay it automatically. The request may have reached Planning Center even when its response was lost.

## Environment variables

| Variable | Purpose / default |
| --- | --- |
| `PCCLI_CLIENT_ID` | Personal Access Token client ID |
| `PCCLI_CLIENT_SECRET` | Personal Access Token secret |
| `PCCLI_MOCK` | `true` uses local fixtures; default `false` |
| `PCCLI_MOCK_FIXTURE` | Default `testdata/mocks/services-library.json`, relative to the working directory |
| `PCCLI_BASE_URL` | API origin; default `https://api.planningcenteronline.com` |
| `PCCLI_UPLOAD_URL` | Upload origin; default `https://upload.planningcenteronline.com` |
| `PCCLI_TIMEOUT` | Positive Go duration per HTTP request; default `30s` |
| `PCCLI_COLOR` | `auto`, `always`, or `never`; default `auto` |
| `PCCLI_USER_AGENT` | HTTP user agent; default `pccli/dev` |
| `NO_COLOR` | Nonempty disables automatic color; explicit `always` wins |

Endpoint overrides must be HTTP(S) origins without credentials, path, query, or fragment. Precedence is shell environment, local `.env`, `~/.pccli/.env`, then defaults. No legacy environment aliases are supported.
