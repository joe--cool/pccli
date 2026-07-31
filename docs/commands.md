---
title: Services Music Library Commands
nav_order: 6
---

# Services Music Library Commands

Services commands use this shape:

```sh
pccli services RESOURCE ACTION
```

## Global Flags

| Flag | Description |
| --- | --- |
| `--json` | Write machine-readable JSON. |
| `-h`, `--help` | Show help. |
| `-v`, `--version` | Show the pccli version. |

## Song Arguments

Commands that accept `SONG` can use a Planning Center song ID or a title/search term.

For read-only commands, a search term is accepted when it resolves to one clear song. For write commands, use a song ID or the exact song title shown by search results. Exact title matching in pccli is case-insensitive.

Hidden songs are excluded from song lookup and song ID checks by default. Use `--hidden` on supported commands when you intentionally want to work with hidden songs.

If multiple songs share the same title, pccli will not guess. Use `songs search` to compare the ID, author, and CCLI number, then run the follow-up command with the song ID.

```sh
pccli services songs search "Amazing"
pccli services songs show "Amazing Grace"
pccli services songs show 1001
pccli services songs show "Lamb of God"
```

## Arrangement and Key Arguments

Commands that accept `ARRANGEMENT` or `KEY` can use the Planning Center ID or the exact name. Read-only arrangement and key commands also accept a unique partial name.

When an arrangement is optional, pccli uses the default arrangement: an arrangement named `Default` when one exists, otherwise the first arrangement returned by Planning Center.

## `pccli services songs list`

Search songs in the Services music library.

```sh
pccli services songs list [flags]
```

| Flag | Description |
| --- | --- |
| `-q`, `--search <value>` | Search song titles using a contains match. |
| `--title <value>` | Filter by title. Supports Planning Center wildcard values such as `Amazing%`. |
| `--author <value>` | Filter by author. Supports wildcard values. |
| `--ccli <number>` | Filter by CCLI number. |
| `--hidden` | Show only hidden songs. |
| `--key <value>` | Filter by key, such as `G` or `Cm`. |
| `--meter <value>` | Filter by meter, such as `4/4`. |
| `--themes <value>` | Filter by themes. |
| `--order <value>` | Planning Center order value, such as `title` or `-updated_at`. |
| `--limit <number>` | Number of songs to fetch. Default: `25`. |

## `pccli services songs search`

Search song titles and show matching IDs for follow-up commands.

```sh
pccli services songs search QUERY [flags]
```

| Flag | Description |
| --- | --- |
| `--hidden` | Search hidden songs instead of visible songs. |
| `--limit <number>` | Number of songs to fetch. Default: `10`. |

## `pccli services songs show`

Show details for one song, including its arrangements and the selected arrangement's keys.

```sh
pccli services songs show SONG
pccli services songs show SONG ARRANGEMENT
```

| Flag | Description |
| --- | --- |
| `--hidden` | Show a hidden song and include hidden status in the output. |

## `pccli services songs create`

Create a song in the Services music library.

```sh
pccli services songs create --title "Amazing Grace" --author "John Newton" --ccli 22025
```

If you only know the CCLI number, `--ccli` can be used without `--title`; pccli sends the CCLI number in the Planning Center-supported metadata lookup field.

| Flag | Description |
| --- | --- |
| `--title <value>` | Song title. |
| `--author <value>` | Song author. |
| `--admin <value>` | Song administrator. |
| `--ccli <number>` | CCLI number. |
| `--copyright <value>` | Copyright text. |
| `--hidden` | Create the song as hidden. |
| `--themes <value>` | Themes or tags text. |

## `pccli services songs update`

Update song metadata.

```sh
pccli services songs update SONG --themes "Grace, Hymn"
```

The update command accepts the same metadata flags as `create`.

## `pccli services songs delete`

Delete a song. Use an ID or exact title and pass `--yes` to make deletion explicit.

```sh
pccli services songs delete SONG --yes
```

| Flag | Description |
| --- | --- |
| `--hidden` | Delete a hidden song. |
| `--yes` | Confirm deletion. |

## `pccli services songs arrangements`

List arrangements for one song, or show one arrangement with its keys.

```sh
pccli services songs arrangements SONG [flags]
pccli services songs arrangements SONG ARRANGEMENT
```

| Flag | Description |
| --- | --- |
| `--hidden` | Look up arrangements for a hidden song. |
| `--limit <number>` | Number of arrangements to fetch. Default: `25`. |

### `pccli services songs arrangements create`

Create an arrangement for a song.

```sh
pccli services songs arrangements create SONG --name "Full Band" --key G --bpm 72 --meter 4/4 --length 4:15
```

### `pccli services songs arrangements update`

Update arrangement metadata. Use an arrangement ID or exact arrangement name.

```sh
pccli services songs arrangements update SONG ARRANGEMENT --bpm 74
```

### `pccli services songs arrangements delete`

Delete an arrangement. Use an arrangement ID or exact arrangement name and pass `--yes`.

```sh
pccli services songs arrangements delete SONG ARRANGEMENT --yes
```

Arrangement create and update flags:

