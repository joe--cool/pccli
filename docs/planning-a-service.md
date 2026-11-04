---
title: Planning a Service
nav_order: 6
---

# Planning a Service

pccli is unofficial and is not built, endorsed, or supported by Planning Center.

Find a service type, choose a plan, then inspect its order, times, and people. Every planning command supports `--json`; parent flags always identify the intended service type and plan.

## Find the plan

```sh
pccli services service-types list
pccli services plans list --service-type "Sunday Worship" --filter future
pccli services plans show 4001 --service-type 3001
```

Use IDs from your account. The mock fixture uses service type **3001 / Sunday Worship** and plan **4001 / Gathered in Grace**. Names in parent flags must match exactly, ignoring case. Duplicate names or plan titles require IDs. Untitled plans remain selectable by ID; dates are displayed for orientation and are not selectors.

`plans list --filter future`, `past`, or `no_dates` selects Planning Center's corresponding collection. Without a filter, pccli uses the collection returned by Planning Center. `--all` follows every returned page; `--limit N` limits display and reports whether more results remain. Pagination is not a snapshot.

Known archived service types and teams are excluded by default, including direct IDs and parent lookups. Add `--include-archived` to select them explicitly. This option filters the returned records locally; it cannot recover archived records that Planning Center omits. Missing archive state is treated as unavailable, and is not proof that a record is active. Plans have no archive selector of their own.

## Read the order and schedule

```sh
pccli services items list --service-type 3001 --plan 4001 --all
pccli services items show 5002 --service-type 3001 --plan 4001
pccli services plan-times list --service-type 3001 --plan 4001 --all
pccli services teams list --service-type 3001
pccli services teams show 6001 --service-type 3001
pccli services scheduled-people list --service-type 3001 --plan 4001 --all
```

Items retain the order returned by Planning Center and expose sequence, pre/during/post service position, type, song ID when available, and length in seconds. Times show their service/rehearsal/other classification and provider timestamps. Teams describe the service type's available teams; scheduled people describe assignments in the selected plan. Assignment IDs identify the scheduling records; person and team IDs identify their relationships when available. Human output translates `C`, `U`, and `D` to Confirmed, Unconfirmed, and Declined; JSON retains the provider's raw status. Other values remain visible as returned.

The command shows a selected subset of planning fields. Unavailable text, counts, boolean values, and relationship IDs appear as `null` in JSON; explicit zero and false remain zero and false. Null and absent optional text both appear as `null`. Human tables leave unavailable cells blank and label blank or unavailable titles `(untitled)`. Item song, arrangement, and key relationship IDs are included when provided, without expanding the song library.

To export the complete returned order:

```sh
set -o pipefail
pccli --json services items list --service-type 3001 --plan 4001 --all \
  | jq '.data[] | {id, sequence, service_position, title, length}'
```

