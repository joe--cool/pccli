---
title: Getting Started
nav_order: 1
---

# pccli

`pccli` is an unofficial Planning Center command line tool. It is not built, endorsed, or supported by Planning Center.

Plan services and maintain your Services song library from a terminal or script, with visible IDs and explicit parent selection.

## Get connected

1. [Install pccli](installation.html).
2. [Configure authentication](authentication.html).
3. [Verify access with the quick start](quick-start.html).

## See what pccli does

Prepare a Sunday service: choose a service type and plan, review the order, then inspect a song's available arrangements. This short mock journey mixes planning and library commands; the focused guides fill in the details.

{% include terminal-demo.html name="overview-demo" title="Prepare Sunday's service" caption="Choose Sunday Worship and its plan, review the order, then inspect Amazing Grace before rehearsal. Under 75 seconds; mock data." %}

<details markdown="1">
<summary>Read the overview steps</summary>

1. Find the service type:

   ```sh
   pccli services service-types list
   ```

   The mock result is **Sunday Worship (3001)**.

2. Choose the plan by its displayed date and title:

   ```sh
   pccli services plans list --service-type 3001
   ```

   The example plan is **Gathered in Grace (4001)** on **November 8, 2026**.

3. Review its order:

   ```sh
   pccli services items list --service-type 3001 --plan 4001
   ```

   The order includes a **Gathering** header, **Amazing Grace**, and **Welcome and Prayer**. The Amazing Grace row displays song ID **1001**. Blank song-ID cells identify items without a supplied song relationship.

4. Check that song before rehearsal:

   ```sh
   pccli services songs show 1001
   ```

   The song has **Full Band (2001)** and **Acoustic (2002)** arrangements. Use the displayed IDs to continue; your account will have different IDs and dates.

</details>

Continue with [plan readiness](planning-a-service.html#check-plan-readiness), [file placement](managing-your-song-library.html#place-files-and-links-in-the-right-scope), or [JSON for scripts](commands.html#json-and-errors) for focused recordings.

## Choose a workflow

| Task | Guide |
| --- | --- |
| Find songs and inspect their arrangements | [Services overview](services.html) |
| Create, copy, and maintain service plans, orders, times, and team assignments | [Planning a service](planning-a-service.html) |
| Maintain metadata, charts, keys, files, and tags | [Managing your song library](managing-your-song-library.html) |
| Look up command arguments and flags | [Command reference](commands.html) |
| Check editable fields and import options | [Song-library metadata fields](song-fields.html) |
| Read results in a script | [JSON and errors](commands.html#json-and-errors) |
