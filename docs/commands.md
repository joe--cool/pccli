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

## `pccli services songs list`

List songs in the Services music library.

```sh
pccli services songs list [flags]
```

| Flag | Description |
| --- | --- |
| `--title <value>` | Filter by title. Supports Planning Center wildcard values such as `Amazing%`. |
| `--author <value>` | Filter by author. Supports wildcard values. |
| `--ccli <number>` | Filter by CCLI number. |
| `--limit <number>` | Number of songs to fetch. Default: `25`. |

## `pccli services songs show`

Show details for one song.

```sh
pccli services songs show SONG_ID
```

## `pccli services songs arrangements`

List arrangements for one song.

```sh
pccli services songs arrangements SONG_ID [flags]
```

| Flag | Description |
| --- | --- |
| `--limit <number>` | Number of arrangements to fetch. Default: `25`. |

## Environment Variables

| Variable | Description |
| --- | --- |
| `PCCLI_CLIENT_ID` | Planning Center Personal Access Token client ID. |
| `PCCLI_CLIENT_SECRET` | Planning Center Personal Access Token secret. |
| `PCCLI_MOCK` | Use local mock responses when set to `true`. |
| `PCCLI_MOCK_FIXTURE` | Path to the mock fixture file. |
| `PCCLI_BASE_URL` | Planning Center API base URL. |
| `PCCLI_TIMEOUT` | HTTP request timeout. Default: `30s`. |
| `PCCLI_COLOR` | Color mode: `auto`, `always`, or `never`. |
| `NO_COLOR` | Disable color when `PCCLI_COLOR` is not `always`. |

## Config File Loading

pccli reads configuration from shell environment variables first, then `.env` in the current directory, then `~/.pccli/.env`, then built-in defaults.
