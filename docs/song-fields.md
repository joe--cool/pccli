---
title: Song-Library Metadata Fields
nav_order: 7
---

# Song-Library Metadata Fields

pccli is unofficial and is not built, endorsed, or supported by Planning Center. These flags map to the documented Services API contract used by pccli (`2018-11-01`). See the official [Song](https://api.planningcenteronline.com/docs/apps/services/versions/2018-11-01/vertices/song), [Arrangement](https://api.planningcenteronline.com/docs/apps/services/versions/2018-11-01/vertices/arrangement), and [Key](https://api.planningcenteronline.com/docs/apps/services/versions/2018-11-01/vertices/key) references.

## Update only what you intend

An omitted flag leaves the value unchanged. An explicit empty string, zero, false, or empty sequence is transmitted when valid; otherwise pccli rejects it clearly. For booleans, use `--flag=false` to turn a setting off. Names and titles cannot be empty; BPM and CCLI numbers must be positive.

```sh
pccli services songs update 1001 --author ""
pccli services arrangements update 2001 --song 1001 --notes "" --length 0 --lyrics-enabled=false --sequence ""
```

The API determines which clears and imports an account supports. A rejected write or failed verification exits nonzero; accepted provider actions may still be unverified. Inspect the reported outcome before retrying. Provider import fields are actions: API acceptance does not confirm that provider processing or generated files have completed.

## Songs

Use these on `songs create` and `songs update SONG`, except the create-only imports below.

| Flag | Type | Meaning |
| --- | --- | --- |
| `--title` | string | Song title; create requires this or `--ccli` |
| `--admin` | string | Song administrator |
| `--author` | string | Author text |
| `--copyright` | string | Copyright text |
| `--themes` | string | Themes text; separate from tag assignments |
| `--ccli` | integer | Positive CCLI song number |
| `--hidden` | boolean | Set state; existing-record selection uses `--visibility` |

Create-only string fields:

| Flag | Meaning |
| --- | --- |
| `--import-ccli-chord-chart` | CCLI chord-chart import request |
| `--import-ccli-chord-pro` | CCLI ChordPro import request |
| `--import-ccli-lyrics` | CCLI lyrics import request |

`--ccli NUMBER` without a title sends that number through Planning Center's title-based metadata lookup. Provider import strings are separate from this metadata lookup. On create, a digits-only title requires the matching `--ccli` number because the API interprets that title as a metadata lookup.

## Arrangements

Use these on `arrangements create --song SONG --name NAME` and `arrangements update ARRANGEMENT --song SONG`.

| Flag | Type | Meaning / valid values |
| --- | --- | --- |
| `--name` | string | Arrangement name; required on create |
| `--bpm` | number | Finite positive tempo |
| `--meter` | string | `2/2`, `2/4`, `3/2`, `3/4`, `4/2`, `4/4`, `5/4`, `6/4`, `3/8`, `6/8`, `7/4`, `7/8`, `9/8`, `12/4`, `12/8` |
| `--chart-key` | string | Musical pitch, such as `G`, `Bb`, or `Cm` |
| `--notes` | string | Arrangement notes |
| `--length` | duration | Nonnegative seconds or `m:ss`; `0` clears; maximum 2147483647 seconds |
| `--lyrics-enabled` | boolean | Enable or disable lyrics |
| `--sequence` | comma-separated strings | Full section labels, e.g. `'Verse 1,Chorus 1'`; empty clears |
| `--chord-chart` | string | Lyrics and chords in standard or ChordPro format |
| `--chord-chart-chord-color` | integer | Nonnegative API chord-color value |
| `--chord-chart-columns` | integer | Positive column count |
| `--chord-chart-font` | string | Font setting |
| `--chord-chart-font-size` | integer | `10`, `11`, `12`, `13`, `14`, `15`, `16`, `18`, `20`, `22`, `24`, `26`, `28`, `32`, `36`, `42`, `48` |
| `--number-chart-enabled` | boolean | Enable or disable number charts |
| `--numeral-chart-enabled` | boolean | Enable or disable numeral charts |
| `--print-margin` | string | `0.0in`, `0.25in`, `0.5in`, `0.75in`, `1.0in` |
| `--print-orientation` | string | `Portrait` or `Landscape` |
| `--print-page-size` | string | `Widescreen (16x9)`, `Fullscreen (4x3)`, `A4`, `Letter`, `Legal`, `11x17` |
| `--isrc` | integer | Nonnegative ISRC field, represented as an integer by this API |
| `--mtid` | string | Alias for rehearsal mix ID |
| `--rehearsal-mix-id` | string | Rehearsal mix ID; avoid supplying a conflicting `--mtid` |
| `--rehearsal-mix-thumbnail` | string | Rehearsal mix thumbnail value |

Import fields accepted on both create and update are all **strings**:

| Request flag | Corresponding options flag |
| --- | --- |
| `--import-chart-pro` | `--import-chart-pro-options` |
| `--import-rehearsal-mix` | `--import-rehearsal-mix-options` |
| `--import-rehearsal-pack` | `--import-rehearsal-pack-options` |

Arrangement create additionally accepts `--import-ccli-chord-chart`, `--import-ccli-chord-pro`, and `--import-ccli-lyrics`, all strings. These three flags are not supported on arrangement update.

## Keys

Use these on `keys create` and `keys update KEY`, always with `--song` and `--arrangement`.

| Flag | Type | Meaning |
| --- | --- | --- |
| `--name` | string | Key record name; required on create |
| `--start` | string | Starting pitch; required on create |
| `--end` | string | Ending pitch; defaults to start on create |
| `--alternate-keys` | string containing JSON | Array of objects with nonempty `name` and valid `key` |

Pitches use `Ab`, `A`, `A#`, `Bb`, `B`, `C`, `C#`, `Db`, `D`, `D#`, `Eb`, `E`, `F`, `F#`, `Gb`, `G`, or `G#`; append `m` for minor.

```sh
pccli services keys update 3001 --song 1001 --arrangement 2001 --alternate-keys '[{"name":"Female lead","key":"Eb"}]'
```

Use `--alternate-keys '[]'` to request an empty alternate-key set. The API defines this attribute as a string containing JSON; pccli validates the array before sending it.

Both create and update also accept these provider import strings:

| Flag |
| --- |
| `--import-ccli-chord-chart` |
| `--import-ccli-chord-chart-options` |
| `--import-ccli-lyrics` |
| `--import-chart-pro` |
| `--import-rehearsal-mix` |
| `--import-rehearsal-pack` |

## Provider import values

The Services schema declares the import and options fields above as strings but does not define a universal payload format. pccli passes the supplied string to that field; it does not invent an import format or convert these flags into booleans. Use values supported by the provider and your Planning Center integration. Import availability can depend on permissions, subscriptions, and provider setup.

Readable metadata includes timestamps, chart text and settings, lyrics, complete sequence, source IDs, archive state, and key minor flags. Use `show --json` when you need all returned metadata, including false or zero values suppressed in concise human output.
