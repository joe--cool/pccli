---
title: Quick Start
nav_order: 4
---

# Quick Start

pccli is an unofficial Planning Center tool, not built, endorsed, or supported by Planning Center. This guide installs it, configures access, and verifies a basic read.

## 1. Install

```sh
go install github.com/joe--cool/pccli/v2/cmd/pccli@latest
pccli --version
```

You can also use a [release binary](installation.html).

## 2. Authenticate

Create a [Personal Access Token](https://api.planningcenteronline.com/oauth/applications), then save its values in `~/.pccli/.env`:

```sh
mkdir -p ~/.pccli
$EDITOR ~/.pccli/.env
```

```dotenv
PCCLI_CLIENT_ID=your_client_id
PCCLI_CLIENT_SECRET=your_secret
```

See [authentication](authentication.html) for permissions and project-local configuration.

## 3. Verify basic access

```sh
pccli --help
pccli services --help
pccli services songs list --limit 1
```

The last command reads Services using your account permissions. An empty result is valid. To try the interface without API access:

```sh
PCCLI_MOCK=true pccli services songs list
```

## Choose your next task

- [Inspect service plans, orders, times, teams, and scheduled people](planning-a-service.html).
- [Find and maintain songs, arrangements, keys, files, and tags](managing-your-song-library.html).
- [Look up a command](commands.html).
- [Use JSON in a script](commands.html#json-and-errors).

Available workflows cover Services planning inspection, plan label updates, and song-library management.
