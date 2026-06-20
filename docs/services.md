---
title: Services
nav_order: 5
---

# Services

Services commands are grouped under `pccli services`.

## Music Library

Use music-library commands to find songs and review arrangements.

```sh
pccli services songs list
pccli services songs show SONG_ID
pccli services songs arrangements SONG_ID
```

Add `--json` to use command output in scripts:

```sh
pccli --json services songs list --title "Amazing%"
```

[Review the Services music-library command reference](commands.html)
