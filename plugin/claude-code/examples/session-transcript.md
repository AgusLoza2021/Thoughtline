# Example — a session with thoughtline active

Annotated transcript showing the agent calling `tl_*` tools at the right moments. Whitespace condensed; the actual MCP traffic is JSON-RPC.

---

**[SessionStart hook fires]**
> Plugin runs `thoughtline protocol --event session-start` → Claude receives the active-protocol markdown as additionalContext. Claude now knows it has `tl_save`, `tl_search`, etc., and when to use them.

---

**user**: arrancamos con la migración del storage layer

**assistant** *(searches memory before proposing anything)*:
- `tl_search { "query": "architecture/storage-layer" }` → 2 hits
- `tl_get_observation { "id": 42 }` → reads full content of "Storage layer split into Open/Save/Search"
- *Now* proposes an approach grounded in what already exists.

---

**user**: dale, encontré que `tl_update` se rompía con el FTS5 contentless. Era eso. Lo reemplacé con una tabla managed con triggers.

**assistant**:
- This is a bugfix with a non-obvious root cause → **save now**.
- `tl_save { "type": "bugfix", "topic_key": "bug/storage/fts5-contentless", "title": "...", "content": "**What**: ... **Why**: ... **Where**: ... **Learned**: ..." }`
- Confirms with a one-liner: "Saved as bug/storage/fts5-contentless."

---

**user**: y ya que estamos, todos los topic_keys los vamos a estandarizar lowercased con `/`

**assistant**:
- New convention → save with `type: convention`.
- `tl_save { "type": "convention", "topic_key": "convention/topic-keys", "scope": "project", ... }`

---

**user**: listo por hoy

**assistant** *(session close protocol)*:
- `tl_session_summary { "id": "<session-id>", "summary": "## Goal ... ## Discoveries ... ## Accomplished ... ## Next Steps ... ## Relevant Files ..." }`
- Confirms: "Session summarized. See you next time."

---

**[Next session, hours later]**

**[SessionStart hook fires again — fresh context, but persisted memory survives]**

**user**: ¿cómo habíamos resuelto lo del FTS5?

**assistant**:
- `tl_search { "query": "FTS5" }` → finds the bugfix from last session
- `tl_get_observation` → reads it in full
- Answers based on actual prior work, not hallucination.

---

The whole point: **the agent remembers what you decided, you don't re-explain.**
