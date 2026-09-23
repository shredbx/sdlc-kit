# shredbx: bestierealestate AI assistant — does it actually work, and is it Next.js-reusable prior art?

Last updated: 2026-09-24
tags: shredbx, bestierealestate, ai-assistant, research

> Scope note: a deep-dive follow-up to `docs/research/shredbx-bestierealestate-and-capabilities.md`
> (which flagged `internal/assistantchat`/`internal/aiassistant`/`internal/assistantcapabilities` as
> "a full, governance-YAML-driven tool-discovery + chat config surface with zero real-estate coupling"
> but did not read the actual chat engine). The user's claim under test: BR's AI assistant "works and
> provides search and property output already in admin panel for testing," and is "production deployed
> and run for a few months... the most advanced system we have built." Verified by reading code, not
> by re-reading the earlier claim. Read-only — nothing built, nothing modeled, nothing touched in
> shredbx.

**Headline verdict: the claim holds up.** This is not scaffolding or a settings-only surface — it is
a genuinely wired, deployed, tool-calling AI assistant with real property search and real streaming
output, running in production since mid/late July 2026 (roughly two months old as of this doc, not
the "few months" the claim states verbatim — see §7 for the exact dates and why that framing is
slightly generous but not fabricated).

---

## 0. The one surprise: the actual engine is not in `apps/api-chi` at all

The starting points named in this task (`internal/assistantchat`, `internal/aiassistant`,
`internal/assistantcapabilities`) are all **Go-side wiring around a separate service**, not the
engine itself. Reading `internal/assistantchat/client.go:1-13` makes this explicit: api-chi is "api-chi's
client to the private chat ENGINE (a separate FastAPI service addressed by ASSISTANT_CHAT_URL)."

The real engine is Python, lives at
`projects/assistant-kit/packages/chat/python/src/sbx_assistant/chat/` (a **shared, non-BR-specific**
package — `sbx-assistant-chat`), and is deployed as its own container
(`clients/bestie/projects/bestierealestate/apps/assistant/{Dockerfile,entrypoint.sh,Makefile,secrets.tpl}`)
running `uv run --no-sync uvicorn sbx_assistant.chat.edges:app --port 5051`
(`apps/assistant/Dockerfile:44`). This matters for the reuse question in §6: the thing worth studying
as prior art is not really "BR's assistant," it's assistant-kit's generic chat engine, with BR as its
one production consumer.

---

## 1. Does it actually provide property SEARCH with real property output? — YES, confirmed by reading the call chain end to end

The full chain, read in order:

