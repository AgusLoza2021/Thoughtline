# Thoughtline — memory instructions for your agent

**Copy this file into your project** as `AGENTS.md` (or paste it into `CLAUDE.md`, `.cursor/rules/*`, a skill, or whatever your tool reads at session start) and tell your agent to follow it.

This file is the whole adoption path. The server validates every save against the catalogue below, so an invented type is a failed save rather than a memory nobody will find again — but a vocabulary the agent never reads is still a vocabulary that does not exist.

---

## 1. Save memories proactively

Do not wait to be asked. Call `tl_save` when any of these happens:

- a design or technical decision gets locked in, with a reason
- a bug takes more than ~30 minutes to track down
- you hit a performance trap you would not have guessed
- an asset's import settings or source turn out to matter for reproducibility
- you build the same scene arrangement or script idiom for the second time
- you learn a pipeline step that must be done the same way every time

One good memory beats five fragments. If a memory with the same `topic_key` already exists, `tl_update` it rather than saving a near-duplicate.

## 2. Pick exactly one `type`

| `type` | Use it for | `topic_key` pattern |
| --- | --- | --- |
| `discovery` | Something non-obvious you found out that no existing memory covers | `<domain>/<subject>` |
| `architecture` | System structure: packages, boundaries, contracts | `architecture/<area>` |
| `bugfix` | A bug, its root cause, and the fix | usually omit it; `bug/<area>` if it recurs |
| `decision` | A technical or product decision with cross-session weight | `decision/<area>/<choice>` |
| `config` | A setting, flag or version pin whose exact value has to be reproduced | `config/<area>` |
| `pattern` | A solution shape you would reuse in a different file | `pattern/<area>` |
| `preference` | Per-developer ergonomics — **`scope` MUST be `personal`** | `preference/<area>` |

These seven are ordered by measured use across 2,336 real memories in 13 projects, read on 2026-09-29, and that is why they are the seven: `discovery` 447, `architecture` 391, `bugfix` 390, `decision` 326, `config` 84, `pattern` 48, `preference` 42. Every agent session on this stack is already told to choose from this list, so a catalogue that contradicts it gets contradicted right back.

If nothing fits, **pick the closest one — do not invent a type.** An invented type is invisible to every future search by category, which is the exact failure this vocabulary exists to prevent. If a type is genuinely missing, open an issue; new types are additive.

A game project can also reach for the optional extensions — `game-design-decision`, `scene-pattern`, `asset-reference`, `perf-gotcha`, `pipeline-step`, `script-pattern`, `convention`. They are not in the table because nothing measured has needed one: together they account for 4 of those 2,336 memories. This file is copied out of its own repository, so it stays self-contained by design; the full catalogue, with the required sections and a worked example per type, is at [`docs/design/memory-domain.md`](https://github.com/AgusLoza2021/Thoughtline/blob/main/docs/design/memory-domain.md).

> Engram's own tools write their own types (`session_summary`, `manual`). Neither is in this catalogue, and this server refuses a save that asks for one — out of scope here, and worth knowing if you are migrating a store that full of them.

## 3. Shape the content

Every memory carries at least these four lines:

```
**What**: one sentence — what was done or learned.
**Why**: what motivated it.
**Where**: the files, paths, scenes or packages it touches.
**Learned**: gotchas, surprises, what you would do differently.
```

Types that ask for more, get more: `bugfix` wants *Symptom* / *Root cause* / *Fix*; `decision` wants *Alternatives considered*; `config` wants *Why this value*; `pattern` wants *When NOT to use it*.

Write for the next session, not for the current one. **A memory nobody can act on is worse than no memory — it looks like knowledge.**

## 4. Tags go in the `tags` field

Pass them as a list of lowercase `key:value` strings:

```
tl_save(type="bugfix", tags=["engine:unity", "asset:texture", "pipeline:fbx", "perf:memory"])
```

`tl_search` filters on that field, and `tl_get_observation` prints it back. Each tag has to match `^[a-z0-9][a-z0-9:_-]{0,40}$`; an uppercase or spaced tag is refused rather than silently normalised.

A tag can also live:

1. **In the `topic_key`**, when its pattern has a slot for it. `perf/android/static-batching` already carries `platform:android`; `scene/godot/inventory-ui` already carries `engine:godot`.
2. **On a `**Tags**:` line as the first line of `content`** — the convention this preset used before the field existed. It is indexed as body text, so those tags stay findable, but they cannot be filtered and an edit to the body can lose them.

Namespaces: `engine:` · `platform:` · `pipeline:` · `asset:` · `phase:` · `tool:` · `perf:`. The canonical list is in [`docs/design/tag-conventions.md`](../docs/design/tag-conventions.md).

## 5. Hard rules

These are not style preferences. Each one costs you something concrete when it drifts.

| Rule | What you lose if it drifts |
| --- | --- |
| `title` is non-empty and ≤ 200 characters | Titles stop working as an index; search results read as a wall of sentences |
| `content` is non-empty and self-contained | The memory looks like knowledge but the next session cannot act on it |
| **Always pass `project`** | The memory becomes invisible to every per-project recall |
| `type` is one of the fourteen in section 2 | An invented type is refused — the save fails instead of filing a memory nobody can find by category again |
| `scope` is `project` or `personal`; `preference` **must** be `personal`, everything else `project` | Memories leak across projects, or hide from the project that needs them |
| `topic_key`, when present, is lowercase with no spaces and no leading slash (`^[a-z0-9][a-z0-9/_.-]{1,128}$`); dots only for version numbers | Nothing breaks loudly — the key quietly stops being greppable, and the `/` shortcut stops finding it |
| Tags match `^[a-z0-9][a-z0-9:_-]{0,40}$`, lowercase | Tags misspell themselves into invisibility, and a filter matches nothing without saying so |
| `content` stays under 64 KiB, counted in bytes | The server refuses the save outright |

---

## Where the rest lives

- The full type catalogue, with required sections and worked examples: [`docs/design/memory-domain.md`](../docs/design/memory-domain.md)
- The tag namespaces: [`docs/design/tag-conventions.md`](../docs/design/tag-conventions.md)
- Why the server is the product and the vocabulary is its gate: [ADR 0008](../docs/decisions/0008-the-engine-is-the-product.md)
