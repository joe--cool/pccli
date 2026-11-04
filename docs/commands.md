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
| `--include-archived` | Explicitly include archived arrangements, or returned archived service types/teams in planning commands |
| `--limit N` | Total results to display, default 25; positive integer |
| `--all` | Return the complete selected collection; conflicts with an explicit `--limit` |
| `--yes` | Required for every delete; never bypasses target validation |

### IDs, names, and parents

Targets accept a positive decimal ID or name. Use `id:1001` to force an ID and `name:1001` to force an exact numeric name. `name:Amazing Grace` forces an exact name. Exact name matching is case-insensitive. Read targets prefer an exact match, then accept a unique literal partial match. Writes and parent flags require IDs or exact names. Duplicate exact names are ambiguous; use the displayed IDs to choose.

Search uses literal text, not API wildcard syntax. `%` and `_` are ordinary characters. `songs list --title TEXT` is an exact case-insensitive filter; `--search TEXT` is a literal contains filter. They conflict with each other.

Arrangements require `--song SONG`. Keys require `--song SONG --arrangement ARRANGEMENT`. There is no automatic first/default arrangement. Attachment scope is a song, optionally an arrangement, optionally a key; `--key` requires `--arrangement`. Parent flags use the same visibility and ambiguity rules across follow-up commands.

Lookups traverse all candidate pages before choosing a target. Display limits never limit identity resolution. Page failures, malformed records, or ambiguous matches stop the operation.

## Service planning

```text
pccli services service-types list [--limit N | --all] [--include-archived]
pccli services service-types show SERVICE_TYPE [--include-archived]
pccli services plans list --service-type SERVICE_TYPE [--filter future|past|no_dates] [--limit N | --all]
pccli services plans show PLAN --service-type SERVICE_TYPE
pccli services plans update PLAN --service-type SERVICE_TYPE [--title TEXT] [--series-title TEXT]
pccli services items list --service-type SERVICE_TYPE --plan PLAN [--limit N | --all]
pccli services items show ITEM --service-type SERVICE_TYPE --plan PLAN
pccli services plan-times list --service-type SERVICE_TYPE --plan PLAN [--limit N | --all]
pccli services teams list --service-type SERVICE_TYPE [--limit N | --all] [--include-archived]
pccli services teams show TEAM --service-type SERVICE_TYPE [--include-archived]
pccli services scheduled-people list --service-type SERVICE_TYPE --plan PLAN [--limit N | --all]
```

Every descendant accepts `--include-archived` when selecting an archived service-type parent. Planning Center must return the record for local archive inclusion to work. Planning commands do not use song `--visibility`. Plans may be untitled; use IDs to select them. Read targets support unique literal partial names/titles, while parents and updates require exact names/titles or IDs. Dates are not selectors.

Plan updates require at least one label flag; explicit empty values clear fields. Updates are checked with GET readback. Items retain provider order and expose sequence and service position. Scheduled-person records identify assignments, with related person/team IDs where provided. Missing optional values remain `null` in machine output. Plan creation, copying, deletion, item/time editing, scheduling changes, and notifications are not supported. [Follow the planning workflow](planning-a-service.html).

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