1. **Descriptor registry** — `apps/api-chi/internal/assistantcapabilities/descriptor.go:104-137`
   defines `properties.search` as a `Tool`: `Endpoint: "GET /api/properties"`, a JSON-Schema
   `InputSchema` (q, for_sale, for_lease, property_type, province, district, sub_district,
   min_price, max_price, bedrooms, furnished, tags, sort, limit), `Output: "Offering[]"`,
   `RBAC: "none"`. Three more tools exist alongside it: `properties.describe_filters` (facet
   discovery), `properties.collections`, `properties.tags`, plus `faq.search`
   (`descriptor.go:189-207`, module `faq`). A `Skill` (`property.find-suggest`,
   `descriptor.go:210-223`) chains `describe_filters → search` with an explicit numbered procedure
   ("parse visitor request into facets… present top 3–5… if none returned, relax the tightest facet
   once and retry").
2. **Served over HTTP** — `GET /api/assistant/capabilities` (`main.go:2817-2836`), rate-limited,
   with a public/authed split (`descriptor.go:62-99` — `IsPublic()`/`Public()` strip any
   write/RBAC-gated tool from the unauthenticated surface; today every declared tool is a public
   read, so the filter is a safety cap for later tools, not currently active).
3. **Discovered at runtime by the Python engine** — `edges.py:438-461` (`http_capability_source`)
   fetches that endpoint and parses it into a `CapabilityDocument`
   (`core.py:1048-1085`, mirrored Pydantic models).
4. **Wrapped as a callable tool** — `core.py:1201-1231` (`RegistryTool`): on call, it enforces the
   RBAC deny-floor, applies JSON-Schema defaults, validates input against the descriptor's schema,
   parses the descriptor's `endpoint` string into an HTTP method + path, and invokes an injected
   `ToolTransport`. The transport is real `httpx` (`edges.py:411-437`,
   `http_tool_transport`) — an actual outbound HTTP GET to Go's `/api/properties`.
5. **The real query** — `GET /api/properties` in `apps/api-chi/main.go` (search around line 925)
   parses filters via `propertyfilter.ParseListFilters` (`internal/propertyfilter/filter.go`, 240
   lines) and computes facets via `propertyfilter.ComputeFacets`/`BuildFacetInputs`
   (`internal/propertyfilter/facets.go`, 405 lines), which run against `pkg/property` (the shared
   core package, ~4,900 lines across the monorepo's `projects/sbx/packages/core/go/pkg/property`).
   This is the same catalog filter language the public `/properties` listing page and the admin
   property list use — not a separate, thinner "AI-only" query path.
6. **Real property output, not just IDs** — `core.py:1234-1272` (`offering_from_listing_item`) maps
   each `/api/properties` JSON row into a typed `Offering` (title, url → the live public
   `/p/{id}` route, price as `{amount, currency}`, cover image). `offerings_to_cards`/
   `offering_to_card` (`core.py:1304-1330`) turn those into `ChatCard` objects (title, subtitle,
   image, a "View listing" button) that ride the SSE stream to the browser (§4) and render as
   actual property cards in the chat UI, not just prose.
7. **Bounded so the model doesn't choke on it** — `TOOL_RESULT_MAX_ITEMS = 8`,
   `TOOL_RESULT_MAX_CHARS = 4000`, `OFFERING_SUMMARY_CHARS = 160` (`core.py:1239-1241`): the
   model gets a budgeted summary, the UI gets the full-fidelity cards — the same tool call feeds
   both, deliberately split (`core.py:1234-1237`, tagged "SC11").

**Verdict on Q1: confirmed, not guessed.** This is a real, generic (RegistryTool serves every
descriptor identically — it is not a hand-written "search properties" function) tool-calling path
from LLM function-call → HTTP → the same production property-filter engine the rest of BR's site
uses → typed results → rendered cards.

---

## 2. LLM provider(s) / SDK

- **Agent loop**: PydanticAI (`pydantic-ai-slim>=2`, `chat/pyproject.toml`) — `from pydantic_ai import
  Tool as _AgentTool` (`core.py:45`); `Toolbox` (`core.py:1486+`) builds the model-facing tool set
  from the `RegistryTool` instances.
- **Model routing / provider abstraction**: LiteLLM (`litellm>=1.89`) — `LiteLLMModel`
  (`edges.py:86-167`) implements PydanticAI's `Model` protocol on top of `litellm`;
  `build_router`/`LiteLLMRouterLLM` (`edges.py:322-393`) build a router from the admin-configured
  `ModelChoice` list (primary + fallbacks, in order).
- **Actual provider in production**: **OpenRouter**, via `openrouter_llm()`
  (`edges.py:395-409`) and the `OPENROUTER_API_KEY` secret injected at container start
  (`apps/assistant/secrets.tpl:5`, resolved from 1Password `op://…/br-openrouter/api-key`; also
  present in `.env.example:16`). So it is provider-agnostic by construction (LiteLLM can front
  Anthropic/OpenAI/DeepSeek/etc. directly) but BR specifically runs everything through the
  OpenRouter aggregator, model chosen by the admin's Models tab — a real deployed config runs
  `deepseek/deepseek-v4-flash` per an inline code comment
  (`docs/plans/2026-07-19-br-public-chat-battle-test-plan.md:39`), not a hardcoded OpenAI/Anthropic
  model.
- **Decision record `#0283`** (`.sbx/decisions/0283-ai-assistant-stack/decision.yml`) named a
  fuller target stack — LiteLLM **Proxy** (a separate gateway service for per-user virtual-key
  billing) + FastMCP (real MCP transport) + pgvector (RAG). **Only part of that is actually wired**:
  plain `litellm` (not the Proxy service) + PydanticAI are live; FastMCP is not used at all — the
  descriptor pattern is, verbatim from the code, "MCP-shape... MCP is the descriptor SHAPE, not
  transport" (`descriptor.go:35`); pgvector/embeddings are not wired — the actual "Grounding" tool
  (`StaticGrounding`, `core.py:991-1029`) is a plain in-memory keyword ranker over admin-configured
  URLs/documents (`_tokenize`/`_rank`, `core.py:949-969`), not vector search. Same "decided, mostly
  not wired" pattern the parent research doc's Part 3/`sbx-framework-inventory.md` already flagged
  elsewhere in this repo — worth naming here rather than assuming the decision record describes what
  ships.

---

## 3. Is it wired end-to-end, or scaffolding? — end-to-end, with one specific by-design limitation

Traced hop by hop and confirmed real (not stubbed) at every step:

`SvelteKit route (admin Chat tab or public widget)` → same-origin `/api/*` pass-through
(`hooks.server.ts`) → **Go handler** (`internal/handler/assistant_chat.go` for admin,
`assistant_public.go` for guests) → **`assistantchat.Client.Stream`** (raw SSE forward, no
buffering) → **Python FastAPI engine** `POST /chat` (`edges.py:820-865`) → PydanticAI `Agent.run`
with the `Toolbox` bound → on a tool call, `RegistryTool` → `http_tool_transport` → **Go**
`GET /api/properties` (or `/api/faq/search`) → `propertyfilter`/`pkg/property` → real DB-backed
results → back up through the same chain, cards attached to the SSE stream → Svelte
`chat-stream.ts` parses frames incrementally → `ChatBlock.svelte` (admin) /
`SiteChatWidget.svelte` (public, confirmed **mounted** in
`apps/web-svelte/src/routes/+layout.svelte:5,57`, i.e. present on every page including admin, not
just built-and-unmounted) render text + cards.

Two things worth being precise about rather than hand-waving as "it just works":

- **A stale doc comment**: `routes/(admin)/manage/ai-assistant/chat/+page.svelte:1-13` still says
  "drive the mock streamed reply," but `ChatBlock.svelte:5-11` (which that page renders) says
  explicitly "The reply is REAL — each send POSTs to the api-chi endpoint... We accumulate the
  streamed text." This is a small, self-correcting documentation drift (an older comment not
  updated after a real implementation landed), not evidence the feature is fake — the actual wire
  path (`STREAM_ENDPOINT = '/api/manage/ai-assistant/chat'`, `chat-stream.ts`) is real and matches
  the Go handler and Python engine described above.
- **The admin model picker is deliberately cosmetic, by design, not a bug**: a July-19 gap analysis
  (`docs/plans/2026-07-19-br-public-chat-battle-test-plan.md:44`, "F3: admin Chat-tab model picker
  is COSMETIC") flagged this as a finding, but the target architecture in the same doc
  (`§2`, "Model governance (PC-7) — enforced by construction: no model field exists at ANY hop") shows
  this was turned into an explicit security decision: the model is always `agents[default].models[0]`
  from the admin-saved config; no per-turn client field can override it, on any surface, so a guest
  (or a compromised admin browser tab) can never redirect spend to an expensive model. `ChatBlock.svelte`'s
  `selectedModelValue` only drives the UI subtitle.
- **The one genuine gap found earlier in the process (now closed)**: `docs/ai-assistant/01-current-state.md`
  (dated 2026-07-23) recorded drift item **D18**: "assistant engine absent from BR deploy compose
  while project.yml claims bundled." That gap is verifiably closed today —
  `clients/bestie/projects/bestierealestate/docker-compose.yml` (current) has a full `assistant:`
  service (port 5051, `ASSISTANT_LEAD_API_BASE`/`ASSISTANT_LISTING_API_BASE` pointing at api-chi,
  `depends_on: api-chi`), and git confirms the fix: commit `8b5fadf4f` "bring engine deploy-bundle
  forward to main — bootstrap emits internal `assistant` service" (2026-07-25), followed by the
  `bestierealestate-v0.1.57` production-generated-files commit (2026-07-26). So as of that date the
  engine ships as part of the real deploy, not just declared in config.

**Verdict on Q3: real end-to-end wiring**, with the caveats above being either stale comments,
intentional security-by-construction, or a documented-and-since-fixed deploy gap — not evidence of
an incomplete feature.

---

## 4. Chat transport: SSE, and streaming is real

- Engine: FastAPI `StreamingResponse` over `text/event-stream`
  (`edges.py:812,865`), an async generator (`event_stream`, `edges.py:822-864`) yielding
  `data: {...}\n\n` delta frames as the model produces tokens, a final `done:true` frame carrying
  `usage`/`cards`, and a reserved `event: error` frame for turns that delivered nothing.
- Go: `streamAssistantTurn` (`assistant_chat.go:125-161`) copies the engine's stream **byte-for-byte,
  unbuffered, flushing after every chunk** (`http.NewResponseController(w).Flush()`), with
  `X-Accel-Buffering: no` set explicitly to defeat proxy buffering — this is a real low-latency
  streaming proxy, not a buffer-then-send.
- Browser: `chat-stream.ts` (`apps/web-svelte/.../ai-assistant/chat-stream.ts`) is a
  hand-rolled, dependency-free SSE frame parser (buffers across chunk boundaries, splits on the
  blank-line delimiter, tolerates CRLF and a trailing unterminated frame) consumed via
  `ReadableStream`/`TextDecoder` (`decodeStream`, same file, lines 138-154) — a from-scratch fetch+SSE
  reader, not `EventSource` (needed because this is a POST with a body, which native `EventSource`
  can't do).

**Verdict on Q4: SSE, and streaming is genuinely implemented** at all three hops (engine emits
incrementally, Go forwards incrementally, browser renders incrementally) — not simulated with a
client-side typewriter effect over a single buffered response.

---

## 5. The public-chat session/auth pattern — a reusable pattern worth naming directly

`internal/assistantchat/token.go` + `internal/handler/assistant_public.go` implement a specific,
well-reasoned pattern for a public, unauthenticated chat widget that a Next.js implementation could
copy near-verbatim:

- The **session id is minted server-side** (`NewSessionID`, 128-bit random,
  `token.go:46-55`) — the browser never chooses it, so a guest can't reset their own quota by
  minting a new id or read another session's history by guessing one.
- The session id travels as an **HMAC-signed token** (`TokenSigner.Mint`/`Verify`,
  `token.go:59-97`, constant-time compare) inside an **HttpOnly cookie**
  (`br_chat_session`, `assistant_public.go:39`) — browser JS never touches it, so it can't be read or
  forged by injected script.
- **Fail-closed, not fail-open**, on missing config: an empty HMAC salt disables the signer entirely
  (`Enabled()`, `token.go:40`) — a misconfigured deploy makes public chat unavailable, never
  unprotected. The same instinct shows up in the rate limiter: `limiter pkgauth.RateLimiter // may be
  nil (Redis down) → fail-open on quota` (`assistant_public.go:51`) is the one deliberate exception,
  scoped to a *quota* limiter (an availability/cost concern) rather than the *auth* token (a
  security concern) — a real distinction between "who are you" and "how much can you use," each
  given its own failure posture.
- **Two independent abuse guards enforced before any LLM call is made** (`assistant_public.go:131-176`):
  a per-IP rolling-24h cap (the cookie-clear backstop, since a guest *can* mint a new session by
  clearing cookies) checked first, then a per-session quota — both are cheap `peek`/`record` calls
  against the existing shared rate-limiter, so an over-quota turn costs zero tokens.
- Config surfaced to the client is **explicitly minimal**: `Surface()` returns "welcome text, actions,
  and the quota the widget shows... but NEVER the model, costs, prompt, or sources"
  (`assistant_public.go:61-62`) — the system prompt, model choice, and knowledge sources never reach
  the browser even indirectly.

This is the one piece of this subsystem that is genuinely non-obvious and worth lifting as a named
pattern (server-minted-id + signed-cookie + fail-closed + dual quota-before-LLM-call), independent of
whether any other code gets reused.

---

## 6. Reusability for a Next.js + Vercel AI SDK assistant product

**The Go/Python backend is not Svelte-coupled.** The coupling to SvelteKit is thin and isolated:

- `hooks.server.ts`'s `/api/*` pass-through (sets `x-forwarded-for`, forwards cookies both ways,
  streams the body) is a same-origin reverse proxy to Go — a Next.js app would do the equivalent with
  a route handler / middleware, or simply call the Go API cross-origin if CORS allows it.
- The only genuinely Svelte-specific code is the **UI layer**: `ChatPanel`/`ChatWidget`/`ChatCard`
  (shared `@sbx/core-ui` Svelte 5 components), `ChatBlock.svelte`, `SiteChatWidget.svelte`, and the
  hand-rolled `chat-stream.ts` parser. None of these carry business logic — they are rendering +
  fetch-and-parse-SSE, both of which have direct, comparably-sized equivalents in React/Next.js
  (the Vercel AI SDK's `useChat`/`streamText` machinery does exactly the frame-parsing job
  `chat-stream.ts` hand-rolls, though the wire *shape* here — `{delta, done, usage, cards, notice}`
  bespoke JSON frames, not the AI SDK's own data-stream protocol — would need an adapter either at the
  Go layer or the Next.js edge).
- The **Go API surface the engine calls** (`/api/properties`, `/api/faq/search`,
  `/api/assistant/capabilities`) is already a generic, framework-agnostic HTTP+JSON API — a Next.js
  frontend can call it exactly as the Python engine does, with no Svelte dependency anywhere in that
  path.
- The **Python engine's own edge** (`POST /chat`, `{message, session_id, chat_id}` → SSE) is likewise
  framework-agnostic HTTP+SSE — nothing about it assumes a Svelte caller.

**What would actually need extracting/rebuilding, concretely:**

1. **Reusable near-as-is (the real engine)**: `sbx_assistant.chat` (core.py/edges.py/prompts.py) —
   the PydanticAI+LiteLLM agent loop, the `RegistryTool`/`Toolbox` generic descriptor executor, the
   SSE edge, the guardrails (input/output size caps, adaptive throttle, repetition guard), the
   Postgres config + conversation stores. This is already a standalone Python package with no BR or
   Svelte coupling — point it at any Go/Node API that serves the same `CapabilityDocument` shape at
   `GET /.../capabilities` and it works.
2. **Needs a thin new trust-boundary layer** if the goal is Next.js instead of Go/SvelteKit: something
   has to play api-chi's role of setting the *trusted* `chat_id` (admin vs. public) server-side and
   proxying the SSE stream — in Next.js this is a Route Handler, not a client-side call to the engine
   directly (the engine must stay unreachable from the browser, per the existing "single engine
   client" design decision this codebase deliberately made on 2026-07-19).
3. **Needs reimplementing in TypeScript**: the tool-*descriptor* list itself
   (`assistantcapabilities/descriptor.go`'s `Default()`) and whatever HTTP API backs `properties.search`
   — i.e., if the target product's property/domain data lives behind a different (e.g. Next.js API
   route or a different Go service) backend, the descriptors need re-declaring against that backend's
   endpoints; the *pattern* (one JSON-Schema-typed descriptor list, fetched by the agent at startup,
   executed generically) is what's worth porting, not the Go struct literals themselves.
4. **Needs redesigning for the Vercel AI SDK specifically**: the AI SDK expects tools defined with its
   own `tool({ description, inputSchema, execute })` shape and has its own streaming protocol
   (`streamText`/`toUIMessageStreamResponse`) — the *cleanest* port is to keep the descriptor-driven
   idea (tools are data, not hand-written functions) but implement the executor once in the AI SDK's
   shape rather than trying to reuse the Python `RegistryTool` bytes.
5. **UI**: rebuild `ChatPanel`/`ChatCard`/`ChatWidget` in React — the AI SDK UI's own chat components
   plus custom card rendering for tool results cover this ground natively and are arguably a better
   starting point than porting the Svelte components.

**Bottom line on Q6**: this is *not* "a Svelte app with an assistant baked in" — it is a
Python chat engine and a Go domain API, both already HTTP-native and framework-agnostic, currently
fronted by SvelteKit. A Next.js/Vercel-AI-SDK frontend could sit in front of the *same* Go domain API
(and, if desired, the same Python engine) via a new thin Next.js trust-boundary layer, without
needing to touch or understand any Svelte code. The genuinely reusable intellectual property is (a)
the descriptor-driven tool-registry pattern, (b) the guardrails/quota/session design in §5, and (c)
the property-filter query engine in Go (`pkg/property`/`internal/propertyfilter`) — none of which are
UI-framework-specific.

---

## 7. On "production deployed and run for a few months"

Verified from git history (dates are commit dates, not deploy dates, but BR's deploy pipeline tags a
release close to each merge):

- **2026-07-08**: `assistant-kit` capability descriptor pipeline lands (Phase 0+1).
- **2026-07-19**: the BR chat backbone, public-chat gate (signed token/quota/IP-cap), and FAQ tool +
  handoff→inquiry all land the same day (commits `f74e8a53e`, `7eb60c204`, `23a4ca6f8`, `006272483`).
  `docs/ai-assistant/01-current-state.md` (2026-07-23) still recorded the engine as **absent from the
  deploy compose** at that point (D18) — i.e., wired in code but not yet actually shipping.
- **2026-07-25/26**: the engine deploy bundle is brought into the real compose
  (`8b5fadf4f`) and shipped as `bestierealestate-v0.1.57` — this is the point the assistant became
  actually deployable end to end, not just code-complete.
- **2026-08-07**: `e04765057`/`36b4d9d8d` — the Usage tab is wired to real counters, and a fix
  changes the usage window to "roll 30 days + all-time totals" with the commit message noting **"a
  quiet month is not 'no activity'"** — phrasing that only makes sense if the assistant had already
  been live long enough to accumulate a low-traffic month of real usage data by early August.
  `bestierealestate-v0.1.65`/`v0.1.67` are both tagged 2026-08-07.
- **Last BR-specific commit found**: 2026-08-07 (the wanflo commit on 2026-08-24 that also touches
  the bestierealestate path appears to be an unrelated cross-client change, not BR feature work).

**Read on the claim**: "production deployed" is accurate from 2026-07-25/26 onward. As of today
(2026-09-24) that is **exactly two months**, and the last confirmed BR-specific activity was
2026-08-07 (about seven weeks ago) — so "a few months... run for a few months" is in the right
neighborhood and not fabricated, but is on the generous side of precise; "about two months, with the
last known activity roughly seven weeks back" is the more exact statement this repo actually
supports. "The most advanced system we have built" (among the AI-assistant attempts specifically) is
plausible and consistent with what was found — this is the only one of the AI-assistant efforts found
in this repo (see also `assistant-kit`'s own history, `sbx.framework`'s never-built "ProcessOS v0.1"
chat plan, and `whisper-python`'s unrelated speech-to-text service) that reached a real, tool-calling,
streaming, deployed state with usage telemetry — but this doc did not attempt a full comparison
against every other AI effort in shredbx, so "most advanced" itself is not independently re-verified
beyond "this one demonstrably works end to end, which is a high bar most half-built systems don't
clear."

---

## Sources referenced (file:line, verified by reading)

- `apps/api-chi/internal/assistantchat/{client.go,token.go}`
- `apps/api-chi/internal/aiassistant/aiassistant.go`
- `apps/api-chi/internal/assistantcapabilities/descriptor.go`
- `apps/api-chi/internal/handler/{assistant_chat.go,assistant_public.go}`
- `apps/api-chi/internal/propertyfilter/{filter.go,facets.go}`
- `apps/api-chi/main.go` (route registration, grepped + read around cited line numbers)
- `apps/web-svelte/src/routes/+layout.svelte`,
  `src/routes/(admin)/manage/ai-assistant/{chat/+page.svelte,...}`,
  `src/lib/components/admin/ai-assistant/{ChatBlock.svelte,chat-stream.ts}`,
  `src/lib/components/public/SiteChatWidget.svelte`
- `apps/assistant/{Dockerfile,entrypoint.sh,Makefile,secrets.tpl}`
- `docker-compose.yml` (current, and via git history)
- `.env.example`
- `projects/assistant-kit/packages/chat/python/{pyproject.toml,src/sbx_assistant/chat/{core.py,edges.py}}`
- `docs/ai-assistant/01-current-state.md` (2026-07-23 snapshot, esp. §10 and drift item D18)
- `docs/plans/2026-07-19-br-public-chat-battle-test-plan.md`
- `.sbx/decisions/0283-ai-assistant-stack/decision.yml`
- `git log`/`git show`/`git tag` on the above paths (commit dates cited in §7)
