# Memory domain — taxonomy, fields, conventions

This is the canonical vocabulary of Thoughtline. It defines what a "memory" is, what types it can take, and how each type is shaped. The content here directly governs the validators in `internal/memory` and the schema in `internal/storage`.

If you are adding a new type, follow [Adding a new memory type](#adding-a-new-memory-type) at the bottom.

---

## The shared envelope

Every memory — regardless of type — carries the same envelope.

| Field            | Type     | Required | Notes                                                                       |
| ---------------- | -------- | -------- | --------------------------------------------------------------------------- |
| `id`             | int      | (auto)   | Local autoincrement primary key                                              |
| `sync_id`        | string   | (auto)   | UUIDv7. Stable across upserts. The "real" identifier                         |
| `project`        | string   | yes      | Identifier of the active project. Defaults to working dir basename          |
| `scope`          | enum     | yes      | `project` (default) or `personal`                                            |
| `type`           | enum     | yes      | One of the values listed below                                              |
| `topic_key`      | string   | no       | Stable key for evolving topics. Same project + same key → upsert            |
| `title`          | string   | yes      | Short, searchable. Imperative form preferred                                |
| `content`        | string   | yes      | Markdown body. No hard cap; oversize content rejected with a clear error    |
| `tags`           | []string | no       | Free-form tags (engine name, platform, etc.)                                |
| `revision_count` | int      | (auto)   | Bumped on every upsert; `0` for first save                                  |
| `created_at`     | int (ms) | (auto)   | First time this topic appeared                                              |
| `updated_at`     | int (ms) | (auto)   | Last time this topic was upserted                                           |
| `deleted_at`     | int (ms) | (auto)   | Soft delete                                                                  |

### About `topic_key`

This is the cleverest concept Thoughtline borrows from Engram. A `topic_key` is a stable, readable string that names *what this memory is about* — like a slug for a wiki article.

When a user (via the AI) calls `tl_save` with a `topic_key`, Thoughtline:

1. Looks up `(project, topic_key)`.
2. If a row exists, **upserts**: same `sync_id`, same `created_at`, but new `content`/`updated_at`/`revision_count + 1`.
3. If no row exists, **inserts**: new `sync_id`, fresh `created_at`, `revision_count = 0`.

This means the same evolving topic accumulates revisions instead of duplicates. Recommended `topic_key` shape: `category/subject` or `category/subcategory/subject`. Examples:

- `architecture/inn-entity-hierarchy`
- `pipeline/blender-to-pc/lantern-import`
- `perf/android/chair-batching`
- `convention/script-naming`

The same convention is what makes [the topic-key shortcut in search](../decisions/0002-search-strategy-fts5-first.md) work: queries containing `/` are matched against `topic_key` first.

### About `scope`

- `project` (default) — bound to a single project. Most memories live here.
- `personal` — cross-project, per-developer. Use sparingly, for ergonomics ("I prefer 4-space indents in shaders") that travel with the dev, not the project.

---

## The type catalogue

### `game-design-decision`

A design choice and the reasoning behind it.

- **When to use**: any time a designer or dev locks in a behavior, mechanic, or constraint with a "why".
- **Required content sections**: *Decision*, *Why*, *Alternatives considered* (even briefly).
- **Topic-key pattern**: `design/<system>/<choice>` — e.g. `design/inventory/grid-vs-list`
- **Example**:

  > **Title**: Lock player loot UI to a 4×6 grid
  > **Why**: Original list view caused cognitive load on mobile (test sessions Apr 12). Grid keeps icons visible at thumb-reach.
  > **Alternatives considered**: scrollable list (rejected — bad on mobile), radial menu (rejected — too novel for our audience).

### `scene-pattern`

A recurring entity hierarchy or component setup in your engine.

- **When to use**: anytime you find yourself building the same arrangement twice. Capture it once so you don't reinvent it.
- **Engine-specific fields**: `tags = ["engine:playcanvas"]` or whichever engine.
- **Topic-key pattern**: `scene/<engine>/<pattern>` — e.g. `scene/playcanvas/interactive-prop`
- **Example**: A "world/static" vs "world/interactive" split with rationale, plus the typical components on each branch.

### `asset-reference`

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

### `perf-gotcha`

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

### `pipeline-step`

A reproducible step in your asset/build pipeline.

- **When to use**: anything that must be done the same way every time. The "tribal knowledge" candidates.
- **Required content sections**: *Inputs*, *Tool / version*, *Steps (numbered)*, *Outputs*, *Verification*.
- **Topic-key pattern**: `pipeline/<source>-to-<target>/<asset-type>` — e.g. `pipeline/blender-to-pc/animated-mesh`
- **Example**:

  > **Title**: Blender → PlayCanvas animated mesh export
  > **Tool**: Blender 4.2.1 + PlayCanvas Asset Pipeline 2.7
  > **Steps**: ... (numbered)
  > **Verification**: open in PC editor, check vertex count matches Blender stat panel ± 1%, check animation FPS = 30.

### `script-pattern`

An engine-script idiom worth remembering.

- **When to use**: a clean way you found to handle a recurring script need (component lifecycle, event wiring, pooling, ...).
- **Required content sections**: *Pattern*, *When to use it*, *When NOT to use it*, *Code skeleton*.
- **Topic-key pattern**: `script/<engine>/<concept>` — e.g. `script/playcanvas/component-pooling`

### `bugfix`

Bug + root cause + fix, with engine/platform context.

- **When to use**: any non-trivial bug, especially if it took >30 min to track down.
- **Required content sections**: *Symptom*, *Root cause*, *Fix*, *Why we were wrong* (the diagnostic step that misled).
- **Topic-key pattern**: usually omitted (each bug is unique). Use `bug/<area>` if you want a stable handle.

### `convention`

Naming, structure, project-wide rules.

- **When to use**: anything a new developer joining the project should learn on day one.
- **Required content sections**: *The rule*, *Why*, *Where it applies*.
- **Topic-key pattern**: `convention/<area>` — e.g. `convention/asset-naming`

### `preference` *(scope = personal)*

Per-developer ergonomics. **`scope` MUST be `personal` for this type.**

- **When to use**: editor settings, formatting preferences, individual workflow quirks that travel with you.
- **Topic-key pattern**: `preference/<area>` — e.g. `preference/keybindings`

---

## Validation rules

Enforced by `internal/memory`:

1. `type` must be one of the catalogue values above. Unknown types are rejected.
2. `scope` must be `project` or `personal`. `preference` requires `personal`; everything else requires `project`.
3. `title` is non-empty and ≤ 200 chars.
4. `content` is non-empty. There is no hard upper bound, but `internal/storage` returns a clear error (not silent truncation) if `len(content) > MaxContentBytes` (default 64 KiB, configurable).
5. `topic_key`, when present, matches `^[a-z0-9][a-z0-9/_-]{1,128}$`. No spaces, no uppercase, no leading slash.
6. `project` is non-empty. Whitespace trimmed.
7. `tags`, when present, each match `^[a-z0-9][a-z0-9:_-]{0,40}$`. Lowercase only. Convention: `key:value` for engine/platform tags (`engine:playcanvas`, `platform:android`).

---

## Adding a new memory type

1. Open an issue describing the use case and at least three real examples.
2. Discuss in the issue whether an existing type covers it (most often, yes).
3. If a new type is justified, write a section in this file matching the pattern above (purpose, required content sections, topic-key pattern, example).
4. Add the type constant to `internal/memory`.
5. Add a validation test.
6. Mention the new type in the README's preview table.
7. Bump CHANGELOG.

New types are minor additions (additive), not breaking changes. No ADR required unless the new type needs new schema columns.

---

## What we deliberately leave out

- **Free-form `type`**. Locking the catalogue down is a feature. It keeps recall predictable and prevents the AI from inventing a new type on every save.
- **Hierarchical types**. No subtypes. If you feel the pull toward `bugfix.android.batching`, use `tags` instead.
- **Per-type custom JSON schemas**. The shared envelope plus tags + markdown content is enough for v1. If a type really needs structured data, that becomes its own ADR.
