---
title: Command Reference
nav_order: 5
---

# Command Reference

## Global Flags

| Flag | Description |
| --- | --- |
| `--json` | Write machine-readable JSON. |
| `-h`, `--help` | Show help. |
| `-v`, `--version` | Show the pccli version. |

## `pccli songs list`

List songs in the Services music library.

```sh
pccli songs list [flags]
```

| Flag | Description |
| --- | --- |
| `--title <value>` | Filter by title. Supports Planning Center wildcard values such as `Amazing%`. |
| `--author <value>` | Filter by author. Supports wildcard values. |
| `--ccli <number>` | Filter by CCLI number. |
| `--limit <number>` | Number of songs to fetch. Default: `25`. |

## `pccli songs show`

Show details for one song.

```sh
pccli songs show SONG_ID
```

## `pccli songs arrangements`

List arrangements for one song.

```sh
pccli songs arrangements SONG_ID [flags]
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
