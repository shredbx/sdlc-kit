# Reference repos index

Last updated: 2026-09-24
tags: index, reference, research

Purpose: one lookup table for every prior-work repo added as a reference so far, so a future session
doesn't need to re-scan any of them from scratch — check here first, read the deep-dive doc if one
exists, and only re-explore the actual repo for something genuinely new. **Update this file in
place** whenever a new reference repo is added or an existing one gets a deeper look — don't create
a parallel index.

## Primary references (deep-dived already)

| Repo | Path | What it is | Deep-dive doc(s) |
|---|---|---|---|
| process-os | `/Users/solo/Projects/workspaces/process-os` | The tool sdlc-kit runs on; v3's actual engine, read directly from source | `process-os-patterns.md` |
| shredbx (v1, live) | `/Users/solo/Projects/workspaces/shredbx` | ~1-year-old real monorepo; source of most porting candidates | `shredbx-codebase-inventory.md`, `shredbx-sdlc-system.md`, `shredbx-bestierealestate-and-capabilities.md`, `shredbx-bestierealestate-ai-assistant.md` (verified deep-dive: BR's AI assistant tool-calling/property-search/streaming, confirmed real end-to-end) |
| sbx.framework (v2) | `/Users/solo/Projects/workspaces/sbx.framework` | Paused predecessor; birthplace of process-os's own engine design | `sbx-framework-inventory.md` |
| sbx-workspace | `/Users/solo/Projects/workspaces/sbx-workspace` | Early filestructure precedent (`platform/python/{applications,frameworks,packages}`, `projects/{products,experiments,prototypes,tools}`, `consumers/`) | CLAUDE.md/README read directly, folder tree only — no standalone doc yet |
| Python quality comparison | — | process-os's own Python vs. shredbx/sbx.framework Python, craftsmanship verdict | `python-implementation-quality-comparison.md` — **done**. Verdict: process-os is the most internally consistent/disciplined of the five codebases read (error-handling split, typing, packaging uniformity), but has zero lint/type-check/CI (a gap shared by every comparison codebase except sbx.framework's CI, which itself has no Python lint/type job either) and a self-admitted test gap in `types/` (string/integer/float/sequence untested). Recommendation: wire up ruff+mypy+CI at the point of porting, not after. |

## `shredbx-workspace-reference` — a graveyard of 9 older/parallel projects

Path: `/Users/solo/Projects/workspaces/___/shredbx-workspace-reference` — each subfolder is its own
independent repo (several carry their own `.git`), representing earlier or parallel attempts
predating the live shredbx/sbx.framework lineage. Surveyed 2026-09-24 at overview depth
(READMEs + manifests + last-commit dates); Python depth only on `whisper-python`, per instruction —
the rest are noted, not opened.

| Subproject | Stack | What it is | Last commit | Notes |
|---|---|---|---|---|
| `whisper-python` | FastAPI + SvelteKit (Python backend) | Speech-to-text service: Whisper model, WebSocket streaming, a parallel/overlapping recording strategy | untracked (no `.git`) | The one real Python codebase here. Clean `routers/services/models` split (`backend/app/`), plus its own independent YAML user-story requirements system (`requirements/system-definition.yml` + `requirements/user-stories/US-*.yml`, even a small `req_cli.py`) — a third independent example (alongside shredbx's `.sbx/` and sbx.framework's rows) of schema/YAML-driven requirements, worth comparing. Pairs with sbx.framework's separate Swift voice-to-text menu-bar app — two unrelated attempts at voice/transcription tooling, worth cross-referencing if that ever becomes a real feature. |
| `bestays-svelte` | SvelteKit 5 + FastAPI | An earlier, standalone Bestays real-estate attempt (Svelte+**FastAPI**, vs. the live shredbx one's Svelte+**Go**) | 2025-11-12 | CI/CD + codecov already wired. A different backend-language choice for the same product idea — comparison value if the Go rebuild's tradeoffs are ever revisited. |
| `ex-nextjs-zustand-dynamic-forms-demo` | Next.js 15 + Zustand + Immer | Dynamic form builder demo, "Domain-Property-Record-Value" architecture | 2025-12-20 | Worth a real look later — schema-driven dynamic forms is directly relevant to our own entity-modeling ambitions and to the AI-assistant chat UI's form needs. |
| `seaside-workspace` | Next.js 15 + React 19 + Supabase | CMS-style content-model starter | 2025-10-06 | A second Next.js-starter reference alongside `bestays-web` — worth comparing tech choices (Supabase vs. our own Go-API pattern) when finalizing the Next.js bootstrap. |
| `remote-claude-code` | unknown (`claude-code-ui/`) | Barely started — one `init` commit | 2025-11-27 | Minimal; skip unless it becomes specifically relevant. |
| `shredbx` (nested copy) | `services/`-based monorepo | An earlier-branded "Experimental Development Lab" version of what's now the live shredbx — different top-level layout (`services/`, not `clients/`+`projects/`) | 2025-11-24 | Evolutionary artifact, not a separate product — shows an earlier structural convention before the current shredbx shape. |
| `shredbx-sandbox-001-background-video-stream` | SvelteKit-based | A sandbox spike | 2025-11-24 | Narrow, likely low reuse value. |
| `sshredb` ("ShredBreak") | Node/SvelteKit-ish | Multi-tenant platform for extreme-sports businesses (kiteschools, wakeparks, skateparks) | untracked | A distinct, unrelated product idea — noted for completeness, not obviously relevant to sdlc-kit's own goals. |
| `ios-swift-apps/` (3 apps) | Swift, Xcode, fastlane, Firebase | Three real **client** iOS apps: `dtac-ev-platform-ios` (EV-charging platform), `major-cineplex-ios` (cinema chain — has fastlane + BuildTools + Configs), `se-digital-erx-ios` (digital prescription/pharmacy) | not checked individually yet | **Significantly more Swift prior art than previously known.** Beats myFamilyTree/SevenTree as the richest Swift asset by volume — three real production client apps vs. one. Worth a proper survey before finalizing any Swift app-starter design. |

## Not yet added as working directories (seen while listing `___`, not explored)

`/Users/solo/Projects/workspaces/___/` also contains `_old-sbx`, `bestie`, `business_solution`,
`claude-code-sdlc` (name suggests direct relevance to this very workspace's own SDLC design — worth
a look if more prior art is needed), `copilot-worktrees`, `kite-cable-nextjs` (another possible
Next.js reference), `whisper-ui-prototype`, `worktrees`. Left untouched — add as a working directory
first if any of these become relevant.

## How to use this file

Skim the tables above before re-scanning a repo from scratch. If a deep-dive doc exists, read that
instead of re-exploring the repo. If you explore something new in any of these repos later, add a
row or update this file rather than letting the finding live only in chat history.
