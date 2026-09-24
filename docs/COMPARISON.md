# Where this sits — Thoughtline, Engram, claude-mem

> **This page was rewritten, and the previous version was wrong in a way worth
> naming.** It compared three projects as if they were three of the same thing:
> three MCP servers competing for the same slot. Thoughtline is not a server
> anymore, and never needed to be one — it is the vocabulary that runs on top of
> one. The old page also asserted that claude-mem is AGPL-3.0, which is false and
> never was true; see [Licensing, corrected](#licensing-corrected) at the bottom.
> The previous text is in this file's git history if you need it.

Three names come up when you search for "memory for an AI coding assistant". They are not three of the same thing, and reading them as three of the same thing is how this page went wrong the first time.

## The one-paragraph version

**[Engram](https://github.com/Gentleman-Programming/engram) is the memory server.** Storage, search, MCP tools, sessions, a TUI, sync. It is what you install.

**[claude-mem](https://github.com/thedotmack/claude-mem) is a different bet on that same layer.** Rather than waiting to be told what to save, it captures what your agent does during a session, compresses it with AI, and injects the relevant parts back later. You install it instead of the other one, not alongside it.

**Thoughtline is neither.** It is the vocabulary: the eleven `type` values and the tag conventions that decide what a memory is *called*. It sits on top of an explicit-save server, which today means Engram. It is documents, not a binary — there is nothing to install and no third slot in your config.

## A layer and two servers, not three columns

| Layer | The decision it owns | Who provides it |
| --- | --- | --- |
| Keeping and finding observations | storage, search, MCP tools, sessions | **Engram** |
| What a memory is called — its `type`, its tags | the shared language the store gets written in | **Thoughtline** (this repository) |
| What was worth keeping at all | capture and compression, decided for you | **claude-mem** |

The third row is not a layer above or below the first two. It is a different answer to the same question, and it is precisely why the vocabulary here does not apply to it: if the capture step is not a typing step, there is no moment at which anyone chooses a `type`, and therefore no place to put a vocabulary. If that trade is what you want, take it — nothing in this repository is for you, and that is not a failure of either project.

## Engram vs claude-mem, which *is* still a real comparison

Two servers, one slot in your editor's config. The axis that separates them is where memory comes from in the first place.

| | [Engram](https://github.com/Gentleman-Programming/engram) | [claude-mem](https://github.com/thedotmack/claude-mem) |
| --- | --- | --- |
| Where memory comes from | Someone saves it on purpose — you, or an agent following a protocol you gave it | The tool records what happened, then compresses it down |
| The bet | **Precision.** What is stored is what someone decided was worth keeping | **Coverage.** Nothing is lost, including the things nobody would have chosen |
| Typing | Explicit: the caller names the `type`. Because typing is a choice, a vocabulary is possible at all | Capture is not a typing step, so a shared vocabulary is not something you can hand it by design |
| Who can contribute a save | Anything that speaks MCP | The tool, watching the session |
| Licence | MIT | Apache-2.0 |
| Adoption, measured 2026-09-24 | ~6.8k stars | ~94.6k stars |

That last row is a snapshot, not a verdict — counts move, and they measure attention rather than fit. Check both repositories before you weight it.

Neither row says which is better, because "better" depends on a question only you can answer: **do you want to decide what gets remembered, or do you want that decided for you?** Engram asks you the first question. claude-mem answers the second one for you. Thoughtline only matters once you have already answered "I want to decide" — it makes that decision cheaper by giving you names to decide *with*.

## What this repository adds to a server

1. **A gamedev vocabulary decided in advance.** Without one, every session invents its own words and the store fills with mush — see the [memory domain](design/memory-domain.md) for why a field that accepts any string gives you no shared language. The eleven types (`scene-pattern`, `asset-reference`, `perf-gotcha`, `pipeline-step`, `script-pattern`, …) exist so that "the lantern texture import settings that didn't blow out the bloom" has a category before you need one.
2. **Tags tuned for engines and pipelines.** `engine:unity`, `platform:switch`, `pipeline:fbx-to-godot` — see [tag conventions](design/tag-conventions.md), including where tags actually live now that they are not a field.
3. **Adoption docs per editor and per engine.** The [integration guides](integrations/) exist so that step 3 of the README — teaching the agent the words — is copy-paste rather than a research project.

That is the whole contribution. It is small on purpose: it is the part Engram deliberately leaves open, filled in for one domain.

## When this repository is the wrong choice

- **Your work is not games.** Then the general vocabulary your server already suggests fits you better, and adding a gamedev taxonomy buys you nothing. Use Engram on its own.
- **You want memory you never think about.** That is claude-mem's trade and it is a good one. Explicit saving is work, and if you will not do that work, a precise store you never write to is worse than an imprecise one that fills itself.
- **You want a bigger tool surface, cloud sync between machines, or an HTTP API.** Those are Engram's, and this repository neither adds to them nor competes with them.

## Can I run more than one?

**Thoughtline and anything else: yes, trivially** — it occupies no slot, because it is not a server. It is a set of words you put in your agent's instructions.

**Engram and claude-mem together: technically yes, practically no.** They do not collide; they are separate servers with separate tool names, and nothing stops you registering both. What you get is the same fact stored twice, once typed and once compressed, and an agent that has to decide which one to ask. Pick one. The interesting thing about this pair is that they are not really rivals: they disagree about whether you should have to think about memory, and that argument is not resolvable by running both.

## Licensing, corrected

This section used to open by stating that **claude-mem is AGPL-3.0**, and that the passive-capture design had to be written carefully around copyleft contamination. It went on to describe a contributor checklist and an automated guard written to enforce that care.

**That was wrong, and it was never right.** Verified against the repository itself on 2026-09-24:

- claude-mem's **first** `LICENSE` (v3.3.8, 2025-09-06) was a custom "Claude Mem License": permissive use and redistribution of the binaries, MIT for the files under `/hooks`, with no-reverse-engineering and no-modification terms for the binaries themselves. Restrictive in its own way — and not copyleft.
- claude-mem's **current** `LICENSE` is the **Apache License 2.0**. The file contains no mention of the AGPL at all.

So the premise was false in both directions. There was never AGPL code near this project, because there was never any AGPL to be near. And moving Apache-2.0 code into an MIT project is permitted with attribution, which is what every project does routinely.

What survives is the small and uninteresting half: no code, prompts or schemas from claude-mem were copied into Thoughtline. The hook list and the queue-then-promote shape are **ideas**, and ideas were never the licensed part. `scripts/check-no-claude-mem.sh` and its PowerShell twin still run in CI enforcing the stricter rule this project wrote for itself while it believed otherwise — a self-imposed choice, not an obligation anyone owed. See `CONTRIBUTING.md` for that checklist in its current form.
