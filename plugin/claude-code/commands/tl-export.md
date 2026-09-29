---
description: Export thoughtline memories to a markdown file the user can review or commit.
argument-hint: [output-path]
---

Build a markdown export of the active project's thoughtline memories.

Steps:

1. Call `tl_context` with a high `RecentLimit` (e.g. 200) to grab everything recent.
2. For each memory, call `tl_get_observation` to retrieve full content.
3. Group by `type` (decisions, conventions, architecture, bugfixes, …).
4. Write to `$ARGUMENTS` if provided, otherwise default to `THOUGHTLINE-EXPORT.md` in the repo root.

Each entry's section should be:

```
## <title>

- **type**: <type>
- **topic_key**: <topic_key or "—">
- **updated**: <ISO date>
- **scope**: <project|personal>

<full content>
```

Confirm with the user before overwriting an existing file.
