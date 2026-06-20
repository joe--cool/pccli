---
title: Getting Started
nav_order: 1
---

# Getting Started

`pccli` is an unofficial command line tool for working with Planning Center from a terminal or script. The first supported workflow is read-only discovery in the Services music library.

<div class="info-banner">
  <p><strong>Unofficial project:</strong> pccli is not built, endorsed, or supported by Planning Center. Use it with the same care you would use for any third-party API client.</p>
</div>

## What You Can Do Today

- List songs in the Services music library.
- Filter songs by title, author, or CCLI number.
- Show details for one song.
- List arrangements for a song.
- Emit JSON for scripts and automation.

## Quick Example

<img class="terminal-demo" src="{{ '/assets/library-demo.gif' | relative_url }}" alt="Terminal demo showing pccli listing songs and arrangements" />

```sh
pccli songs list --title "Amazing%"
pccli songs show 1001
pccli songs arrangements 1001
```

## Next Steps

- [Install pccli](installation.html)
- [Configure authentication](authentication.html)
- [Follow the quick start](quick-start.html)
- [Review the command reference](commands.html)
