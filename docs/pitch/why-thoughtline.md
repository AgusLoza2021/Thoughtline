# Why Thoughtline — pitch & talking points

> Cheat-sheet para explicar Thoughtline en 10 segundos, en una reu, o cuando te tiran una objeción.
> No es documentación técnica. Para eso está el [README](../../README.md) y [ARCHITECTURE.md](../ARCHITECTURE.md).

---

## 1. Para vos (ELI5) — la respuesta de 10 segundos

**Thoughtline es un cuaderno que la IA lleva sobre nuestro proyecto, y que sobrevive entre sesiones.**

- **Sin Thoughtline**: cada vez que abrimos Claude / Cursor / Zed, la IA arranca como un dev nuevo en su primer día. Le explicamos todo. Mañana arranca igual, de cero.
- **Con Thoughtline**: la IA dejó notas la sesión anterior. Hoy las relee antes de contestar. Es como un dev que ya conoce el proyecto.

Eso es todo. Lo demás (MCP, SQLite, FTS5, BM25) son **detalles de implementación** — solo importan si alguien te pregunta cómo funciona por dentro.

### Si te piden un ejemplo concreto

> "El mes pasado decidimos partir el inn en `world/static` y `world/interactive` para que batchee bien en Android. Hoy le pedís a Claude que agregue una silla nueva — sin Thoughtline, te la mete en `static` y rompe el batching. Con Thoughtline, recuerda la convención y la pone donde va."

Multiplicá esto por cada decisión de diseño, cada gotcha de performance, cada paso de pipeline que hoy vive solo en la cabeza de alguien.

---

## 2. Para los bosses — one-pager

> Cuatro argumentos. Cero jerga técnica. En idioma de negocio.

### El problema que resuelve

Cada vez que un dev usa una IA en el proyecto, la IA arranca **sin contexto**. No sabe nuestras convenciones, no recuerda decisiones pasadas, no conoce los gotchas que ya pagamos descubrir. Resultado: el dev pierde tiempo re-explicando o aceptando sugerencias que rompen lo que ya teníamos.

### Los 4 argumentos

**1. Tiempo recuperado**
- Onboarding de un dev nuevo a la IA del proyecto: pasa de días a horas.
- Cada sesión de IA: 10–15 minutos menos de re-contexto.
- A escala de equipo, son horas por semana, todas las semanas.

**2. Retención de conocimiento del proyecto**
- Hoy el "por qué hicimos X así" vive en la cabeza del senior. Si se va, se va con él.
- Thoughtline lo captura como artefacto del proyecto, no como documento muerto en Notion.
- Es disciplina de equipo, no burocracia.

**3. Calidad de output de IA**
- IA sin contexto = sugerencias genéricas que rompen convenciones nuestras.
- IA con memoria del proyecto = sugerencias que respetan la arquitectura existente.
- Traducción directa: **menos PRs con feedback "esto no es como lo hacemos acá"**.

**4. Costo y riesgo: cero**
- **Sin costo recurrente**: no es un SaaS más en el stack. Un binario, una base SQLite local.
- **Sin data leak**: la base vive en el laptop de cada dev. La IP del juego no sale del equipo.
- **Sin lock-in**: licencia MIT, código nuestro. Si algo cambia upstream, no nos afecta.

### El cierre

> "Esto no es un producto nuevo. Es **infraestructura de equipo** — igual que el linter, igual que el CI. Una herramienta que hace que cada dev sea más rápido y más consistente con las convenciones del proyecto. La inversión de construirlo ya está hecha. Lo que falta es adoptarlo y empezar a capitalizar."

---

## 3. Objeciones frecuentes

> Lo que te van a tirar en la reu, y cómo respondés sin trabarte.

| Objeción                                            | Respuesta corta                                                                                                                                                       |
| --------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| "¿Por qué no usamos Engram directamente?"           | Engram es excelente, pero es genérico (backend / product). Nuestro vocabulario es gamedev: scenes, assets, perf gotchas, pipelines. La taxonomía propia hace que la búsqueda sea mucho más precisa. Las mejoras pueden volver upstream. |
| "¿Por qué no Notion / Confluence / wiki?"           | Esas son herramientas para **humanos**. Thoughtline es para que la **IA** lo consuma automáticamente, sin que un dev tenga que ir a buscar y pegar contexto. Son cosas distintas, no competidoras. |
| "¿Por qué construirlo nosotros y no comprar algo?"  | Ya está construido. La M2 ya funciona. El ROI cuenta de acá en adelante, no para atrás.                                                                              |
| "¿Cuánto cuesta mantenerlo?"                        | Próximo a cero. Es un binario Go + SQLite. Sin servidores, sin servicios, sin subscriptions.                                                                          |
| "¿Y si el dev no lo usa?"                           | La IA lo consulta sola, no requiere que el dev haga nada. Es transparente. Lo único que pedimos es que la IA **guarde** decisiones importantes — y eso lo hace ella misma cuando está bien instruida. |
| "¿Esto reemplaza la documentación?"                 | No. La documentación sigue siendo para humanos. Thoughtline es la capa que la IA lee. Pueden convivir; de hecho, se complementan.                                     |
| "¿Por qué no embeddings / búsqueda semántica?"      | Para v1 alcanza con full-text search + ranking BM25 — es rapidísimo y suficiente. Embeddings está reservado en el schema para cuando lo necesitemos (M5).             |
| "¿Y si querés cambiar de proveedor de IA?"          | El protocolo MCP es estándar abierto. Funciona con Claude, Cursor, Zed, y cualquier cliente que lo soporte. No estamos atados a nadie.                                |
| "¿Qué pasa si la base se corrompe / se pierde?"     | Cada dev tiene la suya, y se puede regenerar reusando el repo + sesiones nuevas. No es state crítico, es state acumulativo.                                          |
| "¿Esto es seguro? ¿Qué datos se mandan afuera?"     | Cero. La base es local. La IA solo lee/escribe en el laptop del dev. No hay cloud, no hay terceros, no hay telemetría.                                                |

---

## 4. Cómo usar este documento

- **Antes de la reu**: leelo entero una vez. Asegurate de que la analogía del cuaderno te salga sin pensarla.
- **En la reu**: tenelo abierto en otra pestaña. Si te tiran una objeción, mirás la tabla y respondés.
- **Después de la reu**: si te tiran una objeción nueva que no está acá, agregala a la tabla. Esto crece con el uso.

---

*Última actualización: 2026-04-30*
