---
title: Services
nav_order: 5
---

# Services

Services commands are grouped under `pccli services`.

## Music Library

Use music-library commands to find songs, review arrangements and keys, and manage song attachments.

```sh
pccli services songs list
pccli services songs search "Amazing"
pccli services songs show "Amazing Grace"
pccli services songs arrangements "Amazing Grace"
pccli services songs keys "Amazing Grace"
pccli services songs attachments "Amazing Grace"
```

Follow the full search-to-files workflow in [Managing Your Song Library](managing-your-song-library.html).

Add `--json` to use command output in scripts:

```sh
pccli --json services songs list --title "Amazing%"
```

Create or update song metadata when maintaining the library from a script:

```sh
pccli services songs create --title "Amazing Grace" --author "John Newton" --ccli 22025
pccli services songs update "Amazing Grace" --themes "Grace, Hymn"
pccli services songs arrangements create "Amazing Grace" --name "Full Band" --key G
pccli services songs keys create "Amazing Grace" "Full Band" --name "Default" --start G
```

Attach charts, PDFs, or links at the song level, or add `--arrangement` and `--key` to scope the attachment more narrowly:

```sh
pccli services songs attach "Amazing Grace" --file ./lead-sheet.pdf
pccli services songs attach "Amazing Grace" --url "https://example.com/rehearsal-track"
```

[Review the Services music-library command reference](commands.html).
