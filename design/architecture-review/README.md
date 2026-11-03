# pccli architecture and direction

This packet preserves the original audit, the independent Astra architecture review, and the selected direction. It is contributor material, not public product documentation. The subsequent implementation and its validation are recorded in [Implementation and remaining gaps](implementation.md).

Reviewed baseline: `d8c596895cdff92b54ce91d89ae61335e5125eb8`.

The owner confirmed there is no adoption and no backward-compatibility requirement. The recommended interface can replace existing commands and JSON shapes directly; the plan excludes legacy aliases and deprecation work.

## Reading order

| Document | Purpose |
| --- | --- |
| [Implementation and remaining gaps](implementation.md) | Current capability coverage, implementation decisions, validation, and remaining limits. |
| [Recommended direction](direction.md) | Start here: product vision, selected architecture, reasons, gaps, and the sequence to implement. |
| [Astra architect review](astra-review.md) | Independent assessment, detailed contracts, alternatives, implementation sequence, and test requirements. |
| [Current state and initial proposal](brief.md) | Source map, present command tree, evidence, gap IDs G1–G12, original proposal and external references. |
| [Architect prompt](architect-prompt.md) | Exact review assignment, constraints, required decisions and stress cases; reusable for a future review at a new baseline. |

## What the review established

The audited baseline had a useful foundation: Go/Cobra, product-first commands, concise terminal output, JSON for scripts, and explicit deletion confirmation. Its simplification was incomplete because command grammar varies, target-selection rules live in the command layer, and collection completeness is lost before identity resolution.

The most urgent issues were duplicate records beyond the first page, patch values that lose explicit empty/zero inputs, visibility selection coupled to mutation state, and ambiguous implicit arrangement selection. Those findings drove the implementation and its regression tests.

Evidence in the brief distinguishes executed synthetic reproductions, source observations, uncertain remote API contracts, and product-design judgments. A passing local suite does not prove the remote API accepts every advertised operation.

## Historical scope

The initial review changed contributor documents only. The brief, architect prompt, and Astra report describe that historical baseline, not the current implementation. The subsequent breaking implementation has no compatibility layer, uses current stable dependencies and tools, and includes public documentation and synthetic workflow verification. See the implementation record for evidence and limits; no live account behavior is claimed.
