# Memory domain — the vocabulary

This is the canonical vocabulary of Thoughtline: what a "memory" is, which types it can take, and how each type is shaped.

The engine enforces this vocabulary, and both are in this repository. `tl_save` runs every memory through the domain layer in [`internal/memory`](../../internal/memory) before it reaches the store, so a save carrying a type outside the catalogue, a malformed `topic_key`, a bad tag or an empty title is **rejected** rather than filed. This page is that catalogue: eight fields you set, fourteen types, and the rules the server checks.

[Engram](https://github.com/Gentleman-Programming/engram) is the other engine that speaks this vocabulary, and this repository ships [`cmd/migrate`](../../cmd/migrate/README.md) to move memories out of it. Where the two differ, this page names the difference instead of averaging it. [ADR 0008](../decisions/0008-the-engine-is-the-product.md) records why the engine came back.

If you are adding a new type, follow [Adding a new memory type](#adding-a-new-memory-type) at the bottom.

---

## The envelope

Every memory — regardless of type — carries the same envelope. You set eight fields; the store owns the rest.

**You set these:**

| Field        | Required | Notes                                                                                            |
| ------------ | -------- | ------------------------------------------------------------------------------------------------ |
| `title`      | yes      | Short, searchable. Imperative form preferred. ≤ 200 characters                                     |
| `content`    | yes      | Markdown body — this is where the per-type sections below live                                  |
| `type`       | yes      | One of the fourteen catalogue values below. Anything else is refused, never folded to a default   |
| `topic_key`  | no       | Stable key for evolving topics — see below                                                      |
| `scope`      | no       | `project` (default) or `personal` — see below                                                    |
| `project`    | no       | Defaults to the working directory's basename when omitted                                        |
| `tags`       | no       | A real column, and `tl_search` filters on it — see below                                        |
| `session_id` | no       | Attaches the memory to an open session. Must be a UUIDv7 for a session in the same project       |

**The store owns these — never write them by hand:** `id`, `sync_id`, `brain_id`, `normalized_hash`, `revision_count`, `created_at`, `updated_at`, `deleted_at`. They are what makes provenance and upserts work.

### Where tags go

`tags` is a real column, and `tl_search` filters on it, so a tag you expect to query by belongs in the field. The shape is the one [`tag-conventions.md`](tag-conventions.md) defines: lowercase letters, digits, `:`, `_` and `-`, between 1 and 41 characters, starting with a letter or a digit, `key:value` for the namespaced form. Anything else is rejected rather than quietly dropped.

Two other places a fact can carry a tag:

1. **Inside the `topic_key`**, when the type's key pattern has a slot for it. `perf/android/static-batching` already carries `platform:android`; `scene/playcanvas/interactive-prop` already carries `engine:playcanvas`. This is identity rather than metadata — the key is what an upsert matches on, which is why a platform or an engine the type has a slot for goes here.
2. **On a `**Tags**:` first line inside `content`**, the convention this vocabulary used before the column existed. It is indexed as body text, so those tags stay findable, but they cannot be filtered.

> **Which one to use: the field.** It survives an edit, it is what `tl_search` filters on, and `tl_get_observation` prints it back. The first line still works and is still what a human reading raw markdown sees, but it is not a second source of truth — a tag that matters belongs in the field.

---

### About `topic_key`

A `topic_key` is a stable, readable string naming *what this memory is about* — a slug for a wiki article. It is the store's own concept, and this vocabulary is built on it.

Pass a `topic_key` to `tl_save` and the server upserts on `(project, topic_key)`:

1. **First write — inserts.** New `id`, new `sync_id`, `created_at` stamped, `revision_count` 0.
2. **Every later write with the same key — updates in place.** *Same* `id` and *same* `sync_id`, `created_at` preserved, `updated_at` bumped, `revision_count` + 1.

Measured against this server: two saves sharing a key came back with the same `id` and the same `sync_id`, the first save's `created_at` survived the second (the saves were a second apart, so the timestamps differ visibly), and the revision count went 0, then 1.

> **Gotcha — the upsert replaces, it does not merge.** The later write's `title`, `content` and `tags` all overwrite the earlier ones and the previous text is gone. A keyed memory is a *topic*, not a log: write what is currently true, not a changelog. If you want history, leave `topic_key` unset, or keep the history inside `content`.

Recommended `topic_key` shape: `category/subject` or `category/subcategory/subject`. Examples:

- `architecture/inn-entity-hierarchy`
- `pipeline/blender-to-pc/lantern-import`
- `perf/android/chair-batching`
- `convention/script-naming`

A dot is allowed, and its one intended use is a version number: `audit/v0.0.1-features-apagadas`, `design/gdd/crowd-control-v1.1`. That is what the dotted keys in real stores are for, and it is the one case where the key itself should record which revision of a thing you are describing.

A query containing `/` is treated as a key rather than as text. It is matched against `topic_key` as a GLOB pattern exactly as written, then retried as a prefix, and only if both attempts find nothing does full-text search run. So `architecture/inn` finds `architecture/inn-entity-hierarchy`, `architecture/` finds every key under it, and `design/auth/*` says so explicitly. A query with no `/` never reaches this path, which is why keys are shaped `category/subject` rather than a bare subject — see [ADR 0002](../decisions/0002-search-strategy-fts5-first.md).

### About `scope`

- `project` (default) — bound to a single project. Most memories live here.
- `personal` — cross-project, per-developer. Use sparingly, for ergonomics ("I prefer 4-space indents in shaders") that travel with the dev, not the project.
`preference` **must** be `personal`, and every other type must be `project`. The server checks that pairing and refuses a mismatch, so it is not a style rule you can drift from.

There is no third value. Engram's store carries a `global` scope — two orders of magnitude rarer than `project` in a real store, spread thinly across six projects — and this catalogue used to name it. This server does not accept it: `scope` is a stored and filterable attribute, not a visibility rule, and `personal` already means "cross-project, for this developer". A third value would be a label with no behaviour behind it, so the field stays at two and a `global` save is refused by a message that says so. See [ADR 0008](../decisions/0008-the-engine-is-the-product.md).

---

## The type catalogue

Fourteen types in two tiers.

**Core** — the seven the agent ecosystem already teaches. Every session on this stack is instructed with `type: bugfix | decision | architecture | discovery | pattern | config | preference`, whether or not this repository exists. Where a catalogue and that instruction disagree, an agent obeys whichever it read last and both lose their point. So the core tier is chosen to **agree with it**, not to be interesting.

**Extension** — this repository's own additions, aimed at game projects. Optional: adopt one only if your domain asks for it.

Engram's own tools write their own types, and those belong to neither tier: `session_summary` and `manual` — the field's default when a caller passes no type at all. In the store this catalogue was measured against, `session_summary` alone held 459 observations and `manual` held 91. Neither is in the list below, so a migration has to say what each of them *was*; `manual` in particular is refused outright, which is the right outcome — a memory filed under "miscellaneous" is a memory nobody will find again. This catalogue governs the memories *you* decide to save.

### Core types

Ordered by measured use across 2,336 observations in 13 projects, read from a real Engram store on 2026-09-29. The order is evidence, not taste — and a snapshot, not a constant.

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

By code. `tl_save` validates every field before it writes, and a memory that fails is refused with a message naming the field and the rule. Nothing is filed under a default, so no wrong memory reaches the store for a later session to trust. The rules live in [`internal/memory`](../../internal/memory): `Validate` in `validate.go`, the catalogue in `types.go`, the limits beside them.

| Rule | What you lose if it drifts |
| ---- | -------------------------- |
| `type` is one of the fourteen catalogue values | A free-form type accumulates until `type` is noise — the exact failure this vocabulary exists to prevent |
| `scope` is `project` (the default) or `personal`; `preference` must be `personal` and everything else must be `project` | Memories leak across projects, or hide from the project that needs them |
| `title` is non-empty and ≤ 200 characters, counted in runes | Titles stop working as an index and search results read as a wall of sentences |
| `content` is non-empty and ≤ 64 KiB, counted in bytes | A memory the next session cannot act on is worse than no memory — it looks like knowledge |
| `topic_key`, when present, matches `^[a-z0-9][a-z0-9/_.-]{1,128}$` — lowercase, no spaces, no leading slash, dots only for version numbers | Nothing breaks loudly: the key stops being greppable, and the `/` shortcut stops finding it |
| A tag matches `^[a-z0-9][a-z0-9:_-]{0,40}$` — lowercase, digits, `key:value` for namespaced tags | Tags misspell themselves into invisibility, and a filter matches nothing without saying so |
| `session_id`, when present, is a UUIDv7 for a session in the same project | The memory attaches to a session it does not belong to |

**This file is the catalogue; [`internal/memory/types.go`](../../internal/memory/types.go) is the list.** They are two copies of one claim, which is why the server builds every user-facing type list — the `tl_save` and `tl_search` descriptions, and the message a rejected save returns — from `AllTypes()` at startup instead of typing it out again. Three hand-typed copies existed and all three had gone stale in different ways; one still called `decision` and `architecture` invalid. An earlier revision of this page repeated the catalogue across **fifteen files**, and by the time anyone checked, the list was wrong in all of them at once. Change the catalogue here and in `types.go`, and nowhere else. (The archived `openspec/` proposals carry it too; those are a frozen record and were left alone.)

---

## What your choice of `type` costs

`type` is a label with a closed set of values, and that is the whole of it here: it is stored, it is
returned by `tl_get_observation`, and `tl_search` filters on it by exact match. Nothing else in this
server is keyed on the string. There is no decay, no review horizon and no automatic
re-surfacing.

On Engram, an observation can carry a `review_after` timestamp. Once that passes it reports
`state: "needs_review"` in search results and turns up under `mem_review` with `action: "list"`.
**That horizon exists for exactly three type strings** — `decision` (six months), `policy`
(twelve) and `preference` (three) — and that engine's own comment above the map states the rest:
*"Types absent from this map get `review_after` = NULL (Phase 1 behavior)."*

So on that engine twelve of this catalogue's fourteen types never surface as stale: everything
except `decision` and `preference`. (`policy` is one of the three keys, and this catalogue does not
use it — a quiet reminder of who the map was written for.)

This repository's engine implements no horizon, so here the whole of memory hygiene is the writer's
job: a memory that has aged out is corrected with `tl_update` or retired with `tl_delete`, and
`tl_stats` shows what is there. If this catalogue ever grows a horizon, it should be decided on this
repository's terms and recorded in an ADR — not inherited from a map that was not written for us.
[ADR 0008](../decisions/0008-the-engine-is-the-product.md) records why the vocabulary was left
alone instead of being reshaped to fit that map.

---

## Adding a new memory type

1. Open an issue describing the use case and at least three real examples.
2. Discuss in the issue whether an existing type already covers it. Most often it does — the catalogue is deliberately small.
3. If a new type is justified, add it in these places, in one change:
   - a section to this file following the pattern above: purpose, required content sections, topic-key pattern, a concrete example;
   - the constant and its `AllTypes()` entry in [`internal/memory/types.go`](../../internal/memory/types.go);
   - the table in [`presets/AGENTS.md`](../../presets/AGENTS.md), if it belongs to the **core** tier. That file is copied into other projects, so it carries its own copy of the core list by design.
4. Bump the CHANGELOG.
5. If the new type should not be coupled to `scope` the way `preference` is, that is a change to `Validate` too — say so in the issue.

Everything else follows on its own: the tool descriptions and the rejection message are built from `AllTypes()` at startup, so a type added to the list is offered by the server without a second edit. Adding a type stays additive and non-breaking, and it is now a change to this repository — the code and the catalogue move together.

---

## What we deliberately leave out

- **Free-form `type`.** Engram allows any string, which is right for a general-purpose tool. This server closes the set, so a save carrying an unknown type is refused instead of stored — and the words stay shared because nothing can invent a new one silently.
- **Hierarchical types**. No subtypes. If you feel the pull toward `bugfix.android.batching`, use `tags` instead — that is what the column is for.
- **Per-type custom JSON schemas**. The shared envelope plus tags + markdown content is enough for v1. If a type really needs structured data, that becomes its own ADR.
