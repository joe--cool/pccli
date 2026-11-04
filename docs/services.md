---
title: Services
nav_order: 5
---

# Services

pccli is unofficial and is not built, endorsed, or supported by Planning Center.

Use `pccli services RESOURCE ACTION` to inspect service plans and maintain the Services song library.

| Resource | Tasks |
| --- | --- |
| `service-types` | Discover service types and their IDs |
| `plans` | Create undated plans, inspect and update labels/settings/series, copy selected components, and delete with confirmation |
| `items` | Build and inspect a plan's order, link songs, reorder items, and delete with confirmation |
| `plan-times` | Create, inspect, update, and delete individual service, rehearsal, and other times |
| `teams` | Inspect teams associated with a service type |
| `team-people` | Discover team members as scheduling candidates |
| `team-positions` | Discover available positions within a team |
| `scheduled-people` | Create, inspect, update, and delete whole-plan assignments |
| `needed-positions` | Track whole-plan unfilled quantities with honest position-ID verification |
| `songs` | Find, inspect, create, update, delete, and inspect scheduling history |
| `arrangements` | Maintain song versions, chord charts, print settings, sequence, and archive state |
| `keys` | Maintain named keys and alternate keys within an arrangement |
| `attachments` | List, inspect, add, update, and delete files or links at song, arrangement, or key scope |
| `tags` | Discover available or assigned tags; replace a song's or arrangement's complete assignment |
| `tag-groups` | Discover song and arrangement tag classifications |
| `attachment-types` | Discover available file classifications |

The [front-page overview recording](index.html#see-what-pccli-does) follows a Sunday plan through its order to a song's arrangements. A [focused planning recording](planning-a-service.html#check-plan-readiness) checks times, teams, and responses. Focused recordings cover [file placement](managing-your-song-library.html#place-files-and-links-in-the-right-scope) and [JSON exports](commands.html#json-and-errors).

Start with a read:

```sh
pccli services songs list --search "Amazing"
pccli services songs show 1001
pccli services arrangements list --song 1001
pccli services keys list --song 1001 --arrangement 2001
```

An arrangement always belongs to an explicitly selected song; a key always requires both song and arrangement. There is no inferred first or default arrangement. IDs are visible so follow-up scripts can identify the intended record.

[Plan a service](planning-a-service.html), [follow the song-library workflow](managing-your-song-library.html), [look up commands](commands.html), or [review editable metadata](song-fields.html).
