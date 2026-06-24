---
title: Quick Start
nav_order: 4
---

# Quick Start

This guide gets pccli installed, authenticated, and verified with a basic Planning Center command.

## 1. Install

```sh
go install github.com/joe--cool/pccli/cmd/pccli@latest
```

## 2. Set Up Access

Create a `.env` file with your Planning Center Personal Access Token:

```sh
PCCLI_CLIENT_ID=your_client_id
PCCLI_CLIENT_SECRET=your_secret
```

The token's Planning Center user must have access to the product data you want to use. For Services song-library commands, confirm that user can view the Services music library in Planning Center.

## 3. Confirm pccli Runs

```sh
pccli
```

You should see the pccli banner and command help.

## 4. Choose a Product

Planning Center product commands start at the product name:

```sh
pccli services --help
```

## 5. Verify Planning Center Access

Run a basic read command against Planning Center Services:

```sh
pccli services songs list
```

If you know part of a song title, search for it:

```sh
pccli services songs search "Amazing"
```

Open one result by exact title or ID:

```sh
pccli services songs show "Amazing Grace"
```

## 6. Use JSON for Automation

```sh
pccli --json services songs list --author "Newton"
```

JSON output is intended for scripts and repeatable checks. Human output stays compact by default.

## Try Without API Access

Use mock mode for demos and local documentation checks:

```sh
PCCLI_MOCK=true pccli services songs list
```

## Next Steps

- Manage songs, arrangements, keys, PDFs, and links in [Managing Your Song Library](managing-your-song-library.html).
- Look up every flag in the [Services music-library command reference](commands.html).