Check the pipeline status before using the export. A failed page or cancellation produces no success result; a broken output pipe can leave partial bytes and exits nonzero. See the [JSON contract](commands.html#json-and-errors).

## Build the order of service

Create a header, a timed item, or a song with an explicitly chosen arrangement and optional key:

```sh
pccli services items create --service-type 3001 --plan 4001 --type header --title "Gathering" --position 0
pccli services items create --service-type 3001 --plan 4001 --title "Welcome and Prayer" --length 3:00 --service-position during
pccli services items create --service-type 3001 --plan 4001 --song 1001 --arrangement 2001 --key 3001
```

`--type` accepts `item` or `header`; omit it when linking a song. Song linkage requires both `--song` and `--arrangement`, and `--key` additionally selects a key record within that arrangement. Parent names must match exactly; IDs avoid duplicate-name ambiguity. Song visibility and archived arrangement selection use the existing `--visibility` and `--include-archived` rules. pccli never chooses a first arrangement for you.

Item fields include `--title`, `--description`, `--html-details`, `--length` (seconds or `m:ss`), `--service-position pre|during|post`, and `--chart-sequence` (comma-separated arrangement section labels). Explicit empty text, zero length, and empty chart sequence are transmitted; omitted fields remain unchanged. Length and initial `--position` must be nonnegative. Position is available only during creation and maps to the initial sequence number.

```sh
pccli services items update 5003 --service-type 3001 --plan 4001 --description "" --length 0
pccli services items update 5002 --service-type 3001 --plan 4001 --chart-sequence "Verse 1,Chorus 1"
pccli services items update 5002 --service-type 3001 --plan 4001 --chart-sequence ""
pccli services items reorder --service-type 3001 --plan 4001 --sequence 5001,5003,5002
pccli services items delete 5003 --service-type 3001 --plan 4001 --yes
```

Reordering requires **every item ID in the selected plan exactly once**. Inspect the complete list first; omissions, duplicate IDs, foreign IDs, malformed pages, or a changed preflight collection stop the request. pccli rechecks the complete collection before sending the order, then reads all items again to verify the returned order and any known sequence values. This is not a remote transaction; concurrent edits can still cause a verification error.

Updates can replace song linkage using the same explicit parent flags. Empty song selectors and song linkage on a header are rejected; removing an existing song relationship, changing an existing item's type directly, media/layout editing, and copying item attachments are not supported. HTML details are retained in JSON and terminal-sanitized for human display. Every item write performs independent readback; late failures retain the known item ID and plan in recovery information. Deletes require `--yes` and verify HTTP 404 afterward. Static mock mode supports reading, not this editing workflow.

## Check plan readiness

{% include terminal-demo.html name="planning-demo" title="Check Sunday's plan readiness" caption="Review Gathered in Grace, confirm its service time, inspect available teams, then read assignment responses. Under 90 seconds; mock data." %}

<details markdown="1">
<summary>Read the readiness steps</summary>

1. `pccli services plans show 4001 --service-type 3001` shows **Gathered in Grace**, two scheduled people, and one needed position.
2. `pccli services plan-times list --service-type 3001 --plan 4001` shows **Morning Service**, from **09:00 to 10:15 UTC** on November 8, 2026. Timestamps are displayed as returned.
3. `pccli services teams list --service-type 3001` shows the available **Worship Team (6001)**. Availability is distinct from a plan assignment.
4. `pccli services scheduled-people list --service-type 3001 --plan 4001` shows **Alex Sample / Vocals / Confirmed** and **Jordan Example / Keys / Unconfirmed**, with assignment, person, and team IDs.

The recording ends on the response summary. Use the written commands above for exports and additional detail; the recording covers one selected plan.

</details>

## Create and maintain plans

Create an empty, undated plan in an explicit service type:

```sh
pccli services plans create --service-type 3001 --title "Advent Sunday"
pccli services plans create --service-type 3001
```

Labels are optional. Supplying no labels creates an untitled plan; `--title ""` explicitly requests an empty title. Ordinary creation does not copy items, people, notes, or times. Keep the returned plan ID for follow-up commands. Public access can be set explicitly with `--public=true|false`.

Update labels or settings:

```sh
pccli services plans update 4001 --service-type 3001 --title "Gathered in Grace"
pccli services plans update 4001 --service-type 3001 --series-title "Life Together"
pccli services plans update 4001 --service-type 3001 --series-title ""
pccli services plans update 4001 --service-type 3001 --public=false --reminders-disabled=true
```

Use `--series-id ID` to assign an existing Series by positive decimal ID; pccli validates it before writing and checks the returned series relationship afterward. `--series-id` conflicts with `--series-title`. Public access and reminder settings accept explicit true or false.

Omitted fields remain unchanged. An explicit empty value requests clearing that field. The API must return the same value during readback for pccli to report success. A successful request followed by a failed readback reports a verification error; inspect the plan before retrying. pccli never automatically replays updates. If creation succeeds but later verification fails, the error includes a recovery record with the returned plan ID, service type, stage, and created receipt. Keep that identity and inspect it before creating another plan. JSON errors expose this as `error.recovery`; human errors print the same recovery information.

Delete only the intended plan after checking its ID:

```sh
pccli services plans delete 4001 --service-type 3001 --yes
```

Deletion requires `--yes`, validates the exact identity, and verifies that the plan returns HTTP 404 afterward. A verification failure means the request was accepted but the final state could not be confirmed; inspect the resource before retrying.

Plan copying, time editing, scheduling changes, and notifications remain outside this interface. Mock mode supports the read journeys above; its static fixture does not emulate updates or prove live account permissions.
