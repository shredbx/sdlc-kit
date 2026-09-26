# Local LLM provider — design

## Objective

Add a third model tier to the `chat` agent alongside the two that already exist (`echo` for
no-key local dev, `openrouter:{model}` for cloud): a **local, in-process, no-additional-service**
model, so a consumer can run a real (if lightweight) LLM without an API key, without a network
call, and without standing up a separate model server.

## What this is not

- **Not Ollama.** Ollama is a real alternative (PydanticAI has a dedicated `OllamaProvider`), but
  it runs as its own background daemon — an additional service to install, run, and keep alive
  alongside the FastAPI process. That conflicts with "lightweight, no additional service," which
  was the explicit ask. Ollama stays a documented future option, not built now.
- **Not a declarative per-consumer config file.** FAQ, tools, session storage, and the model
  provider all already follow the same shape: the framework (`agent_framework`, sdlc-kit-owned)
  provides the swappable mechanism, and the consumer picks among them with a few lines of plain
  Python plus env vars (`CHAT_MODEL`, `OPENROUTER_API_KEY`, `DATABASE_URL`, `DEBUG_MODE`). This
  addition is one more branch in that same existing if/else, not a new configuration layer.
  Nothing today calls for the bigger config-file/resolver approach, so it isn't built.
- **Not a routing-only coordinator with a separate NLG model chained behind it.** That's a real
  pattern (small model routes, big model writes prose) but it's two model calls instead of one,
  and it isn't needed unless the single local model actually proves unreliable at tool-routing in
  real testing. Start with one model doing both jobs; only reach for the split if that fails.

## Runtime: llama-cpp-python, in-process

PydanticAI has no first-party llama.cpp model class — its own docs say: if a service is
OpenAI-API-compatible, use a custom provider; otherwise subclass `Model`, or use `FunctionModel`
for a lighter-weight custom implementation. `FunctionModel` is the exact primitive already used by
`debug_model.py` and the existing `echo` fallback in `chat/config.py` — this follows the same
pattern, wrapping `llama_cpp.Llama` calls instead of a regex parser or a static string.

`llama-cpp-python` runs the model directly inside the same Python process — no server, no port, no
daemon to keep alive. `Llama.from_pretrained(repo_id=..., filename=...)` downloads the GGUF file
from Hugging Face Hub into the HF cache on first use — this is the literal mechanism for "on
deploy it should download the model if the consumer picks it."

One real implementation gotcha: llama.cpp inference is synchronous, CPU-bound C++ work. Called
directly inside an `async def` model function, it would block FastAPI's event loop for the whole
generation. It must run via `asyncio.to_thread(...)`. The model itself loads once, at process
startup (module-level singleton — same pattern as `search_faq.py`'s `_store`), not per request.

## Model choice

**Start with `Qwen2.5-1.5B-Instruct` (GGUF, ~1-1.5GB at Q4_K_M/Q8_0)** — confirmed to exist as both
`Qwen/Qwen2.5-1.5B-Instruct-GGUF` and a `bartowski/Qwen2.5-1.5B-Instruct-GGUF` quantized mirror. The
Qwen2.5 family has a real reputation for tool-calling relative to its size, and `llama-cpp-python`
supports `chat_format="chatml-function-calling"` generically for ChatML-templated models — no
special "functionary" fine-tune required.

Two escalation paths if 1.5B's tool-routing proves unreliable in real testing, **not pre-decided,
to be picked based on actual results**:
- **`Qwen2.5-7B-Instruct-GGUF`** — llama.cpp's own function-calling docs specifically name this
  size as having proven native function-calling support. Heavier (~4-5GB quantized), same
  mechanism, just a bigger `LOCAL_MODEL_FILE`.
- **`google/functiongemma-270m-it`** (GGUF, confirmed real, <300MB) for tool-routing only, paired
  with the existing deterministic `to_reply()` formatters (`property_search_to_reply`,
  `handoff_to_reply`) for `search_properties`/`handoff` replies — those already turn a tool result
  into the final `AgentReply` + cards with zero LLM involved, proven in the debug-mode demo. Only
  genuinely freeform turns (FAQ phrasing, small talk, "something cheaper?" refinement) would still
  need a second, larger model call. Google's own docs are explicit that FunctionGemma "is not
  intended for use as a direct dialogue model" — it would only ever cover the routing half.

## Where it plugs in

`chat/config.py`'s existing model selection:

```python
model = FunctionModel(_echo_no_key_configured) if not os.environ.get("OPENROUTER_API_KEY") else resolve(model_name)
```

becomes a three-way branch keyed off a new `LLM_PROVIDER` env var (`local` / `openrouter` / unset):
unset still falls back to `echo` (so a bare `pytest`/CI run never tries to download a model);
`local` builds the new local model; `openrouter` keeps today's cloud path unchanged. A fresh
consumer's `.env.example` would ship `LLM_PROVIDER=local` pre-selected, since that's meant to be
the product's real default going forward, not just a fallback.

New framework module (mirrors `providers/openrouter.py`'s placement, since it's generic — any
consumer's agent could use it, not just bestays):

```
platform/python/frameworks/agent-framework/src/agent_framework/core/providers/
  openrouter.py         (existing)
  local_llama.py         (NEW — build_local_model(repo_id, filename) -> FunctionModel)
```

New env vars read by the consumer's `chat/config.py`: `LLM_PROVIDER`, `LOCAL_MODEL_REPO`
(default `Qwen/Qwen2.5-1.5B-Instruct-GGUF`), `LOCAL_MODEL_FILE` (default a `Q4_K_M` or `Q8_0`
glob) — same env-var-driven pattern as `CHAT_MODEL`/`OPENROUTER_API_KEY` today.

New dependencies on `agent-framework`'s own `pyproject.toml` (not the consumer's — the wrapper
module lives in the framework): `llama-cpp-python`, `huggingface-hub`.

## Testing

Unit tests mock the `llama_cpp.Llama` boundary (inject a fake object with the same
`create_chat_completion` shape) rather than actually loading a model in CI — same reasoning
`test_debug_model.py` already follows for its own `FunctionModel`-based tests: deterministic,
fast, no download, no real compute. Real model behavior (does 1.5B actually route reliably to the
three real tools) gets validated the same way the debug-mode work was: a real server, real curl
requests, real seeded data — not asserted from a pytest run.

## Explicitly out of scope here (separate, already-identified threads)

- Ollama as an alternative local provider.
- Embedding-model-backed FAQ retrieval (confirmed as a good future fit for the same in-process,
  no-additional-service pattern — `llama-cpp-python` supports a pure embedding mode — but
  `search_faq`'s own docstring already flags keyword-overlap as a stand-in for this; not this
  change).
- Search-refinement state, handoff's missing-field tracking, failure/escalation clarification —
  the three real gaps found when comparing the current build against the real-estate chatbot tech
  task. Separate design pass.
- Session identity (signed cookie mechanism) — paused mid-design in an earlier part of this
  session, unrelated to this change.
