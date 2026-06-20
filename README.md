# Planning Center CLI

[![CI](https://github.com/joe--cool/pccli/actions/workflows/ci.yml/badge.svg)](https://github.com/joe--cool/pccli/actions/workflows/ci.yml)
[![Docs](https://github.com/joe--cool/pccli/actions/workflows/pages.yml/badge.svg)](https://github.com/joe--cool/pccli/actions/workflows/pages.yml)
[![Release](https://github.com/joe--cool/pccli/actions/workflows/release.yml/badge.svg)](https://github.com/joe--cool/pccli/actions/workflows/release.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/joe--cool/pccli)](go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/joe--cool/pccli)](https://goreportcard.com/report/github.com/joe--cool/pccli)
[![GitHub release](https://img.shields.io/github/v/release/joe--cool/pccli?include_prereleases)](https://github.com/joe--cool/pccli/releases)
[![License](https://img.shields.io/github/license/joe--cool/pccli)](LICENSE)
[![Conventional Commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-yellow.svg)](https://www.conventionalcommits.org/)

`pccli` is an unofficial command line tool for Planning Center. It is not built, endorsed, or supported by Planning Center.

Documentation: https://joe--cool.github.io/pccli/

## Install

Install with Go:

```sh
go install github.com/joe--cool/pccli/cmd/pccli@latest
```

## Configure

Create a Planning Center Personal Access Token, then copy `.env.example` to `.env` and fill in:

```sh
PCCLI_CLIENT_ID=your_client_id
PCCLI_CLIENT_SECRET=your_secret
```

`.env` is ignored by git. The API uses HTTP Basic Auth for Personal Access Tokens.

For user-level config outside a project directory, use `~/.pccli/.env`. pccli loads shell environment variables first, then local `.env`, then `~/.pccli/.env`.

## Use

```sh
pccli services songs list --title "Amazing%"
pccli services songs show 1001
pccli services songs arrangements 1001
```

Add `--json` to any command for automation:

```sh
pccli --json services songs list --author "Newton"
```

Run with mock data for demos or screenshots:

```sh
PCCLI_MOCK=true go run ./cmd/pccli services songs list
```

## Develop

This repo is Go-first because single binary distribution is a good fit for church staff and volunteers who may not be developers. Use `mise install` when available, or install the Go version from `.mise.toml`.

```sh
go test ./...
go build -o bin/pccli ./cmd/pccli
```

See [DEVELOPER.md](DEVELOPER.md) and [AGENTS.md](AGENTS.md) before adding a new feature. User documentation is published from `docs/`.
