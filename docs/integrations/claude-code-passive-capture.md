# Claude Code — Passive Capture Integration

<!-- retired-v0.1.0 -->
> **Retired — the v0.1.0 MCP server this page was written for is unmaintained.**
> Its opt-in steps and hook wiring are kept at the bottom as a record of how the
> project worked, not as a path to follow. For what this project is now — a gamedev
> memory vocabulary that runs on
> [Engram](https://github.com/Gentleman-Programming/engram) — read the
> [README](../../README.md), the [memory domain](../design/memory-domain.md) and the
> [tag conventions](../design/tag-conventions.md).

## What survives from this page

The feature described below — a queue of raw hook events that you review and promote into typed memories — was implemented by the retired server, and its `tl_pending_*` / `tl_promote` tools went with it. The *idea* did not go anywhere, and Engram ships it: passive capture is one of Engram's tools. Wire it to Claude Code's hooks following Engram's own documentation; none of this repository is needed for that, and there is no env var or hook file here to copy.

What this page still has to offer is the part that is not engine-specific — and it is the part worth reading before you turn capture on.

> **The privacy trade-off does not depend on who wrote the server.** Passive capture stores raw hook payloads: `UserPromptSubmit` (your prompts, verbatim), `PreToolUse` and `PostToolUse` (tool inputs and outputs, which may include file contents, secrets, or paths you would rather not keep). Whichever server you run, that is a copy of your raw session sitting next to your curated memories. The old default was OFF for this reason. Keep it that way until you have decided where that data lands and who can read it.

The two-stage shape is the other thing worth keeping, because it is the reason this design works at all: **capture raw, then promote deliberately.** A git hook cannot tell a durable decision from a passing remark, so it should not try. It records, untyped and unjudged, into a queue. The judgement — which of the eleven `type` values, which `topic_key`, whether the body is worth re-reading in six months — happens later, when someone actually looks at the queue. If you wire passive capture, keep that split. A server that auto-promotes everything into typed memories on the way in will fill your memory with noise you then have to search through.

---

> **Legacy — the v0.1.0 setup.** Everything below documents `thoughtline`, the
> retired server, and its `tl_*` tools. It is a record of how the project worked,
> not instructions to follow.

Thoughtline can passively capture raw Claude Code hook events into a local queue
(`pending_events`). You review the queue with `tl_pending_list` and promote
selected events into typed memories with `tl_promote`. **Default: OFF.**

## Privacy notice

When enabled, this feature stores raw hook payloads — including `UserPromptSubmit`
(your prompts verbatim) and `PreToolUse` / `PostToolUse` (tool inputs and outputs,
which may include file contents, secrets, or sensitive paths) — **unencrypted in
the same SQLite file as your memories**.

Only enable this if you understand the privacy implications and control access
to your local machine. See [design §11](../../openspec/changes/archive/passive-capture-hooks/design.md)
for the full privacy posture and planned future mitigations.

## Opt in

Set the environment variable before launching Claude Code:

```bash
# Linux / macOS
export THOUGHTLINE_PASSIVE_CAPTURE=1

# Windows PowerShell
$env:THOUGHTLINE_PASSIVE_CAPTURE = "1"

# Windows cmd
set THOUGHTLINE_PASSIVE_CAPTURE=1
```

To make it persistent, add the export to your shell profile or set it as a
system/user environment variable.

The env var takes precedence over any config file setting (v1: config file not
yet supported — env var only).

## Hook registration

Copy or merge `plugin/claude-code/hooks/hooks.json` into your Claude Code hooks
configuration. The file registers 6 events:

| Claude Code event | Captured as |
|-------------------|-------------|
| `SessionStart` | `SessionStart` |
| `UserPromptSubmit` | `UserPromptSubmit` |
| `PreToolUse` | `PreToolUse` |
| `PostToolUse` | `PostToolUse` |
| `Stop` | `Stop` |
| `SessionEnd` | `SessionEnd` |

Each hook invokes `thoughtline hook <event-name>` with the event JSON on stdin.
The hooks use `"on_failure": "continue"` so a hook failure never blocks your
Claude Code session.

## Triage workflow

After a session, call `tl_pending_list` to see captured events:

```
tl_pending_list project="my-project" status="pending" limit=20
```

Inspect a specific event with `tl_pending_get`:

```
tl_pending_get id=42
```

Promote selected events into memories with `tl_promote`:

```
tl_promote items=[
  {
    "pending_event_id": 42,
    "type": "bugfix",
    "title": "Fixed nil pointer in startup",
    "content": "**What**: ...\n**Why**: ...\n**Where**: main.go"
  }
]
```

Promoted memories are immediately searchable via `tl_search`.

## Retention janitor

Pending events older than 7 days are automatically archived. Archived events
older than 30 days are hard-deleted. Run the janitor manually or via cron:

**Unix cron** (daily at 03:00):

```cron
0 3 * * * /usr/local/bin/thoughtline worker
```

**Windows Task Scheduler** (run daily):

```powershell
$action  = New-ScheduledTaskAction -Execute "thoughtline" -Argument "worker"
$trigger = New-ScheduledTaskTrigger -Daily -At "3:00AM"
Register-ScheduledTask -TaskName "ThoughtlineWorker" -Action $action -Trigger $trigger
```

Custom retention windows:

```bash
thoughtline worker --retention 14d --hard-delete 60d
```

## Opt out

Unset the environment variable. No data is deleted — you can opt back in at any
time. To purge the queue manually:

```sql
-- Connect to your thoughtline.db and run:
DELETE FROM pending_events WHERE status = 'pending';
```
