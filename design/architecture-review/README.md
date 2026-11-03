# pccli architecture and direction

This packet records the current project, an independently reviewed design proposal, and the recommended implementation direction. It is contributor material, not public product documentation. It proposes future behavior; the application has not been refactored as part of this review.

Reviewed baseline: `d8c596895cdff92b54ce91d89ae61335e5125eb8`.

The owner confirmed there is no adoption and no backward-compatibility requirement. The recommended interface can replace existing commands and JSON shapes directly; the plan excludes legacy aliases and deprecation work.

## Reading order

| Document | Purpose |
| --- | --- |
| [Recommended direction](direction.md) | Start here: product vision, selected architecture, reasons, gaps, and the sequence to implement. |
| [Astra architect review](astra-review.md) | Independent assessment, detailed contracts, alternatives, implementation sequence, and test requirements. |
| [Current state and initial proposal](brief.md) | Source map, present command tree, evidence, gap IDs G1–G12, original proposal and external references. |
| [Architect prompt](architect-prompt.md) | Exact review assignment, constraints, required decisions and stress cases; reusable for a future review at a new baseline. |

## What the review established

The current CLI has a useful foundation: Go/Cobra, product-first commands, concise terminal output, JSON for scripts, and explicit deletion confirmation. The simplification is incomplete because command grammar varies, target-selection rules live in the command layer, and collection completeness is lost before identity resolution.

The most urgent issues are duplicate records beyond the first page, patch values that lose explicit empty/zero inputs, visibility selection coupled to mutation state, and ambiguous implicit arrangement selection. Fixing these takes priority over renaming commands or adding features.

Evidence in the brief distinguishes executed synthetic reproductions, source observations, uncertain remote API contracts, and product-design judgments. A passing local suite does not prove the remote API accepts every advertised operation.

## Scope of this deliverable

- Architecture/design documents and a contributor navigation link only.
- No application behavior changes, dependency changes, live Planning Center operations, publication, or commits.
- The implementation sequence is recommended engineering direction; it does not claim that the proposed runtime changes have already been built or validated against a live account.
