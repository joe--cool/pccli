---
title: Quick Start
nav_order: 4
---

# Quick Start

This guide gets pccli installed, authenticated, and ready for product-specific workflows.

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

## 4. Choose a Product

Planning Center product commands start at the product name:

```sh
pccli services --help
```

## 5. Work With Services

### List Songs

```sh
pccli services songs list
```

Filter by title when you know what you are looking for:

```sh
pccli services songs list --title "Amazing%"
```

### Inspect a Song

```sh
pccli services songs show SONG_ID
```

### Review Arrangements

```sh
pccli services songs arrangements SONG_ID
```

## 6. Use JSON for Automation

```sh
pccli --json services songs list --author "Newton"
```

## Try Without API Access

Use mock mode for demos and local documentation checks:

```sh
PCCLI_MOCK=true pccli services songs list
```
