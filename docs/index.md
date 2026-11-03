---
title: Getting Started
nav_order: 1
---

# pccli

`pccli` is an unofficial Planning Center command line tool. It is not built, endorsed, or supported by Planning Center.

Use a terminal or script to find and maintain your Services song library, with clear IDs and explicit song, arrangement, and key context.

## Get connected

1. [Install pccli](installation.html).
2. [Configure authentication](authentication.html).
3. [Verify access with the quick start](quick-start.html).

## Choose a workflow

| Task | Guide |
| --- | --- |
| Find songs and inspect their arrangements | [Services overview](services.html) |
| Maintain metadata, charts, keys, files, and tags | [Managing your song library](managing-your-song-library.html) |
| Look up command arguments and flags | [Command reference](commands.html) |
| Check editable fields and import options | [Song-library metadata fields](song-fields.html) |
| Read results in a script | [JSON and errors](commands.html#json-and-errors) |

```sh
pccli services songs list --search "Amazing"
pccli services songs show 1001
pccli services arrangements list --song 1001
```

## See the workflow

<img class="terminal-demo" src="{{ '/assets/library-demo.gif' | relative_url }}" alt="Mock terminal demo finding a song and listing its arrangements, keys, and attachments" />
