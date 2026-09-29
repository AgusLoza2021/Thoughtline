# Memory domain — the vocabulary

This is the canonical vocabulary of Thoughtline: what a "memory" is, which types it can take, and how each type is shaped.

Since 2026-09-24 this vocabulary runs on [Engram](https://github.com/Gentleman-Programming/engram) — the storage engine is no longer part of this repository. That changes one thing above all: the **enforcement model**. No code can reject a malformed memory for you any more. The catalogue below is enforced by *instructing your agent*, which makes the encoding rules and the per-type sections the entire mechanism rather than documentation of a validator.

If you are adding a new type, follow [Adding a new memory type](#adding-a-new-memory-type) at the bottom.

---

## The envelope

Every memory — regardless of type — carries the same envelope. On Engram you set six of these fields and Engram owns the rest.

**You set these:**

| Field       | Required | Notes                                                                                        |
| ----------- | -------- | -------------------------------------------------------------------------------------------- |
| `title`     | yes      | Short, searchable. Imperative form preferred. ≤ 200 chars                                     |
| `content`   | yes      | Markdown body — this is where the per-type sections below live                                |
| `type`      | yes      | One of the catalogue values below. Engram accepts **any** string, so the discipline is yours |
| `topic_key` | no       | Stable key for evolving topics — see below                                                    |
| `scope`     | no       | `project` (default), `personal`, or `global` — see below                                      |
| `project`   | no       | Defaults to Engram's resolved project (explicit → `ENGRAM_PROJECT` → cwd)                      |

**Engram owns these — never write them by hand:** `id`, `sync_id`, `revision_count`, `created_at`, `updated_at`, `last_seen_at`, `duplicate_count`, `session_id`. They are what makes provenance and upserts work.

### Where tags go

The retired engine gave every memory a `tags` array. **Engram has no tags field** — `mem_save` accepts `title`, `content`, `type`, `scope`, `topic_key`, `project` and `session_id`, and nothing else. So the tagging convention splits in two:

1. **Engine, platform and pipeline tags belong in the `topic_key`.** The key patterns in the catalogue below already carry them: `perf/android/static-batching` states the platform, `scene/playcanvas/interactive-prop` states the engine. This is *better* than a tag — the key is a first-class field you can upsert against and address by name.
2. **Every other tag goes on a `**Tags**:` line as the first line of `content`**, following the namespaces in [`tag-conventions.md`](tag-conventions.md) (asset category, phase, tooling, performance buckets, ...).

Rule 2 is viable because Engram's full-text search indexes the body, so a tag that exists only inside `content` stays findable. Verified against Engram 2.x on 2026-09-24 with a body-only probe token: saved inside the body, found by search, never present in the title.

What you give up is tag **filtering** — Engram cannot filter by tag, only search for it. What you keep is a namespaced, greppable vocabulary in every memory. If Engram ever grows a tags field, only rule 2 changes.

---

### About `topic_key`

A `topic_key` is a stable, readable string naming *what this memory is about* — a slug for a wiki article. It is Engram's own concept, and this vocabulary is built on it.

Pass a `topic_key` to `mem_save` and Engram upserts on `(project, topic_key)`:

1. **First write — inserts.** New `id`, new `sync_id`, `created_at` stamped, `revision_count` 1.
2. **Every later write with the same key — updates in place.** *Same* `id` and *same* `sync_id`, `created_at` preserved, `updated_at` bumped, `revision_count` + 1.

Verified end to end against Engram 2.x on 2026-09-24: two saves sharing a key returned the same `id`, kept `created_at` from the first, and reported `revision_count` 2.

> **Gotcha — the upsert replaces, it does not merge.** The later write's `title` and `content` overwrite the earlier ones and the previous text is gone. A keyed memory is a *topic*, not a log: write what is currently true, not a changelog. If you want history, leave `topic_key` unset, or keep the history inside `content`.

Recommended `topic_key` shape: `category/subject` or `category/subcategory/subject`. Examples:

- `architecture/inn-entity-hierarchy`
- `pipeline/blender-to-pc/lantern-import`
- `perf/android/chair-batching`
- `convention/script-naming`

A dot is allowed, and its one intended use is a version number: `audit/v0.0.1-features-apagadas`, `design/gdd/crowd-control-v1.1`. That is what the dotted keys in real stores are for, and it is the one case where the key itself should record which revision of a thing you are describing.

The retired engine added a search shortcut that matched queries containing `/` against `topic_key` first — see [ADR 0002](../decisions/0002-search-strategy-fts5-first.md). On Engram there is no such shortcut: you search the key as ordinary text, which is exactly what the examples above are shaped to survive.

### About `scope`

- `project` (default) — bound to a single project. Most memories live here.
- `personal` — cross-project, per-developer. Use sparingly, for ergonomics ("I prefer 4-space indents in shaders") that travel with the dev, not the project.
- `global` — cross-project, machine-wide. Measured on a real store: **30 memories across 6 projects**, two orders of magnitude rarer than `project`. The rarity is the useful part: a memory that belongs to *every* project is usually a symptom of one that belongs to none, so reach for `global` deliberately rather than by default.

---

## The type catalogue

Fourteen types in two tiers.

**Core** — the seven the agent ecosystem already teaches. Every session on this stack is instructed with `type: bugfix | decision | architecture | discovery | pattern | config | preference`, whether or not this repository exists. Where a catalogue and that instruction disagree, an agent obeys whichever it read last and both lose their point. So the core tier is chosen to **agree with it**, not to be interesting.

**Extension** — this repository's own additions, aimed at game projects. Optional: adopt one only if your domain asks for it.

Engram's own tools write their own types, and those belong to neither tier: `session_summary` (459 observations) and `manual` (91 — the field's default when you pass no type at all). This catalogue governs the memories *you* decide to save.

### Core types

Ordered by measured use across 2,336 observations in 13 projects. The order is evidence, not taste.

| Type           | Measured use | What it captures                                                 |
| -------------- | ------------ | ---------------------------------------------------------------- |
| `discovery`    | 447          | A non-obvious thing you found out that no existing memory covers  |
| `architecture` | 391          | System structure: packages, boundaries, contracts                |
| `bugfix`       | 390          | A bug, its root cause, and the fix                               |
| `decision`     | 326          | A technical or product decision with rationale                   |
| `config`       | 84           | A configuration or environment fact that must be reproduced       |
| `pattern`      | 48           | A reusable solution shape                                        |
| `preference`   | 42           | Per-developer ergonomics (**scope = personal**)                  |

#### `discovery`

A non-obvious thing you found out that no existing memory covers. The largest core type by a wide margin — a third of everything an agent chooses to save is this one.

- **When to use**: you learned something the next session would not have guessed, and knowing it changes what that session does. If a more specific type fits — a bug, a decision, a performance trap, a config value — use that one instead. `discovery` is what is left over, which is exactly why it is so large.
- **Required content sections**: *What*, *Why it matters*, *Where*.
- **Topic-key pattern**: `<domain>/<subject>`, reusing the domain the finding belongs to — e.g. `build/gradle-daemon-lock`
- **Example**:

  > **Title**: Gradle daemon keeps the library lock after a failed export
  > **What**: When an export fails mid-write, the daemon holds the build directory locked and the next build hangs with no error at all.
  > **Why it matters**: The hang reads as an engine bug and costs an hour. Stopping the daemon is the fix, and the two look unrelated.
  > **Where**: project build settings; any CI job that exports before building.

#### `architecture`

High-level structural decisions about the system — packages, boundaries, data flow, module contracts. Added in the `adopt-thoughtline-replace-engram` migration — maps 1:1 from Engram's `architecture` type.

- **When to use**: system-level knowledge that every developer touching the project needs: package boundaries, interface contracts, deployment constraints, dependency rules.
- **Required content sections**: *What*, *Why*, *Where* (affected packages/files), *Learned* (gotchas).
- **Topic-key pattern**: `architecture/<area>` — e.g. `architecture/storage-layer`
- **Scope**: MUST be `project`.
- **Example**:

  > **Title**: Storage layer owns all SQL; domain layer never imports `database/sql`
  > **Why**: Keeps the domain (`internal/memory`) free of persistence concerns, testable without DB, and replaceable without touching business logic.
  > **Where**: `internal/storage`, `internal/memory`
  > **Learned**: `storage.Save()` must return the full populated `Memory` struct so callers never need to re-query.

#### `bugfix`

Bug + root cause + fix, with engine/platform context.

- **When to use**: any non-trivial bug, especially if it took >30 min to track down.
- **Required content sections**: *Symptom*, *Root cause*, *Fix*, *Why we were wrong* (the diagnostic step that misled).
- **Topic-key pattern**: usually omitted (each bug is unique). Use `bug/<area>` if you want a stable handle.

#### `decision`

An architectural or product decision and the reasoning behind it. Added in the `adopt-thoughtline-replace-engram` migration — maps 1:1 from Engram's `decision` type.

- **When to use**: any time a team or solo dev locks in a technical or design approach with a "why" that matters across sessions. Distinct from `game-design-decision` (which is gameplay-facing) — use this for infrastructure, tooling, and cross-cutting concerns.
- **Required content sections**: *What*, *Why*, *Alternatives considered*, *Date*.
- **Topic-key pattern**: `decision/<area>/<choice>` — e.g. `decision/auth/jwt-vs-session`
- **Scope**: MUST be `project`.
- **Example**:

  > **Title**: Use WAL journal mode for all SQLite connections
  > **Why**: WAL allows concurrent readers + one writer; eliminates the "database is locked" errors we saw with DELETE mode under the MCP server's concurrent tool calls.
  > **Alternatives considered**: DELETE mode (rejected — lock contention), in-memory (rejected — no persistence).

#### `config`

A configuration, flag, path, version pin, or environment fact whose **exact value** has to be reproduced.

- **When to use**: when the value itself *is* the knowledge. If a wrong value fails loudly and immediately, you probably do not need a memory; if it fails silently, or fails months later on someone else's machine, you do.
- **Required content sections**: *What*, *Where*, *Why this value*.
- **Topic-key pattern**: `config/<area>` — e.g. `config/ci/gradle-jvm-args`
- **Example**:

  > **Title**: CI builds need a 4 GB heap for the Gradle JVM
  > **What**: `org.gradle.jvmargs=-Xmx4g` in `gradle.properties`.
  > **Where**: `gradle.properties` at the project root; the Android export job reads it.
  > **Why this value**: 2 GB runs out of memory during IL2CPP linking on the runner. 4 GB is the smallest value that completes; 8 GB wastes runner memory and slows the job down.

#### `pattern`

A reusable solution shape — a way of solving a recurring problem, independent of the file it first appeared in.

- **When to use**: you have written the same arrangement twice, or you are about to. Capture it once instead of reinventing it.
- **Required content sections**: *Pattern*, *When to use it*, *When NOT to use it*.
- **Topic-key pattern**: `pattern/<area>` — e.g. `pattern/async/cancellation-tokens`
- **Example**:

  > **Pattern**: One cancellation source per request, disposed by the handler that created it.
  > **When to use it**: any handler that fans out to several calls and must abandon all of them together.
  > **When NOT to use it**: a background loop that outlives the request — it needs its own lifetime, not the request's.

#### `preference` *(scope = personal)*

Per-developer ergonomics. **`scope` MUST be `personal` for this type.**

- **When to use**: editor settings, formatting preferences, individual workflow quirks that travel with you.
- **Topic-key pattern**: `preference/<area>` — e.g. `preference/keybindings`

### Extension types

This repository's own additions, for game projects. Measured use across the same 2,336 observations: **4 uses in total** — `convention` 3, `perf-gotcha` 1, and zero for `game-design-decision`, `scene-pattern`, `asset-reference`, `pipeline-step` and `script-pattern`.

That is the honest summary. A game project may still want them and nothing here says otherwise, but nothing measured has needed one yet — so treat this tier as available rather than recommended. Adopt a type only if your domain actually asks for it.

#### `game-design-decision`

A design choice and the reasoning behind it.

- **When to use**: any time a designer or dev locks in a behavior, mechanic, or constraint with a "why".
- **Required content sections**: *Decision*, *Why*, *Alternatives considered* (even briefly).
- **Topic-key pattern**: `design/<system>/<choice>` — e.g. `design/inventory/grid-vs-list`
- **Example**:

  > **Title**: Lock player loot UI to a 4×6 grid
  > **Why**: Original list view caused cognitive load on mobile (test sessions Apr 12). Grid keeps icons visible at thumb-reach.
  > **Alternatives considered**: scrollable list (rejected — bad on mobile), radial menu (rejected — too novel for our audience).

#### `scene-pattern`

The game-dev specialisation of the core `pattern` type: a recurring entity hierarchy or component setup in your engine.

- **When to use**: anytime you find yourself building the same arrangement twice. Capture it once so you don't reinvent it.
- **Engine-specific fields**: `tags = ["engine:playcanvas"]` or whichever engine.
- **Topic-key pattern**: `scene/<engine>/<pattern>` — e.g. `scene/playcanvas/interactive-prop`
- **Example**: A "world/static" vs "world/interactive" split with rationale, plus the typical components on each branch.

#### `asset-reference`

Path, version, import settings, and origin of a model/texture/sound/font/shader.

- **When to use**: when an asset's import settings or origin matter for reproducibility.
- **Required content sections**: *Path*, *Source* (DCC tool, vendor, license), *Import settings*, *Reason for these settings*.
- **Topic-key pattern**: `asset/<category>/<name>` — e.g. `asset/texture/lantern-base`
- **Example**:

  > **Title**: lantern_base.png — bloom-safe import
  > **Path**: `assets/textures/props/lantern_base.png`
  > **Source**: Substance Painter export, original PSD in `Dropbox/.../lantern_v3.psd`
  > **Import settings**: sRGB, mip-on, max 1024, no premultiplied alpha
  > **Why**: max-2048 caused bloom blowout on the inn cellar scene at night.

#### `perf-gotcha`

Performance traps you only learn by hitting them.

- **When to use**: drawcall budgets, GC pauses, batching breakage, asset-size cliffs, frame-time spikes.
- **Required content sections**: *Symptom*, *Root cause*, *Fix*, *Threshold/budget if any*, *Platform*.
- **Tag with**: platform (`android`, `ios`, `webgl`), engine.
- **Topic-key pattern**: `perf/<platform>/<area>` — e.g. `perf/android/static-batching`
- **Example**:

  > **Title**: Static batching breaks above 187 chairs in inn scene (Android)
  > **Symptom**: frame-time doubles in cellar room
  > **Root cause**: PlayCanvas batch size cap — over the threshold, batches split and drawcalls spike
  > **Fix**: cap chair count per batch group at 180; budget 7 chairs of headroom for moving stock

#### `pipeline-step`

A reproducible step in your asset/build pipeline.

- **When to use**: anything that must be done the same way every time. The "tribal knowledge" candidates.
- **Required content sections**: *Inputs*, *Tool / version*, *Steps (numbered)*, *Outputs*, *Verification*.
- **Topic-key pattern**: `pipeline/<source>-to-<target>/<asset-type>` — e.g. `pipeline/blender-to-pc/animated-mesh`
- **Example**:

  > **Title**: Blender → PlayCanvas animated mesh export
  > **Tool**: Blender 4.2.1 + PlayCanvas Asset Pipeline 2.7
  > **Steps**: ... (numbered)
  > **Verification**: open in PC editor, check vertex count matches Blender stat panel ± 1%, check animation FPS = 30.

#### `script-pattern`

The engine-script specialisation of the core `pattern` type: an idiom worth remembering.

- **When to use**: a clean way you found to handle a recurring script need (component lifecycle, event wiring, pooling, ...).
- **Required content sections**: *Pattern*, *When to use it*, *When NOT to use it*, *Code skeleton*.
- **Topic-key pattern**: `script/<engine>/<concept>` — e.g. `script/playcanvas/component-pooling`

#### `convention`

Naming, structure, project-wide rules.

- **When to use**: anything a new developer joining the project should learn on day one.
- **Required content sections**: *The rule*, *Why*, *Where it applies*.
- **Topic-key pattern**: `convention/<area>` — e.g. `convention/asset-naming`

---

## How the vocabulary is enforced

There is no validator any more. Engram accepts any `type` string — a deliberate design choice on its part, and precisely the reason this vocabulary earns its place. **Nothing stops your agent from inventing `perf_bugfix_thing` except being told not to.**

So these rules are a contract with your agent, not a gate. What makes them real is [`presets/AGENTS.md`](../../presets/AGENTS.md) — a file you copy into your project so that this catalogue sits where your agent reads its instructions. Step 3 of the adoption path in the [README](../../README.md) is that copy step.

**This file is the only complete copy of the catalogue.** Every other document — the README, the editor guides, the comparison — names a type or two as an example and links here. An earlier revision repeated the full list across **fifteen files, this one included**, and by the time anyone checked, the list was wrong in all of them at once. (The archived `openspec/` proposals carry it too; those are a frozen record and were left alone, along with the retired engine's own copy in `internal/memory/`.) Change the catalogue here, and here only.

| Rule | What you lose if the agent drifts |
| ---- | --------------------------------- |
| `type` is one of the 14 catalogue values; a new type needs an issue first | Free-form types accumulate until `type` is noise — the exact failure this vocabulary exists to prevent |
| `scope` is `project` (the default), `personal` or `global`; `preference` **must** be `personal`, and everything else stays `project` unless it genuinely applies to every project | Memories leak across projects, or hide from the project that needs them |
| `title` is non-empty and ≤ 200 chars, short and searchable | Titles stop working as an index and search results read as a wall of sentences |
| `content` is non-empty and self-contained | A memory the next session cannot act on is worse than no memory — it looks like knowledge |
| `topic_key`, when present, matches `^[a-z0-9][a-z0-9/_.-]{1,128}$` — lowercase, no spaces, no leading slash, dots only for version numbers | Nothing breaks loudly; the key simply stops being greppable and consistent |
| Tags on the `**Tags**:` line match `^[a-z0-9][a-z0-9:_-]{0,40}$`, lowercase, `key:value` for namespaced tags | Tags misspell themselves into invisibility |
| Keep `content` well under 64 KiB | The retired engine rejected oversize content with a clear error; Engram does not, so this is a writing guideline now. Check Engram for its own limits |

---

## What your choice of `type` costs

`type` is not only a label. Engram keys one piece of lifecycle behaviour on the exact string, and
this catalogue is not shaped to match it.

Every observation can carry a `review_after` timestamp. Once that passes, the observation reports
`state: "needs_review"` in search results and turns up under `mem_review` with `action: "list"`.
**That horizon exists for exactly three type strings** — `decision` (six months), `policy`
(twelve) and `preference` (three) — and Engram's own comment above the map states the rest:
*"Types absent from this map get `review_after` = NULL (Phase 1 behavior)."*

So **twelve of this catalogue's fourteen types never surface as stale and never appear in
`mem_review`**: everything except `decision` and `preference`. (`policy` is one of the three keys,
and this catalogue does not use it — a quiet reminder that the map was not written for us.)

This does not make the catalogue wrong. It means the review tools are not the whole of memory
hygiene here: for the other twelve types, deciding that a memory has aged out is your judgement,
usually expressed as a `mem_update` or a `mem_delete`. Engram's review machinery is real, and it
covers two of our fourteen types.

See [ADR 0007](../decisions/0007-vocabulary-not-mechanics.md) for why this vocabulary is
deliberately not reshaped to fit that map, and what is being asked upstream instead.

---

## Adding a new memory type

1. Open an issue describing the use case and at least three real examples.
2. Discuss in the issue whether an existing type already covers it. Most often it does — the catalogue is deliberately small.
3. If a new type is justified, add a section to this file following the pattern above: purpose, required content sections, topic-key pattern, a concrete example.
4. If it belongs in the **core** tier, add it to the table in [`presets/AGENTS.md`](../../presets/AGENTS.md) as well. That file is copied into other projects, so it carries its own copy of the core list by design — this file and that one are the only two copies of the catalogue.
5. Bump the CHANGELOG.

New types are additive and never breaking, because Engram stores the type string verbatim. Adding one is a documented convention rather than a schema migration — which is the whole advantage of no longer owning the engine.

---

## What we deliberately leave out

- **Free-form `type` — as an engine feature.** Engram allows it, which is right for a general-purpose tool. This vocabulary closes the catalogue by convention instead, so the words stay shared.
- **Hierarchical types**. No subtypes. If you feel the pull toward `bugfix.android.batching`, use `tags` instead.
- **Per-type custom JSON schemas**. The shared envelope plus tags + markdown content is enough for v1. If a type really needs structured data, that becomes its own ADR.
