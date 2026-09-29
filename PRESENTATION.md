# Thoughtline — Presentación

> Memoria persistente para asistentes de IA. Local. Para siempre.
> *Persistent memory for AI assistants. Local. Forever.*

---

## 🎯 El problema / The problem

**ES** — Tu IA olvida TODO entre sesiones. Cada vez que volvés a trabajar, tenés que re-explicarle:
- Por qué la jerarquía de tu escena está organizada así
- Qué configuración de import usaste para esa textura
- Cuál era el bug de batching en Android que ya resolviste
- Las convenciones del proyecto, las decisiones del equipo, las trampas que ya pisaste

Resultado: perdés tiempo, la IA repite errores, el conocimiento del proyecto se evapora.

**EN** — Your AI forgets EVERYTHING between sessions. Every time you come back, you re-explain:
- Why your scene hierarchy is organized that way
- What import settings you used for that texture
- The Android batching bug you already solved
- Project conventions, team decisions, the traps you already stepped on

Result: you lose time, AI repeats mistakes, project knowledge evaporates.

---

## 💡 La solución / The solution

**ES** — Thoughtline es **un archivo SQLite en tu máquina** que la IA escribe mientras trabajás y lee cuando volvés. Sin nube, sin cuenta, sin suscripción. Tu data, tuya.

Hablás con la IA → ella guarda lo importante → en la próxima sesión, lo recuerda.

**EN** — Thoughtline is **one SQLite file on your machine** that the AI writes to while you work and reads from when you return. No cloud, no account, no subscription. Your data, yours.

You talk to the AI → it saves what matters → next session, it remembers.

---

## 🧠 ¿Cómo funciona? / How does it work?

```
┌─────────────────┐    MCP    ┌──────────────┐    ┌─────────────┐
│  Tu IA          │ ────────▶ │ Thoughtline  │ ─▶ │ thoughtline │
│  (Claude, etc.) │           │ (servidor)   │    │ .db (local) │
└─────────────────┘           └──────────────┘    └─────────────┘
```

**ES** — La IA llama a herramientas (`tl_save`, `tl_search`...) usando un protocolo estándar llamado **MCP** (Model Context Protocol). Thoughtline guarda y busca en una base de datos local con búsqueda full-text (FTS5).

**EN** — The AI calls tools (`tl_save`, `tl_search`...) using a standard protocol called **MCP** (Model Context Protocol). Thoughtline saves and searches a local database with full-text search (FTS5).

---

## 🎮 ¿Para quién? / Who is it for?

**ES** — Pensado para **equipos de desarrollo de videojuegos**. Otras herramientas de memoria hablan el idioma del backend (`bugfix`, `decision`, `pattern`). Thoughtline habla el idioma del gamedev:

**EN** — Built for **game-dev teams**. Other memory tools speak backend language. Thoughtline speaks gamedev:

| Tipo / Type            | Captura / Captures                                          |
| ---------------------- | ----------------------------------------------------------- |
| `scene-pattern`        | Jerarquías de entidades, setups de componentes              |
| `asset-reference`      | Rutas, versiones, settings de import de modelos/texturas    |
| `perf-gotcha`          | Trampas de performance (draw calls, GC, batching)           |
| `pipeline-step`        | Pasos del pipeline (Blender → engine, etc.)                 |
| `script-pattern`       | Idioms de scripting por engine                              |
| `game-design-decision` | Decisiones de diseño y el porqué                            |

Funciona con **cualquier engine** (Unity, Unreal, Godot, PlayCanvas) y **cualquier IDE que hable MCP** (Claude Code, Cursor, Zed, Rider, VS Code).

Works with **any engine** and **any MCP-capable IDE**.

---

## 🛠️ Las herramientas / The tools

12 herramientas MCP que la IA usa sola — vos no tenés que pensar en ellas:

12 MCP tools the AI uses on its own — you don't have to think about them:

| Tool                 | Qué hace / What it does                                    |
|----------------------|------------------------------------------------------------|
| `tl_save`            | Guarda una memoria / Saves a memory                        |
| `tl_search`          | Busca con FTS5 + BM25 / Full-text search                   |
| `tl_get_observation` | Trae el contenido completo / Fetches full content          |
| `tl_context`         | Memorias recientes del proyecto / Recent project memories  |
| `tl_update`          | Edita una memoria / Patches a memory                       |
| `tl_delete`          | Borra (soft) una memoria / Soft-deletes                    |
| `tl_session_start`   | Abre una sesión / Opens a session                          |
| `tl_session_summary` | Cierra con un digest estructurado / Closes with digest     |
| `tl_stats`           | Estadísticas del proyecto / Project stats                  |
| `tl_pending_*`       | Captura pasiva vía hooks / Passive capture via hooks       |

---

## 🖥️ Dashboard interactivo / Interactive dashboard

```bash
thoughtline ui
```

