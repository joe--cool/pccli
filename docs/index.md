---
title: Getting Started
nav_order: 1
---

# pccli

`pccli` is an unofficial command line tool for working with Planning Center from a terminal or script. Commands are organized by Planning Center product so each workflow starts from the part of Planning Center you already know.

<div class="info-banner">
  <p><strong>Unofficial project:</strong> pccli is not built, endorsed, or supported by Planning Center. Use it with the same care you would use for any third-party API client.</p>
</div>

## Start

1. [Install pccli](installation.html).
2. [Configure authentication](authentication.html).
3. [Run the quick start](quick-start.html).

## Products

### Services

Use Services commands to search the music library, confirm songs, review arrangements and keys, and manage song files.

[Open the Services guide](services.html)

[Learn how to manage your song library](managing-your-song-library.html)

## Example

<img class="terminal-demo" src="{{ '/assets/library-demo.gif' | relative_url }}" alt="Terminal demo showing pccli Services music-library commands" />

```sh
pccli services songs search "Amazing"
pccli services songs show "Amazing Grace"
pccli services songs arrangements "Amazing Grace"
```
