# PydanticAI: session/state management patterns and anti-patterns

Last updated: 2026-09-26
tags: pydantic-ai, agent-framework, session, memory, research

Grounded against the installed `ai:building-pydantic-ai-agents` skill's
`references/INPUT-AND-HISTORY.md` (read directly) plus first-party docs at `pydantic.dev/docs/ai/`
(fetched directly) and secondary tutorial/community sources (flagged as such, cross-checked where a
claim could be verified). Written for `platform/python/frameworks/agent-framework`'s own
`Session`/`SessionStore` design — generic to the framework, no client-specific detail (per this
repo's own `feedback_no-client-info-in-sdlc-kit` rule).

**Every finding below is evaluated against agent-framework's actual current shape, not repeated at
face value.** Current shape, for reference: `Session{id, user_id, messages, data}`
(`core/store/base.py`), `MemoryStore`/`PostgresStore` adapters, session id resolved from an
unauthenticated client-supplied `X-Session-Id` header (or a random UUID if absent), one shared
`Agent` instance built once at app startup (`agents/<name>/config.py`), tools as plain functions
with no `RunContext` parameter today.

## What's already right — confirmed against first-party sources, not assumed

**One shared `Agent` instance + stateless tools + explicit history load/save per request is the
documented-correct shape**, not something to "fix." A community FastAPI-integration write-up
(theneuralbase.com, secondary source) frames this explicitly as "Pattern 2" against the
anti-pattern of a request-scoped or agent-held-state design: build one `Agent` at app scope, keep
its tools free of stored state, load history by conversation id, pass it as `message_history=`,
write back only `new_messages()`. That is exactly `agent_framework/server/routes/chat.py`'s current
shape. One specific claim from that same source — that vanilla FastAPI's `Depends()` accepts a
`scope='request'` keyword — does **not** match plain FastAPI's actual signature and could not be
confirmed against a first-party source; treat it skeptically. It doesn't matter for us anyway: our
session/store resolution is already request-scoped structurally (`session_dependency(store)` calls
`store.get_or_create(session_id)` fresh every request), not via that mechanism.

**`conversation_id`, appending only `new_messages()`, is the framework's own recommended shape** —
confirmed directly in the skill reference and `pydantic.dev/docs/ai/core-concepts/message-history/`:
"load a thread's history, pass it as `message_history`, and write back `new_messages()` once the run
finishes. Appending each run's new messages rather than rewriting the full list keeps each write
proportional to the turn instead of to the conversation." Our own `PostgresStore.save()` currently
rewrites the *whole* `messages` array as one JSONB blob every turn — functionally correct at our
current scale (no real users yet), but worth knowing it diverges from the framework's own
recommended incremental-write shape if/when message history grows large enough for that to matter.
Not a fix to make now — a fact to know before assuming our current shape is the documented one.

**`StepPersistence` (a first-party capability, `pydantic.dev/docs/ai/core-concepts/storage/`) is
real, and — importantly — its own docs say we don't need it yet.** It adds an append-only event
log, resumable/forkable snapshots, and a tool-effect ledger on top of message history — for
recovering an agent mid-multi-step-task after a crash. Direct quote: if you just need to save and
restore a single chat thread, reaching for it is overkill — "a `jsonb` column is enough for that."
That is precisely what `PostgresStore` already is. Correctly scoped for where we are; revisit only
if we start building genuinely resumable multi-step agent workflows, not for ordinary chat
continuity.

## The real anti-pattern that applies directly to us: mutable deps under concurrent tool calls

A live, first-party GitHub issue (`pydantic/pydantic-ai#4322`, closed as "not planned" — meaning no
official framework-level fix exists) demonstrates: **mutating `AgentDepsT` (whatever object your
tools receive via `RunContext.deps`) from inside a tool call is unsafe when the agent makes parallel
tool calls in the same turn** — a read-pause-write race loses updates. The maintainers' own response
offered no built-in guard; community suggestions were frozen (`@dataclass(frozen=True)`) deps or
manual locking, neither built in.

**Why this matters for us specifically, not generically:** the moment any tool gains a `ctx:
RunContext[ChatDeps]` parameter and writes into `session.data` (the natural way to accumulate
facts across turns — see below), and the agent is ever allowed to call two such tools in the same
turn, we inherit exactly this race. Two mitigations, both cheap at our scale: (a) don't give the
model multiple state-writing tools it could plausibly call in parallel in one turn — keep
fact-writing tools singular and narrow; (b) if that's ever not enough, treat `session.data` writes
as read-modify-write against the store's `save()`, not as in-memory dict mutation shared across
concurrent calls. Not an issue today (nothing writes to `session.data` yet), but real for the very
next thing being designed.

## The other real anti-pattern: an unauthenticated session id is not an identity boundary

PydanticAI's own first-party `Memory` capability (`pydantic.dev/docs/ai/harness/memory/`) — a
per-user/per-tenant fact store, namespaced so one agent instance can serve many users without
cross-contamination — states its own limitation plainly: "namespace isolation... is not a
cryptographic security boundary," and memory is "model-written, untrusted content." The namespace
resolver assumes *your application* already knows who's who; the capability only organizes storage
once you do. That assumption doesn't hold for us today: `X-Session-Id` is whatever the client sends,
unverified. Adopting any per-user memory/namespace mechanism — this capability or a hand-rolled
equivalent — before that's addressed just gives an attacker-controlled session id a clean,
well-organized place to plant or read data. This is the one finding in this doc that's a genuine
blocker, not just a scoping note: **collecting real contact PII behind an unauthenticated session id
is a real gap to close (or explicitly, consciously accept for now) before wiring that up for real
users, independent of which storage mechanism is chosen.**

`Memory` itself is otherwise well-matched to "accumulate facts across turns for the model to use
later" — but it's designed for a live model to read/write it (`inject_memory` re-injects prior
entries into the prompt). Reaching for it now, with no model in the loop yet, is solving a problem we
don't have yet while skipping past the auth gap that actually blocks it. The right-sized move for the
current AI-free milestone is a plain, explicit, deterministic mechanism — a tool that writes
structured fields into `Session.data` (already an unused field on the existing dataclass, no schema
change needed) and another that reads them back and reports what's still missing — with `Memory`
as the natural upgrade path once a real model and a real identity mechanism both exist.

