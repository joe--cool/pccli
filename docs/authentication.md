---
title: Authentication
nav_order: 3
---

# Authentication

pccli is unofficial and is not built, endorsed, or supported by Planning Center.

## Create a token

Create a Personal Access Token in your [Planning Center developer account](https://api.planningcenteronline.com/oauth/applications). Follow the [official authentication guide](https://api.planningcenteronline.com/docs/overview/authentication). pccli uses the generated client ID and secret with HTTP Basic Auth.

A token acts with its owner's Planning Center permissions. Keep it private; never commit it or paste it into issues, logs, or shared examples.

## Save your credentials

For configuration shared across directories:

```sh
mkdir -p ~/.pccli
$EDITOR ~/.pccli/.env
```

Put these values in the file:

```dotenv
PCCLI_CLIENT_ID=your_client_id
PCCLI_CLIENT_SECRET=your_secret
```

In a repository checkout, you can instead copy `.env.example` to a local `.env` and edit it. The repository ignores `.env`.

Configuration precedence is:

1. Shell environment variables.
2. `.env` in the current working directory.
3. `~/.pccli/.env`.
4. Built-in defaults.

Only the documented `PCCLI_*` configuration names are supported. A malformed dotenv file produces an error without echoing its contents.

## Verify access

```sh
pccli services songs list --limit 1
```

An empty list can still confirm access. Authentication or permission errors require checking the token and its owner's Services access. Help and version do not load credentials.

For mock demonstrations, use `PCCLI_MOCK=true`; this uses local fixtures and does not verify live account access. See [configuration options](commands.html#environment-variables).
