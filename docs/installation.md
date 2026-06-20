---
title: Installation
nav_order: 2
---

# Installation

Released binaries will be published for macOS, Linux, and Windows through GitHub Releases.

## Build From Source

Until the first release is published, install from source with Go:

```sh
go install github.com/joe--cool/pccli/cmd/pccli@latest
```

Confirm the binary is available:

```sh
pccli --version
```

## Requirements

- Go is required only when building from source.
- A Planning Center user account is required for API access.
- A Personal Access Token is recommended for single-church/local use.

## Updates

When using `go install`, run the same install command again to update to the latest published version.
