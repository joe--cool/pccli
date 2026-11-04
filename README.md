# Planning Center CLI

[![CI](https://github.com/joe--cool/pccli/actions/workflows/ci.yml/badge.svg)](https://github.com/joe--cool/pccli/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/joe--cool/pccli)](https://github.com/joe--cool/pccli/releases)
[![Go version](https://img.shields.io/github/go-mod/go-version/joe--cool/pccli)](go.mod)

`pccli` is an unofficial Planning Center CLI. It is not built, endorsed, or supported by Planning Center.

Inspect service plans, orders, times, teams, and scheduled people, and maintain plan titles. Find and maintain the Services song library: songs, arrangements, musical keys, files, links, and tag assignments. Human output is concise; every data command supports a versioned `--json` envelope.

## Install and authenticate

[Download a platform archive](https://joe--cool.github.io/pccli/downloads.html), or install with Go:

```sh
go install github.com/joe--cool/pccli/v2/cmd/pccli@latest
pccli --version
mkdir -p ~/.pccli
$EDITOR ~/.pccli/.env
```

Create a [Planning Center Personal Access Token](https://api.planningcenteronline.com/oauth/applications) and put its values in that file:

```dotenv
PCCLI_CLIENT_ID=your_client_id
PCCLI_CLIENT_SECRET=your_secret
```

Tokens use HTTP Basic Auth and act with their owner's permissions. Keep them private. Configuration precedence is shell environment, current-directory `.env`, `~/.pccli/.env`, then defaults. See [authentication](docs/authentication.md) and [.env.example](.env.example).

## Find, inspect, and maintain

```sh
pccli services songs list --search "Amazing"
pccli services songs show 1001
pccli services arrangements list --song 1001
pccli services keys list --song 1001 --arrangement 2001
pccli services attachments list --song 1001 --arrangement 2001
pccli services songs update 1001 --author "John Newton"
pccli services arrangements update 2001 --song 1001 --chart-key G --notes "Verse, chorus, repeat"
```

Use IDs or exact names for writes and parent flags. Reads may resolve a unique partial target; ambiguous matches always require an ID. Hidden songs require `--visibility hidden` or `all`; changing hidden state uses `--hidden=true|false`. Arrangements and keys always have explicit parents.

```sh
pccli --json services songs list --all
PCCLI_MOCK=true pccli services songs list
```

[Documentation](https://joe--cool.github.io/pccli/) · [Quick start](docs/quick-start.md) · [Service planning](docs/planning-a-service.md) · [Song-library workflows](docs/managing-your-song-library.md) · [Command reference](docs/commands.md) · [Metadata fields](docs/song-fields.md)

## Develop

Use the versions pinned in `.mise.toml` (Go 1.27.3):

```sh
mise install
go test ./...
go build -o bin/pccli ./cmd/pccli
```

Read [DEVELOPER.md](DEVELOPER.md) and [AGENTS.md](AGENTS.md) before changing behavior. Public documentation lives in `docs/`.
