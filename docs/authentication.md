---
title: Authentication
nav_order: 3
---

# Authentication

pccli uses Planning Center Personal Access Tokens for the current single-organization workflow.

<div class="warning-banner">
  <p><strong>Keep tokens private.</strong> A Personal Access Token acts with your Planning Center user permissions. Do not commit it, paste it into issues, or share it with other users.</p>
</div>

## Create a Personal Access Token

Planning Center documents token creation in its API authentication guide:

- [Planning Center API authentication](https://api.planningcenteronline.com/docs/overview/authentication)
- [Planning Center developer account](https://api.planningcenteronline.com/oauth/applications)

Create a Personal Access Token, then use the generated `client_id` and `secret`.

## Configure pccli

For project-local config, copy the example env file:

```sh
cp .env.example .env
```

Fill in:

```sh
PCCLI_CLIENT_ID=your_client_id
PCCLI_CLIENT_SECRET=your_secret
```

`.env` is ignored by git. For a user-level config that works outside this repo, create:

```sh
mkdir -p ~/.pccli
$EDITOR ~/.pccli/.env
```

pccli loads configuration in this order:

1. Shell environment variables.
2. `.env` in the current directory.
3. `~/.pccli/.env`.
4. Built-in defaults.

This lets a project directory override your user-level config when needed.

## Permissions

Planning Center API requests act as the user who owns the token. If a command cannot see a song or arrangement, confirm that the token's user has the right Planning Center Services permissions.
