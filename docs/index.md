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

## See what pccli does

Find a song, compare its arrangements, and select a key using the IDs in your results. After connecting your account, you can also maintain metadata, charts, files, and tags with the same resource-first commands.

{% include terminal-demo.html name="overview-demo" title="Explore your song library" caption="Find Amazing Grace, inspect its arrangements, then list the keys for Full Band." %}

<details markdown="1">
<summary>Read the overview steps</summary>

1. Find the song:

   ```sh
   pccli services songs list --search "Amazing"
   ```

   The mock result is **Amazing Grace**, song ID **1001**.

2. Inspect its metadata and versions:

   ```sh
   pccli services songs show 1001
   ```

   The song view shows author, CCLI number, notes, and two arrangements: **Full Band (2001)** in G and **Acoustic (2002)** in D.

3. Select Full Band explicitly:

   ```sh
   pccli services keys list --song 1001 --arrangement 2001
   ```

   Its key records are **Default (3001)** in G and **Female Lead (3002)** in Bb. Use the returned IDs in follow-up commands. Your own account will have different IDs.

</details>

Continue with [file placement](managing-your-song-library.html#place-files-and-links-in-the-right-scope) or [JSON for scripts](commands.html#json-and-errors) for focused recordings.

## Choose a workflow

| Task | Guide |
| --- | --- |
| Find songs and inspect their arrangements | [Services overview](services.html) |
| Maintain metadata, charts, keys, files, and tags | [Managing your song library](managing-your-song-library.html) |
| Look up command arguments and flags | [Command reference](commands.html) |
| Check editable fields and import options | [Song-library metadata fields](song-fields.html) |
| Read results in a script | [JSON and errors](commands.html#json-and-errors) |
