# Feature: knowledge-check

## Objective
Let AI agents consult the owner's hand-written knowledge corpus (Markdown recipes, e.g.
`~/.claude/skills/unity-knowledge/entries/*.md`, 176 entries) before they build something, through Thoughtline.

## Problem / why
The corpus is real, hand-written knowledge, but today it is only reachable by an agent reading a skill index and
deciding to open entries. Nothing checks, at the moment of writing code, whether a recipe or a project rule applies.

## Scope
- `internal/knowledge`: parse entries (frontmatter `title/tags/source`; procedural template `## Guidance / Applies
  when / Why this works / Procedure / Validate the result / Limits`; code template `## Context / Example / Notes`),
  scan a directory, read topic groups from the index file, detect foreign-stack markers.
- `thoughtline knowledge sync <dir>`: import entries as memories in a dedicated project (default
  `knowledge-unity`), topic_key `knowledge/<slug>`, idempotent; entries whose file disappeared are soft-deleted.
  The `.md` files stay the source of truth.
- `tl_check` MCP tool + `thoughtline check` CLI (+ `--hook` mode for Claude Code PreToolUse): given a task and
  file paths, return the top applicable entries with Applies when / Validate the result / Limits, plus matching
  rules of the current project (project rules win); exclude configured foreign stacks.
- Docs: `docs/TOOLS.md`, `README.md`, protocol CORE TOOLS, `CHANGELOG.md`.
- Measure precision against recent NinjaSurvivor commits before enabling the hook.

## Constraints
- Closed type set (14); tags match `^[a-z0-9][a-z0-9:_-]{0,40}$`; topic_key `^[a-z0-9][a-z0-9/_-]{1,128}$`.
- No new schema migration unless unavoidable (knowledge lives in `memories` under its own project).
- Token economy (architecture/token-economy): hook output small, deduped per file per session.
- Stdlib `testing`; every task keeps `go build ./...` and `go test ./...` green.

## Delivery
Strategy: ask-on-risk. Forecast ~900 authored lines over T1-T4 (> 400): slice decision deferred to the owner before
any PR; work-unit commits on `feat/knowledge-check`. Push/PR/merge are the owner's decisions.
RDD: off (global) — no native review; ordinary checks only.

## Tasks
- [x] T1 `internal/knowledge` parser + scanner + group index + stack detection, with tests. Route: delegated (nan writer; new package, 2+ files). Commit `aaf131d`. Evidence: `go test ./...` all ok; real corpus 176 entries, 176 grouped, procedural 111 / recipe 65; stacks tactile 13, fiber 9, svn 7, level-editor 7, belt 6, spine 2.
- [x] T2 `thoughtline knowledge sync` CLI (upsert by topic_key, soft-delete removed, tags, type mapping), with tests. Route: delegated. Commit `1ec01c4`. Evidence: real corpus on temp DB `created=176` then `unchanged=176`; `go test ./...` all ok after rebase. Types: procedural→convention, recipe→script-pattern; identify knowledge by tag `kind:knowledge`. Refuses to prune on a 0-entry scan.
  - Rebased the branch onto origin/main (local main was 88 commits behind; the installed binary's 14-type set lives upstream). Local `main` fast-forwarded to origin/main.
- [x] T3 `tl_check` MCP tool + `thoughtline check [--hook]` CLI, with tests. Route: delegated. Commit `02ad8f2`. Evidence: `go test ./...` 13/13 ok; search.go diff reviewed (MatchAny opt-in, default byte-identical).
- [x] T5b/T5c Absolute coverage gate + IDF weight + wordless sync footer (reopened from T5 measurement). Route: delegated. Commit `705be73`.
  - Run 2 (coverage≥2 only): still noisy — generic tokens (`games` from the GorodiskiGames split, state, setup, one, per).
  - Run 3 (vendor dirs dropped before CamelCase split, code stopwords, IDF weight ≥ 2·idf(df=10% N) = 6.46 at N=176): hook mode injects on 5/20 commits, relevant top hit 3/5; silent on 15/20. Task+paths mode with raw commit subjects: 14/20 with hits, weaker (~1/3) — agents should pass a descriptive task.
  - Why: T5 run 1 on 20 NinjaSurvivor commits (`measure.ps1`): top-hit precision ≈ 20-25% (paths-only and task+paths). Relative normalisation makes the top hit always 1.00, so MinScore never gates; the footer's group/source words are FTS-indexed and add false matches. Explicit tasks ("camera shake when hit") work well.
- [x] T4 Docs (TOOLS.md, README, protocol, CHANGELOG, memory skill). Route: delegated (mechanical). Evidence: `go test ./...` 13/13 ok; `thoughtline protocol` lists tl_check + "check knowledge before writing code" bullet. protocolVersion left at "1" (also pinned in plugin.json; bump both together if ever).
- [x] T5 Precision measurement on ~20 recent NinjaSurvivor commits; tune threshold. Route: inline (scratchpad `measure.ps1`, throwaway DB). Run 1: top-hit precision ≈ 20-25%, every commit injected → reopened as T5b/T5c.
- [ ] T6 Install binary, sync the unity corpus, wire MCP env + PreToolUse hook for NinjaSurvivor (Claude + Pi). Route: inline.

## Acceptance criteria
- Sync of the 176-entry corpus: 176 memories, re-run = 0 changes, deleting a file soft-deletes its memory.
- `tl_check` on a task like "add camera shake on hit" returns the camera-shake entries first, with their sections.
- Foreign-stack entries (SVN/Fiber/Tactile/belt) excluded when configured.
- Hook adds context at most once per file per session and stays silent below the threshold.

## Checks
`go build ./...`, `go test ./...`, `go vet ./...`.

## Progress / evidence
- Baseline 2026-10-17: build OK, all 12 packages `ok`.

## Next step
T2 (in progress).
