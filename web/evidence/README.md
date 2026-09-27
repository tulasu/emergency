# Curriculum Flow B — MCP evidence pack

Pairs of design (`.pen` Export) and live screenshots after pixel loop.

| Frame | Design | Live |
| --- | --- | --- |
| B1 Modules list `hP3BU` | `design/hP3BU.png` | `live/B1-modules.png` |
| B2 Module detail `W86teS` | `design/W86teS.png` | `live/B2-module.png` (= `B2-module-detail.png`) |
| B5 New module `Y3rajI` | `design/Y3rajI.png` | `live/B5-new-module.png` (= `B5-modules-new.png`) |
| B4 Lesson variants `Duwvc` | `design/Duwvc.png` | `live/B4-lesson.png` |
| B4c Variant editor `O6VLW` | `design/O6VLW.png` | `live/B4c-variant-editor.png` |
| B6 Open lesson `rV1R8` | `design/rV1R8.png` | `live/B6-open-modal.png` (= `B6-open-lesson-modal.png`) |
| Cards pool `ghGs1` | `design/ghGs1.png` | `live/pool-cards.png` |
| Lessons pool `umr5M` | `design/umr5M.png` | `live/pool-lessons.png` |
| Student home `F1NUvg` | `design/F1NUvg.png` | `live/student-home.png` |
| Student module `azbHm` | `design/azbHm.png` | `live/azbHm-student-module.png` |

Shell contract: bg `#F3F3F3`, pad/gap 24, sidebar 220, Manrope, accent `#3657DB`.

Dev note: colliding page/API paths (`/modules`, `/tickets`, …) are proxied only when `Accept` is not `text/html` via `server/middleware/api-proxy.ts`.
