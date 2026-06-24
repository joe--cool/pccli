# Repository Instructions

This repo builds `pccli`, a Go CLI for Planning Center. Treat it as a user-journey CLI, not a thin wrapper around every API endpoint.

## Before Building

- Research the current Planning Center documentation for the feature area.
- Look for community notes or forum threads that explain how users actually manage that workflow.
- Compare the intended workflow against mature SaaS CLIs before choosing command names, flags, output, and error behavior.
- Decide whether the change belongs under an existing resource/workflow command or needs a new subcommand group.

## Implementation Rules

- Use Go and the current toolchain pinned in `.mise.toml`.
- Prefer proven libraries over custom CLI infrastructure. Keep dependencies current; upgrade stale dependencies when a feature needs newer library behavior.
- Use Cobra for command structure, Viper/godotenv for env config, Charmbracelet libraries for terminal UI, GoReleaser for releases, and VHS for demo recordings unless there is a specific reason not to.
- Keep default output human-readable and concise. Every data command should support `--json`.
- Design human detail views as composed product surfaces, not raw data dumps. Preserve valuable metadata, but use hierarchy, grouping, alignment, and scan-friendly sections so the output is easy to consume.
- Do not show noisy false/default state in ordinary output. For example, omit `Hidden: false`; show status fields only when they are requested or materially explain the resource.
- Keep non-interactive operation first. Add prompts only when they improve a real operator workflow, and ensure automation still has flags.
- Start commands with the Planning Center product, then keep the rest shallow and resource-first. Prefer `pccli services songs list` over deeper API-shaped paths like `pccli services library songs list`.
- Add extra namespace layers only when they resolve a real user-facing ambiguity.
- Apply workflow rules across follow-on commands, not only the first command in a resource group. If a command starts from a resource lookup, inherited behaviors such as visibility filters, name/ID resolution, and ambiguity handling should remain consistent.
- Exclude hidden resources by default wherever the API exposes hidden state, including searches, name resolution, and direct ID checks. Require an explicit flag to include or operate on hidden resources.
- When names are ambiguous or duplicate records exist, do not guess. Stop with a clear error and include enough disambiguating metadata for the user to choose an ID.
- Never read, print, or commit `.env` secrets. Update `.env.example` when config changes.
- For Planning Center Personal Access Tokens, use HTTP Basic Auth with `PCCLI_CLIENT_ID` and `PCCLI_CLIENT_SECRET`.

## Verification

- Run `go test ./...` before finishing code changes.
- Run `go mod tidy` after dependency changes.
- For CLI behavior, test both plain output and `--json`.
- Use mock mode for demos and screenshots: `PCCLI_MOCK=true`.
- Use Conventional Commits for commit messages so Release Please can manage changelogs, tags, and GitHub releases.
- AI tools are useful, but contributors are responsible for validating any generated changes before review. Do not pass unverified AI output on to maintainers to debug or validate.

## Documentation

- Keep `README.md` concise and task-oriented.
- Public GitHub Pages content lives in `docs/` and should be user-facing product documentation: installation, authentication, quick start, command reference, and generated demos.
- Do not put research notes, CLI comparisons, or implementation rationale in public docs. Put contributor guidance in `DEVELOPER.md`; use issues or PRs for research trails.
- Make the unofficial Planning Center status clear in README and public docs.
- Update VHS tapes and generated-demo references when command examples change.
- Keep quick start docs setup-first: installation, authentication, and verifying basic access. Put real operational walkthroughs in user workflow guides rather than turning the quick start into a smoke test or full feature tutorial. New feature areas should be introduced as workflow choices, not as assumptions about where every user starts.
- If the banner changes, update `scripts/banner.ansi`, run the banner sync task, and do not commit generated binaries from `bin/`.
- If a command changes user-visible behavior, update the README or docs in the same change.