**ES** — Dashboard TUI (terminal) hecho con Bubbletea. 3 temas, búsqueda full-text, cubo 3D animado en el header. Read-only — nunca modifica la base.

**EN** — Bubbletea TUI dashboard. 3 themes, full-text search, animated 3D cube in the header. Read-only — never mutates the DB.

---

## 🆚 Cómo se compara / How it compares

| Capacidad / Capability                  | Thoughtline | Cursor   | ChatGPT  | Notion   |
|-----------------------------------------|-------------|----------|----------|----------|
| Local-first, tu data en tu máquina      | ✅          | ❌ nube  | ❌ nube  | ✅       |
| Funciona en múltiples IDEs              | ✅ MCP      | ❌       | ❌       | ❌       |
| La IA guarda sola (sin copy-paste)      | ✅          | ✅       | ✅       | ❌       |
| La IA recuerda en sesiones nuevas       | ✅          | ✅       | ✅       | ❌       |
| Sobrevive compactaciones de contexto    | ✅          | ⚠️       | ⚠️       | ✅       |
| Por-proyecto (sin contaminación cruzada)| ✅          | ⚠️       | ❌       | ✅       |
| Tipos de memoria nativos para gamedev   | ✅          | ❌       | ❌       | ❌       |
| Costo / Cost                            | Gratis MIT  | Cursor   | ChatGPT  | Gratis-Pago |

---

## 🏛️ Arquitectura / Architecture

```
AI Client ──MCP/stdio JSON-RPC──▶ Thoughtline Server
                                        │
                                        ▼
                                  Tool layer
                                  (tl_save, tl_search, ...)
                                        │
                                        ▼
                                  Memory domain
                                  (types, topic keys, scopes)
                                        │
                                        ▼
                                  SQLite + FTS5 / BM25
                                        │
                                        ▼
                                  thoughtline.db
```

**Stack**: Go 1.25+ · SQLite (modernc.org/sqlite, sin CGO) · MCP-go · Bubbletea TUI.

---

## 🚀 Instalación / Install

**Una línea en Windows / One line on Windows:**
```powershell
irm https://raw.githubusercontent.com/AgusLoza2021/Thoughtline/main/scripts/install.ps1 | iex
```

**Manual (cualquier OS / any OS):**
```bash
go install github.com/AgusLoza2021/Thoughtline/cmd/thoughtline@latest
```

**Plugin para Claude Code:**
```bash
claude plugin install github:AgusLoza2021/Thoughtline/plugin/claude-code
```

---

## 📊 Estado actual / Current status

| Milestone        | Goal                                                | Status     |
| ---------------- | --------------------------------------------------- | ---------- |
| M0 Bootstrap     | Repo, docs, ADRs, taxonomía / taxonomy              | 🟢 done    |
| M1 Save          | `tl_save` + SQLite + FTS5 + topic-key upsert        | 🟢 done    |
| M2 Search        | `tl_search` FTS5 + BM25 + `tl_get_observation`      | 🟢 done    |
| M3 Context       | `tl_context`, `tl_update`, `tl_delete`              | 🟢 done    |
| M4 Sessions      | `tl_session_start` + `tl_session_summary`           | 🟢 done    |
| M5 Dashboard     | `thoughtline ui` + `tl_stats`                       | 🟢 done    |
| M6 Smarts        | Embeddings semánticos / Semantic embeddings         | 🔵 deferred |
| **Brain (extra)**| Brain entity, typed graph, events bus               | 🟢 done    |
| **Passive capture** | Captura por hooks de Claude Code / via hooks     | 🟢 done    |

---

## 🌱 Lineage / Origen

**ES** — Thoughtline está parado en los hombros de **[Engram](https://github.com/Gentleman-Programming/engram)** de Alan Buscaglia. Reusamos su forma MCP, su layout de storage, FTS5, topic-key upserts. Lo que agregamos: **taxonomía gamedev-first** y vocabulario para Unity/Unreal/Godot/PlayCanvas. MIT-compatible — las mejoras pueden volver upstream.

**EN** — Thoughtline stands on the shoulders of **Engram** by Alan Buscaglia. We reuse its MCP shape, storage layout, FTS5, topic-key upserts. What we add: **gamedev-first taxonomy** and engine vocabulary. MIT — improvements flow back upstream.

---

## 🎤 El mensaje en una frase / The one-liner

**ES** — *"Tu IA finalmente recuerda lo que tu equipo ya aprendió."*

**EN** — *"Your AI finally remembers what your team already learned."*

---

## 📚 Links

- Repo: https://github.com/AgusLoza2021/Thoughtline
- Arquitectura: [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)
- Taxonomía: [`docs/design/memory-domain.md`](docs/design/memory-domain.md)
- Decisiones (ADRs): [`docs/decisions/`](docs/decisions/)
- Comparación honesta: [`docs/COMPARISON.md`](docs/COMPARISON.md)

License: MIT
