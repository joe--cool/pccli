---
title: Services
nav_order: 5
---

# Services

pccli is unofficial and is not built, endorsed, or supported by Planning Center.

Use `pccli services RESOURCE ACTION` to work with the Services song library.

| Resource | Tasks |
| --- | --- |
| `songs` | Find, inspect, create, update, delete, and inspect scheduling history |
| `arrangements` | Maintain song versions, chord charts, print settings, sequence, and archive state |
| `keys` | Maintain named keys and alternate keys within an arrangement |
| `attachments` | List, inspect, add, update, and delete files or links at song, arrangement, or key scope |
| `tags` | Discover available or assigned tags; replace a song's or arrangement's complete assignment |
| `tag-groups` | Discover song and arrangement tag classifications |
| `attachment-types` | Discover available file classifications |

Start with a read:

```sh
pccli services songs list --search "Amazing"
pccli services songs show 1001
pccli services arrangements list --song 1001
pccli services keys list --song 1001 --arrangement 2001
```

An arrangement always belongs to an explicitly selected song; a key always requires both song and arrangement. There is no inferred first or default arrangement. IDs are visible so follow-up scripts can identify the intended record.

[Follow the song-library workflow](managing-your-song-library.html), [look up commands](commands.html), or [review editable metadata](song-fields.html).
