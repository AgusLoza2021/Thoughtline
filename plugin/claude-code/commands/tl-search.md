---
description: Search thoughtline memories by keyword or topic_key GLOB.
argument-hint: <query>
---

Take the user's argument as a search query and call `tl_search` with it.

Behaviour:
- If the query contains a `/`, it is treated as a `topic_key` GLOB (e.g. `decision/sqlite-*`)
- Otherwise it is FTS5 + BM25 keyword search
- Show top 5 results with title, type, snippet, and relative time
- For any result that looks promising, offer to call `tl_get_observation` for the full content

If there are no results, say so plainly. Do not paraphrase or invent matches.

User query: $ARGUMENTS
