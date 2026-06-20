---
title: Quick Start
nav_order: 4
---

# Quick Start

This guide gets pccli installed, authenticated, and verified. The first available workflow is Services music-library discovery; future workflows should fit into this same setup-first quick start.

## 1. Install

```sh
go install github.com/joe--cool/pccli/cmd/pccli@latest
```

## 2. Configure

Create a `.env` file with your Planning Center Personal Access Token:

```sh
PCCLI_CLIENT_ID=your_client_id
PCCLI_CLIENT_SECRET=your_secret
```

## 3. Confirm pccli Runs

```sh
pccli
```

You should see the pccli banner and command help.

## 4. Choose a Workflow

The current workflow reads from the Services music library.

### List Songs

```sh
pccli songs list
```

Filter by title when you know what you are looking for:

```sh
pccli songs list --title "Amazing%"
```

### Inspect a Song

```sh
pccli songs show SONG_ID
```

### Review Arrangements

```sh
pccli songs arrangements SONG_ID
```

## 5. Use JSON for Automation

```sh
pccli --json songs list --author "Newton"
```

## Try Without API Access

Use mock mode for demos and local documentation checks:

```sh
PCCLI_MOCK=true pccli songs list
```
