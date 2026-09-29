# Thoughtline — memory instructions for your agent

**Copy this file into your project** as `AGENTS.md` (or paste it into `CLAUDE.md`, `.cursor/rules/*`, a skill, or whatever your tool reads at session start) and tell your agent to follow it.

This file is the whole adoption path. Engram accepts **any string** as a memory `type` and has **no validator** — so nothing stops your agent from inventing `perf_bugfix_thing` except being told not to. A vocabulary the agent never reads is a vocabulary that does not exist.

---

## 1. Save memories proactively

Do not wait to be asked. Call `mem_save` when any of these happens:

- a design or technical decision gets locked in, with a reason
- a bug takes more than ~30 minutes to track down
- you hit a performance trap you would not have guessed
- an asset's import settings or source turn out to matter for reproducibility
- you build the same scene arrangement or script idiom for the second time
- you learn a pipeline step that must be done the same way every time

One good memory beats five fragments. If a memory with the same `topic_key` already exists, `mem_update` it rather than saving a near-duplicate.

## 2. Pick exactly one `type`

| `type` | Use it for | `topic_key` pattern |
| --- | --- | --- |
| `game-design-decision` | A gameplay or design choice and its "why" | `design/<system>/<choice>` |
| `scene-pattern` | A recurring entity hierarchy or component setup | `scene/<engine>/<pattern>` |
| `asset-reference` | An asset's path, source, import settings, and why | `asset/<category>/<name>` |
| `perf-gotcha` | A performance trap you only learn by hitting it | `perf/<platform>/<area>` |
| `pipeline-step` | A reproducible asset or build pipeline step | `pipeline/<source>-to-<target>/<kind>` |
| `script-pattern` | An engine-script idiom worth reusing | `script/<engine>/<concept>` |
| `bugfix` | A bug, its root cause, and the fix | usually omit it; `bug/<area>` if it recurs |
| `convention` | A project-wide naming or structure rule | `convention/<area>` |
| `preference` | Per-developer ergonomics — **`scope` MUST be `personal`** | `preference/<area>` |
| `decision` | A technical or product decision with cross-session weight | `decision/<area>/<choice>` |
| `architecture` | System structure: packages, boundaries, contracts | `architecture/<area>` |

If nothing fits, **pick the closest one — do not invent a type.** An invented type is invisible to every future search by category, which is the exact failure this vocabulary exists to prevent. If a type is genuinely missing, open an issue; new types are additive.

> Engram's own tools write their own types (`session_summary`, for example). That is fine and out of scope — this table governs the memories *you* decide to save.

## 3. Shape the content

Every memory carries at least these four lines:

```
**What**: one sentence — what was done or learned.
**Why**: what motivated it.
**Where**: the files, paths, scenes or packages it touches.
**Learned**: gotchas, surprises, what you would do differently.
```

Types that ask for more, get more: `game-design-decision` wants *Alternatives considered*; `perf-gotcha` wants *Symptom* / *Root cause* / *Fix*; `pipeline-step` wants numbered *Steps* plus a *Verification*; `asset-reference` wants *Path* / *Source* / *Import settings* / *Reason*.

Write for the next session, not for the current one. **A memory nobody can act on is worse than no memory — it looks like knowledge.**

## 4. Tags have no field

Engram has **no tags field**. Tags have exactly two places to live:

1. **In the `topic_key`**, when its pattern has a slot for it. `perf/android/static-batching` already carries `platform:android`; `scene/godot/inventory-ui` already carries `engine:godot`.
2. **On a `**Tags**:` line as the first line of `content`** — comma-separated, lowercase, `key:value`:

```
**Tags**: engine:unity, asset:texture, pipeline:fbx, perf:memory

**Symptom**: ...
```

Write that line **always**, even when the `topic_key` already implies one of its tags. It costs a line, it survives a key that later drifts, and for `phase:`, `tool:`, `perf:`, `asset:`, and any type whose pattern has no engine slot, it is the only home there is.

Namespaces: `engine:` · `platform:` · `pipeline:` · `asset:` · `phase:` · `tool:` · `perf:`. The canonical list is in [`docs/design/tag-conventions.md`](../docs/design/tag-conventions.md).

Tags stay **searchable** because Engram's full-text search indexes the body. You cannot *filter* by tag — treat the line as an aid to recall, not a database index.

## 5. Hard rules

These are not style preferences. Each one costs you something concrete when it drifts.

| Rule | What you lose if it drifts |
| --- | --- |
| `title` is non-empty and ≤ 200 characters | Titles stop working as an index; search results read as a wall of sentences |
| `content` is non-empty and self-contained | The memory looks like knowledge but the next session cannot act on it |
| **Always pass `project`** | The memory becomes invisible to every per-project recall |
| `scope` is `project` or `personal`; `preference` **must** be `personal`, everything else `project` | Memories leak across projects, or hide from the project that needs them |
| `topic_key`, when present, is lowercase with no spaces and no leading slash (`^[a-z0-9][a-z0-9/_-]{1,128}$`) | Nothing breaks loudly — the key quietly stops being greppable |
| Tags match `^[a-z0-9][a-z0-9:_-]{0,40}$`, lowercase | Tags misspell themselves into invisibility |
| Keep `content` well under 64 KiB | Engram does not enforce a limit; very large memories degrade search and recall |

---

## Where the rest lives

- The full type catalogue, with required sections and worked examples: [`docs/design/memory-domain.md`](../docs/design/memory-domain.md)
- The tag namespaces: [`docs/design/tag-conventions.md`](../docs/design/tag-conventions.md)
- Why this repository owns the vocabulary and not the tool mechanics: [ADR 0007](../docs/decisions/0007-vocabulary-not-mechanics.md)
