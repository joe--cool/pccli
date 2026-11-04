---
title: Installation
nav_order: 2
---

# Installation

pccli is an unofficial project, not built, endorsed, or supported by Planning Center.

## Install a release

Download the archive for your operating system and CPU from the [Downloads page]({{ site.docs_root | append: '/downloads.html' }}). Extract it and put `pccli` (or `pccli.exe`) in a directory on your `PATH`. Release assets include checksums.

## Install with Go

With Go installed, run:

```sh
go install github.com/joe--cool/pccli/v2/cmd/pccli@latest
```

Add Go's binary installation directory (normally `~/go/bin`) to your `PATH` if needed. Source builds use the Go version declared in the module; the current repository pins Go 1.27.3.

```sh
pccli --version
pccli --help
```

Help and version work without credentials. To update a Go installation, run the same `go install` command again. For release archives, replace the binary with the new release.

Next, [configure your Personal Access Token](authentication.html).