| Flag | Description |
| --- | --- |
| `--name <value>` | Arrangement name. Required for create. |
| `--key <value>` | Chord chart key, such as `G` or `Cm`. |
| `--bpm <number>` | Beats per minute. |
| `--meter <value>` | Meter, such as `4/4`. |
| `--length <value>` | Length as seconds or `m:ss`. |
| `--lyrics-enabled` | Enable lyrics for the arrangement. |
| `--notes <value>` | Arrangement notes. |
| `--sequence <values>` | Comma-separated sequence, such as `V1,V2,C`. |
| `--hidden` | Create, update, or delete the arrangement on a hidden song. |

## `pccli services songs keys`

List available keys for an arrangement. If `ARRANGEMENT` is omitted, pccli uses the default arrangement.

```sh
pccli services songs keys SONG [ARRANGEMENT] [flags]
```

| Flag | Description |
| --- | --- |
| `--hidden` | Look up keys for a hidden song. |
| `--limit <number>` | Number of keys to fetch. Default: `25`. |

### `pccli services songs keys create`

Create a key for an arrangement. If `ARRANGEMENT` is omitted, pccli uses the default arrangement.

```sh
pccli services songs keys create SONG --name "Female Lead" --start Bb
pccli services songs keys create SONG ARRANGEMENT --name "Female Lead" --start Bb --end Bb
```

### `pccli services songs keys update`

Update key metadata. Use arrangement and key IDs or exact names.

```sh
pccli services songs keys update SONG ARRANGEMENT KEY --start A --end A
```

### `pccli services songs keys delete`

Delete a key. Use arrangement and key IDs or exact names and pass `--yes`.

```sh
pccli services songs keys delete SONG ARRANGEMENT KEY --yes
```

Key create and update flags:

| Flag | Description |
| --- | --- |
| `--name <value>` | Key name. Required for create. |
| `--start <value>` | Starting key, such as `G` or `Cm`. Required for create. |
| `--end <value>` | Ending key. Defaults to the starting key on create. |
| `--hidden` | Create, update, or delete the key on a hidden song. |

## `pccli services songs attachments`

List song, arrangement, or key attachments.

```sh
pccli services songs attachments SONG [flags]
pccli services songs attachments SONG --arrangement ARRANGEMENT
pccli services songs attachments SONG --arrangement ARRANGEMENT --key KEY
```

| Flag | Description |
| --- | --- |
| `--arrangement <value>` | Scope attachments to an arrangement ID or exact name. |
| `--key <value>` | Scope attachments to a key ID or exact name. Requires `--arrangement`. |
| `--hidden` | Look up attachments for a hidden song. |
| `--filename <value>` | Filter by filename. |
| `--type <value>` | Filter by Planning Center attachment type. |
| `--limit <number>` | Number of attachments to fetch. Default: `25`. |

## `pccli services songs attach`

Attach a file, Planning Center upload UUID, link, or inline content to a song. Add `--arrangement` and `--key` when the attachment belongs to a specific arrangement or key.

```sh
pccli services songs attach SONG --file ./lead-sheet.pdf
pccli services songs attach SONG --url "https://example.com/rehearsal-track"
pccli services songs attach SONG --upload-id us1-16207df7-b6cc-4abe-ca1a-306c6f7e423d
```

| Flag | Description |
| --- | --- |
| `--file <path>` | Upload and attach a local file such as a PDF. |
| `--upload-id <uuid>` | Attach an existing Planning Center upload UUID. |
| `--url <url>` | Attach a remote link. |
| `--content <value>` | Attach inline attachment content. |
| `--filename <value>` | Filename to show in Planning Center. |
| `--attachment-type-ids <ids>` | Comma-separated Planning Center attachment type IDs. |
| `--item-details` | Import this attachment to item details. |
| `--page-order <value>` | Planning Center page order value. |
| `--song-part <value>` | Song part label for generated charts. |
| `--arrangement <value>` | Scope attachment to an arrangement ID or exact name. |
| `--key <value>` | Scope attachment to a key ID or exact name. Requires `--arrangement`. |
| `--hidden` | Attach to a hidden song. |

## Environment Variables

| Variable | Description |
| --- | --- |
| `PCCLI_CLIENT_ID` | Planning Center Personal Access Token client ID. |
| `PCCLI_CLIENT_SECRET` | Planning Center Personal Access Token secret. |
| `PCCLI_MOCK` | Use local mock responses when set to `true`. |
| `PCCLI_MOCK_FIXTURE` | Path to the mock fixture file. |
| `PCCLI_BASE_URL` | Planning Center API base URL. |
| `PCCLI_UPLOAD_URL` | Planning Center file upload base URL. |
| `PCCLI_TIMEOUT` | HTTP request timeout. Default: `30s`. |
| `PCCLI_COLOR` | Color mode: `auto`, `always`, or `never`. |
| `NO_COLOR` | Disable color when `PCCLI_COLOR` is not `always`. |

## Config File Loading

pccli reads configuration from shell environment variables first, then `.env` in the current directory, then `~/.pccli/.env`, then built-in defaults.
