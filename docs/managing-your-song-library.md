---
title: Managing Your Song Library
nav_order: 6
---

# Managing Your Song Library

This guide is for worship planners and operators who maintain songs in Planning Center Services. It starts where people usually start in the web app: find the song, confirm it is the right one, then work with arrangements, keys, and files.

Use IDs when scripting. Use titles when working manually.

## 1. Find the Song

Start with search when you know part of a title.

```sh
pccli services songs search "Amazing"
```

The result includes IDs because Planning Center APIs identify songs by ID. Keep that ID when you are building a script or when several songs have similar titles.

pccli searches visible songs by default. If you need to inspect songs hidden in Planning Center, use `--hidden`.

```sh
pccli services songs search "Amazing" --hidden
```

If two songs have the same title, compare the author and CCLI number in the search results and use the ID for the follow-up command. pccli will not choose between duplicate exact titles for you.

pccli treats exact title checks as case-insensitive. Read-only commands can also use a unique portion of a title. For example, both commands can resolve the same song when only one match exists:

```sh
pccli services songs show "Jesus, Lamb of God"
pccli services songs show "Lamb of God"
```

For automation, use structured filters and JSON.

```sh
pccli --json services songs list --title "Amazing%" --author "Newton"
pccli --json services songs list --ccli 22025
```

## 2. Confirm the Song

Open the song by title, unique partial title, or ID. This is the best first stop after search because it shows the song details, available arrangements, and the keys for the selected arrangement.

```sh
pccli services songs show "Amazing Grace"
pccli services songs show 1001
pccli services songs show "Amazing Grace" "Acoustic"
```

If you do not name an arrangement, pccli uses the default arrangement. It prefers an arrangement named `Default`; if Planning Center returns no arrangement with that name, pccli uses the first arrangement returned by Planning Center.

Hidden status is only shown when you use `--hidden`.

If more than one song matches, pccli stops and shows the matching IDs so you can choose intentionally.

## 3. Review Arrangements

Arrangements are the practical versions of a song that teams use: full band, acoustic, youth band, special event versions, and similar variants.

```sh
pccli services songs arrangements "Amazing Grace"
```

Use the arrangement ID or exact arrangement name for key and attachment work. Read-only arrangement commands can also use a unique partial arrangement name.

```sh
pccli services songs arrangements "Amazing Grace" "Full Band"
pccli services songs keys "Amazing Grace" "Full Band"
pccli services songs attachments "Amazing Grace" --arrangement "Full Band"
```

The arrangement command is still useful even though `songs show` includes arrangement context: use `show` to orient yourself, and use `arrangements` when you want a focused list or one arrangement's details for a script or operator task.

## 4. Review Key-Specific Materials

Some files belong to a specific key rather than the general song or arrangement. Use the key ID or exact key name from `songs keys`.

```sh
pccli services songs keys "Amazing Grace"
pccli services songs attachments "Amazing Grace" --arrangement "Full Band" --key "Default"
```

This is useful for key-specific chord charts, number charts, lead sheets, or PDFs.

## 5. Add, Update, or Delete Metadata

Create songs with the metadata operators usually know first.

```sh
pccli services songs create --title "Amazing Grace" --author "John Newton" --ccli 22025
```

If you only know the CCLI number, you can create with `--ccli`; Planning Center supports using that field to fetch song metadata.

```sh
pccli services songs create --ccli 22025
```

For updates, use an ID or exact title. pccli intentionally does not allow broad search terms for write commands.

```sh
pccli services songs update "Amazing Grace" --themes "Grace, Hymn"
pccli services songs update 1001 --hidden
```

Create and maintain arrangements from the same song-centered workflow.

```sh
pccli services songs arrangements create 1001 --name "Full Band" --key G --bpm 72 --meter 4/4 --length 4:15
pccli services songs arrangements update 1001 "Full Band" --bpm 74
pccli services songs arrangements delete 1001 "Full Band" --yes
```

Create and maintain keys under the arrangement. If you omit the arrangement on `keys create`, pccli uses the default arrangement.

```sh
pccli services songs keys create 1001 --name "Female Lead" --start Bb
pccli services songs keys update 1001 "Full Band" "Female Lead" --start A --end A
pccli services songs keys delete 1001 "Full Band" "Female Lead" --yes
```

Deletes require `--yes` so scripts are explicit and accidental terminal history edits are less risky.

## 6. Attach Files and Links

Attach local files, such as PDFs, directly from disk.

```sh
pccli services songs attach "Amazing Grace" --file ./lead-sheet.pdf
```

Attach a file to an arrangement or key when the material is not meant for the whole song.

```sh
pccli services songs attach "Amazing Grace" --arrangement "Full Band" --file ./full-band.pdf
pccli services songs attach "Amazing Grace" --arrangement "Full Band" --key "Default" --file ./full-band-g.pdf
```

Attach links when the material lives outside Planning Center.

```sh
pccli services songs attach "Amazing Grace" --url "https://example.com/rehearsal-track"
```

For advanced upload workflows, use an existing Planning Center upload UUID.

```sh
pccli services songs attach "Amazing Grace" --upload-id us1-16207df7-b6cc-4abe-ca1a-306c6f7e423d
```

## 7. Script the Same Workflow

In scripts, prefer IDs and JSON output.

```sh
song_id="$(pccli --json services songs list --ccli 22025 | jq -r '.[0].id')"
pccli --json services songs arrangements "$song_id"
pccli services songs attach "$song_id" --file ./lead-sheet.pdf
```

This keeps automation deterministic while preserving the title-based workflow for interactive use.

## Command Reference

For every flag and command form, see the [Services music-library command reference](commands.html).
