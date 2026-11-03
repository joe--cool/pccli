---
title: Managing Your Song Library
nav_order: 6
---

# Managing Your Song Library

pccli is an unofficial Planning Center tool, not built, endorsed, or supported by Planning Center. Start by finding a song, confirm its identity, then select the version and destination you mean to maintain. Example IDs below are placeholders; use IDs from your results.

## Find and inspect

```sh
pccli services songs list --search "Amazing"
pccli services songs list --title "Amazing Grace"
pccli services songs show 1001
pccli services songs show 1001 --arrangement 2001
```

Search text is literal. Use `--visibility hidden` to inspect hidden songs, or `--visibility all` to include both states. Exact names are case-insensitive. A unique partial target works for reads; duplicate matches require an ID. Song show gives an overview and expands keys only when you select an arrangement.

## Choose an arrangement and key

```sh
pccli services arrangements list --song 1001
pccli services arrangements show 2001 --song 1001
pccli services keys list --song 1001 --arrangement 2001
pccli services keys show 3001 --song 1001 --arrangement 2001
pccli services arrangements sections 2001 --song 1001
```

Parent flags accept IDs or exact names. An arrangement is always explicit for key operations; pccli never chooses the first or a named default arrangement for you. The sections command helps inspect full labels before changing sequence.

## Maintain metadata and charts

```sh
pccli services songs create --title "Amazing Grace" --author "John Newton" --ccli 22025
pccli services songs update 1001 --themes "Grace, Hymn"
pccli services arrangements create --song 1001 --name "Full Band" --chart-key G --bpm 72 --meter 4/4 --length 4:15
pccli services arrangements update 2001 --song 1001 --notes "Verse, chorus, repeat" --sequence 'Verse 1,Chorus 1,Verse 2'
pccli services keys create --song 1001 --arrangement 2001 --name "Female Lead" --start Bb
pccli services keys update 3001 --song 1001 --arrangement 2001 --start A --end A
```

Updates change only supplied fields. For example, `--notes ""` requests a clear and `--lyrics-enabled=false` turns lyrics off. Invalid empty or numeric values fail clearly. [All metadata fields](song-fields.html) include chord-chart text, print settings, alternate keys, and provider imports.

To hide or restore a song, select its current visibility independently from its new state:

```sh
pccli services songs update 1001 --hidden=true
pccli services songs update 1001 --visibility hidden --hidden=false
```

## Archive a version without deleting it

```sh
pccli services arrangements archive 2001 --song 1001
pccli services arrangements list --song 1001 --include-archived
pccli services arrangements unarchive 2001 --song 1001 --include-archived
```

Ordinary arrangement lookups exclude archived versions. Follow-up operations on an archived arrangement require `--include-archived` explicitly.

## Place files and links in the right scope

A file can belong to a song, arrangement, or key. List the association where you intend to place it:

```sh
pccli services attachments list --song 1001
pccli services attachments list --song 1001 --arrangement 2001
pccli services attachments list --song 1001 --arrangement 2001 --key 3001
pccli services attachments add --song 1001 --arrangement 2001 --file ./full-band.pdf
pccli services attachments add --song 1001 --arrangement 2001 --key 3001 --url https://example.com/chart.pdf --filename chart.pdf
pccli services attachments show 4001 --song 1001 --arrangement 2001
pccli services attachments update 4001 --song 1001 --arrangement 2001 --filename full-band-revised.pdf
```

Choose one add source: local `--file`, existing `--upload-id`, `--url`, or inline `--content`. Updates can change metadata or replace a source with `--upload-id`, `--url`, or `--content`. Read the outcome: `verified` confirms readable requested values; `accepted` lists fields that cannot be confirmed through API readback.

If an upload succeeds but attachment creation fails, keep the recovery receipt. Inspect the destination before retrying an unknown outcome. Reuse the receipt with `--upload-id` only after confirming it was not attached. [Attachment reference](commands.html#attachments).

## Classify with tags

```sh
pccli services tag-groups list --all
pccli services tags list --all
pccli services tags list --song 1001
pccli services tags replace --song 1001 --tags 5001,5002
pccli services tags replace --song 1001 --arrangement 2001 --clear
pccli services attachment-types list --all
```

Tag replacement sends the complete intended set. It removes assignments you leave out. Required and single-selection group rules still apply. Tags classify songs or arrangements; `--themes` remains separate text metadata. Tag groups and attachment types are read-only discovery commands.

## Inspect scheduling history

```sh
pccli services songs schedules 1001 --all
pccli services songs last-scheduled 1001 --service-type 789
```

These read the song's scheduling associations. They do not edit plans or create a scheduling workflow.

## Delete intentionally

```sh
pccli services attachments delete 4001 --song 1001 --arrangement 2001 --yes
pccli services keys delete 3001 --song 1001 --arrangement 2001 --yes
pccli services arrangements delete 2001 --song 1001 --yes
pccli services songs delete 1001 --yes
```

Each example is independent. Deletes require `--yes`, IDs or exact names, and explicit parent context. Confirm the selected scope before running them.

## Export for a script

```sh
pccli --json services songs list --all | jq '.data[] | {id, title, ccli_number}'
pccli --json services arrangements list --song 1001 --all
```

The envelope is `{"schema":"pccli/v1","data":...,"meta":...}`. Check the exit code and completeness metadata before processing an export. Use IDs for subsequent commands, and stop for ambiguity or an incomplete lookup. [JSON and errors](commands.html#json-and-errors).
