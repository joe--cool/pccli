---
title: Planning a Service
nav_order: 6
---

# Planning a Service

pccli is unofficial and is not built, endorsed, or supported by Planning Center.

Find a service type, choose a plan, then inspect its order, times, and people. Every planning command supports `--json`; parent flags always identify the intended service type and plan.

Choose a task: [inspect a plan](#find-the-plan), [create one](#create-and-maintain-plans), [copy selected parts](#copy-selected-parts-of-a-plan), [build the order](#build-the-order-of-service), [set times](#set-service-and-rehearsal-times), or [schedule people](#schedule-a-whole-plan-team).

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

Notification sending, automatic scheduling, and split-time scheduling remain outside this interface. Mock mode supports the read journeys above; its static fixture does not emulate updates or prove live account permissions.

## Copy selected parts of a plan

Create a new plan in the **same service type**, choosing exactly which components to carry forward:

```sh
pccli services plans copy 4001 --service-type 3001 --title "Next Sunday" --copy-items --copy-times --time-offset 168h
pccli services plans copy 4001 --service-type 3001 --copy-people --copy-needed-positions
```

Select at least one component:

| Flag | Copied behavior |
| --- | --- |
| `--copy-items` | Supported items and headers, selected song/arrangement/key links, and provider item order |
| `--copy-times` | Service, rehearsal, and other times with their names and classification |
| `--copy-people` | Whole-plan assignments and notes, reset to Unconfirmed without preparing notifications |
| `--copy-needed-positions` | Whole-plan unfilled quantities; separate from assigned people |
| `--copy-settings` | Known public and reminder settings |

The available title and Series selection carry forward. When no Series ID is returned, the available series label is used. `--title ""` explicitly requests an empty title. Without `--copy-settings`, the new plan's `public` field is set to false. Service-type scheduled publication rules still apply independently; this flag does not guarantee private access.

When copying times, choose exactly one of `--time-offset DURATION` or `--preserve-dates`. Durations such as `168h`, `-24h`, or `0` shift every start and end by the same elapsed time. They do not mean a calendar week or preserve local wall-clock time across daylight-saving changes. `--preserve-dates` retains the original dates. Time flags require `--copy-times`.

Item copying requires a known supported item type. Every copied song must have known song, arrangement, **and selected key IDs**; an unavailable or null key stops the copy before creating a destination. Ordinary item creation still permits an optional key. Media items and nonempty layout/background/media relationships are unsupported. Files, attachments, item leaders, notes outside assignment notes, and other unlisted components are not cloned. Copying people requires current active team membership and a uniquely named position in a whole-plan team; availability and scheduling permission remain provider decisions. Decline history is not copied.

The command validates the complete selected source and referenced records before creating the destination, then rechecks them. These requests cannot form a remote transaction; concurrent changes can still occur between requests. Each created record is independently read back, and the final plan is refreshed after its children are created. Needed-position copies report `outcome: accepted`, with an unverified entry for each created record and its requested position ID, because the API does not expose that ID for confirmation.

On a later failure, `error.recovery` retains the source and destination context, created IDs, failing stage, and last-step outcome. Contradictory returned IDs appear as observed records rather than being claimed as newly created. A rejected step after earlier creations reports partial success; lost write responses report unknown commitment. No rollback or automatic replay occurs. Inspect the reported records before retrying; repeating the whole command can create duplicates.

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

## Set service and rehearsal times

An undated plan gains its date from its times. Create one explicit time at a time:

```sh
pccli services plan-times create --service-type 3001 --plan 4001 --name "Morning Service" --starts-at "2026-11-08T09:00:00-08:00" --ends-at "2026-11-08T10:15:00-08:00" --time-type service
pccli services plan-times show 7001 --service-type 3001 --plan 4001
pccli services plan-times update 7001 --service-type 3001 --plan 4001 --starts-at "2026-11-08T09:30:00-08:00"
pccli services plan-times delete 7001 --service-type 3001 --plan 4001 --yes
```

Creation requires `--starts-at` and `--time-type service|rehearsal|other`. `--ends-at` and `--name` are optional; no end or duration is invented when omitted. Timestamps require RFC3339 with an explicit offset or `Z`, years 0001–9999, and at most nine fractional second digits. Empty timestamps are rejected; `--name ""` clears the label. An end must not precede its start.

Updates preserve omitted fields. Changing only a start or end validates the resulting pair against the current known counterpart before writing. If an existing start is unavailable, provide it when setting an end. Supplying no name on a time is valid; use its ID to select an unnamed time. Names used for updates or parent flags must match exactly and unambiguously.

Readback compares timestamp instants at nanosecond precision, so equivalent `Z` and numeric-offset representations can verify successfully. JSON and human output retain the provider's returned timestamp text. Changes affect only the selected time; bulk changes to later plans, scheduling people, assigning teams or positions to times, and reminder rules are outside these commands. Every write uses independent readback and the same recovery information as item writes. Deletes require `--yes` and verify HTTP 404 afterward. The videos continue to demonstrate reads with static mock data.

## Schedule a whole-plan team

Discover members and positions before creating an assignment:

```sh
pccli services team-people list --service-type 3001 --team 6001 --all
pccli services team-positions list --service-type 3001 --team 6001 --all
pccli services scheduled-people create --service-type 3001 --plan 4001 --team 6001 --person 9001 --position 10001
pccli services scheduled-people show 8001 --service-type 3001 --plan 4001
pccli services scheduled-people update 8001 --service-type 3001 --plan 4001 --status confirmed --notes "Bring your folder"
pccli services scheduled-people delete 8001 --service-type 3001 --plan 4001 --yes
```

Team people are **membership candidates**, not a promise of availability, position eligibility, background-check clearance, or permission to schedule. Planning Center makes those final decisions. IDs avoid duplicate-name ambiguity; write selectors require exact names or IDs and traverse complete candidate collections.

These mutations support teams whose returned `schedule_to` is explicitly `plan`. Split-time teams, unavailable scheduling mode, and assignments known to permit partial acceptance are rejected before writing. The position must belong to the team. Assignments store the position's name, so duplicate or case-colliding names within that team are rejected even when a position ID is supplied.

Creation defaults to **Unconfirmed** and explicitly sends `prepare_notification: false`. `--status` accepts `unconfirmed`, `confirmed`, `declined`, or `U`, `C`, `D`; readback accepts equivalent provider spellings while JSON retains the returned value. Optional `--notes` and `--decline-reason` preserve explicit empty strings. No command prepares or sends a notification on your behalf. Updates omit notification-preparation fields and preserve any existing state; previously prepared notifications remain under Planning Center's existing behavior.

Updates can change status, notes, decline reason, or `--position`. Optional `--team` guards the existing assignment's exact team; it does not move the assignment. Changing the person or team requires deleting and creating an assignment explicitly. Creation and updates require current team membership and known nonarchived person state. `team-people` reads exclude known archived people by default, while `--include-archived` includes returned archived records for inspection. Unknown archive state remains visible as unavailable and is not proof of activity; it blocks scheduling changes. `--include-archived` never authorizes scheduling an archived person.

Deleting an exact scoped assignment does not require the person to remain active or in the team, so departed-member assignments can be cleaned up. Team scope, whole-plan mode, and `--yes` still apply. Every mutation performs independent readback or verifies HTTP 404 after deletion; failures retain known record IDs and parent context.

## Track unfilled positions

```sh
pccli services needed-positions list --service-type 3001 --plan 4001 --all
pccli services needed-positions create --service-type 3001 --plan 4001 --team 6001 --position 10001 --quantity 1
pccli services needed-positions show 11001 --service-type 3001 --plan 4001
pccli services needed-positions update 11001 --service-type 3001 --plan 4001 --quantity 0
pccli services needed-positions delete 11001 --service-type 3001 --plan 4001 --yes
```

Quantity is required and nonnegative; zero is transmitted explicitly. These commands support whole-plan teams and reject records with explicit time or time-preference relationships. The provider's `scheduled_to` string is retained without assigning it an undocumented meaning. Missing optional relationships remain unavailable; they do not establish that a record has no time context.

Creation sends the selected position ID, but the API does not expose that ID for independent readback. A successful create therefore reports **`outcome: accepted`**, **`unverified_fields: ["team_position_id"]`**, and the separate **`requested_position_id`**. Readback still verifies quantity, team identity, and the readable position name. Inspect the record in Planning Center when exact position identity matters. Quantity updates report `verified` after readback; the model never invents a canonical position ID from the requested value.

Split-time scheduling, automatic scheduling, invitation sending, and time-specific needed positions are outside these commands. Static mock fixtures demonstrate the reads; stateful request tests establish the wire and recovery behavior, not live account eligibility or delivery.