## Slot-filling, generically (non-PydanticAI-specific, but directly applicable)

Established conversational-agent design practice — corroborated by an arXiv survey on goal-oriented
dialogue systems and by Rasa's own (a mature, real open-source framework) documented "slot" concept
— converges on one point worth stating plainly because it's easy to skip under a "the model is smart
enough" assumption: **LLMs alone cannot reliably manage multi-turn slot-filling state; wrap
extraction in deterministic slot-state tracking (an explicit state class/dict), not implicit LLM
reasoning about what's already been collected.** Concretely: don't trust the model to "remember" it
already has a name from three turns ago and silently not ask again — that's exactly the class of bug
that produces silent data loss or a repeated question that annoys a real user. Track what's filled
and what's missing as data, in the store, and hand the model that state explicitly each turn (or let
a deterministic tool report it) rather than hoping message-history re-reading gets it right.

## Bottom line for the next piece of design work

1. `Session.data` (already present, unused) is the right place for accumulated structured facts —
   no schema change needed to start.
2. Whatever writes to it should be a narrow, explicit, single-purpose tool (or the direct-invocation
   debug path) — not free-text extraction trusted to a future LLM.
3. Track "what's still missing" as data (a computed list against a known required-field set), not as
   something the model is expected to infer from re-reading history.
4. Treat the unauthenticated-session-id-as-identity gap as a real, named tradeoff to accept or close
   *before* real contact PII starts flowing through it — not something either storage pattern above
   papers over.
5. `StepPersistence` and `Memory` are both real, well-designed, first-party mechanisms — and both
   are for a later milestone (resumable multi-step workflows; live-model-facing cross-turn memory),
   not this one.
