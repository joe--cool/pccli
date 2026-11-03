# Independent Astra architect prompt

You are the independent architect for pccli, an unofficial Planning Center Go CLI. The owner wants a simple, understandable, operator-friendly CLI and clean internal APIs. They want one decisive architecture, vision and implementation direction that another agent can execute without another architecture round.

Owner clarification received during the review: "Nothing needs to be backward compatible. There isn’t any adoption yet." This supersedes compatibility uncertainty in the initial proposal. Choose the best coherent command, JSON and error contracts on their merits. Do not add legacy aliases, compatibility adapters, deprecation cycles, adoption checks or migration gates. Update docs and demos with the new interface. Record the clarification and its implications in the report.

Read `design/architecture-review/brief.md` first, then `AGENTS.md` and the source/doc files identified in its source map. You have access to the same checkout. Do not read `.env` files or use real credentials. This is a design review, not authorization to change application code, make live writes, publish or open a PR. Do not spawn additional agents.

The brief contains current facts, locally reproduced problems, unverified remote contract questions, and an initial proposal. Treat them differently. Confirm key facts in the code. Do not turn uncertain API mismatches into confirmed failures. The proposed sibling-resource command tree is a hypothesis: reject it if another model better serves these users. Do not optimize merely for fewer tokens, fewer files, formal layering, or resemblance to one fashionable CLI.

Your task is to produce a coherent end-state design with a defensible route from this checkout. Make choices where possible; if evidence is missing, choose a reversible default and identify the exact verification needed. Do not return a menu of equally recommended alternatives, a generic clean-code checklist, or feedback that requires the owner to resolve routine engineering questions.

Evaluate, concretely:

1. Product vision and the small set of user journeys worth optimizing; what stays out of scope.
2. Current strengths, architecture problems, and corrections to the supplied audit. Separate proven defects, risks, preferences, and unknowns.
3. Recommended command tree for songs, arrangements, keys and attachments. Show list/detail/create/update/delete or add only where supported. Specify all parent inputs, target lookup, search behavior, defaults, hidden selection vs state changes, musical key vs key record naming, numeric-name escape, confirmations, and unsupported capability handling.
4. Package/dependency ownership. Show the request-to-result path and Go type/function/interface sketches sufficient to locate implementation responsibilities. Keep abstractions proportional to the codebase. Decide where DTOs, product models, envelopes, pagination traversal, uniqueness checks, detail composition and presentation belong.
5. Exact rules for bounded list output vs exhaustive resolution, incomplete lookups, remote failures, duplicate records across pages and visibility sets, and invalid parent/child scopes. Discuss what can and cannot be guaranteed with concurrent remote changes.
6. Mutation semantics for absent/empty/zero/false/empty arrays/null. Distinguish valid clearing from invalid resource values; avoid accepting a no-op clear as success. Explain create vs patch inputs.
7. Human and JSON output contracts, empty results, pagination metadata, machine-readable errors if justified, exit codes, stdout/stderr, and signal cancellation. Establish deliberate future-facing contracts; existing script compatibility is not required.
8. Typed API capabilities/version policy, retry rules, upload streaming, partial success/recovery and errors. Unknown API write support must not become an implementation promise.
9. Alternatives rejected with costs and reasons, especially keeping nested resource commands vs the sibling-resource proposal, keeping search vs merging it, and minimal extraction vs extra architectural layers.
10. Clean transition to the new interface with no backward-compatibility work. Distinguish immediate bug fixes, behavior changes, internal refactoring, public command/schema changes, and deferred feature work. Remove replaced syntax and update docs/demos together.
11. Gap register with priorities, evidence needed and exit criteria. Define a behavioral test matrix and a sequence of reviewable PR-sized steps with dependencies, acceptance criteria and stop conditions. The first step must be something an implementing agent can start directly.

Use at least these stress cases:

- A second exact song title exists on page two after a page containing one exact and nine partial matches.
- A page fetch fails midway through identity resolution.
- A title is entirely numeric; ID-like strings include negative values or overflow.
- Duplicate exact titles span visible and hidden resources; the user asks for all visibility states.
- Two arrangements share the name Default; there is no reliable API default marker.
- A hidden song is unhidden, and a visible song is hidden, by both ID and name.
- A patch clears author/notes/sequence and sets length to zero or a boolean false.
- A song has no arrangements but the caller explicitly requests one.
- An upload succeeds and attachment creation fails or times out; blindly replaying could duplicate it.
- The API returns an untrusted next-page URL or a malformed collection.
- A human runs help with no credentials, a script requests a large JSON list, and cancellation arrives during retry wait.

Deliver your complete review to `design/architecture-review/astra-review.md`. Include your model/role, date, reviewed SHA, a clear recommendation, and any meaningful disagreement with the brief. Explain your choices using evidence and concrete command examples, not just principles. It is acceptable to correct or replace the initial proposal. Do not edit the brief or other repository files. Your final reply should identify the report and summarize the strongest recommendations and remaining blockers. Finish and stop after delivering; do not keep working or schedule follow-up activity.
